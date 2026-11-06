// Package audiochunk splits an audio file into speech chunks using ffmpeg.
package audiochunk

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Options -.
type Options struct {
	MinDuration     float64 // chunks shorter than this are dropped
	MaxDuration     float64 // chunks are never longer than this
	MaxMergeGap     float64 // speech parts separated by a longer pause are not merged
	Padding         float64 // extra audio kept around speech so word edges are not cut
	SilenceNoise    string  // silencedetect noise threshold, e.g. "-35dB"
	SilenceDuration float64 // minimal pause length treated as silence
}

// DefaultOptions matches the old vad-chunk service settings (min 1s, max 20s).
var DefaultOptions = Options{
	MinDuration:     1,
	MaxDuration:     20,
	MaxMergeGap:     1,
	Padding:         0.2,
	SilenceNoise:    "-35dB",
	SilenceDuration: 0.3,
}

// MixedChannels marks a chunk taken from all channels mixed down to mono.
const MixedChannels = -1

// Chunk is a part of the source audio, in seconds.
type Chunk struct {
	Start   float64
	End     float64
	Channel int // source channel index (0 - left, 1 - right) or MixedChannels
}

func (c Chunk) Duration() float64 {
	return c.End - c.Start
}

// Detect returns speech chunks of the audio file sorted by start time.
// Each channel of a stereo file (e.g. operator and client) is chunked separately;
// mono files and stereo files with identical channels are chunked as one mixed channel.
func Detect(ctx context.Context, audioPath string, opt Options) ([]Chunk, error) {
	total, channels, err := probe(ctx, audioPath)
	if err != nil {
		return nil, err
	}

	if channels == 2 {
		dualMono, err := isDualMono(ctx, audioPath)
		if err != nil {
			return nil, err
		}
		if dualMono {
			channels = 1
		}
	}

	if channels < 2 {
		silences, err := detectSilences(ctx, audioPath, MixedChannels, opt)
		if err != nil {
			return nil, err
		}
		chunks := split(total, silences, opt)
		for i := range chunks {
			chunks[i].Channel = MixedChannels
		}
		return chunks, nil
	}

	var chunks []Chunk
	for ch := 0; ch < channels; ch++ {
		silences, err := detectSilences(ctx, audioPath, ch, opt)
		if err != nil {
			return nil, err
		}
		for _, c := range split(total, silences, opt) {
			c.Channel = ch
			chunks = append(chunks, c)
		}
	}
	sort.SliceStable(chunks, func(i, j int) bool { return chunks[i].Start < chunks[j].Start })
	return chunks, nil
}

// Extract writes the chunk to dstPath as mono 16-bit PCM wav.
func Extract(ctx context.Context, audioPath, dstPath string, chunk Chunk) error {
	args := []string{"-hide_banner", "-loglevel", "error", "-y",
		"-i", audioPath,
		"-ss", formatSeconds(chunk.Start),
		"-t", formatSeconds(chunk.Duration()),
	}
	if chunk.Channel == MixedChannels {
		args = append(args, "-ac", "1")
	} else {
		args = append(args, "-af", channelFilter(chunk.Channel))
	}
	args = append(args, "-c:a", "pcm_s16le", dstPath)

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg extract: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// probe returns the duration in seconds and the channel count of the first audio stream.
func probe(ctx context.Context, audioPath string) (float64, int, error) {
	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "error",
		"-select_streams", "a:0",
		"-show_entries", "stream=channels:format=duration",
		"-of", "default=noprint_wrappers=1",
		audioPath,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("ffprobe: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	var duration float64
	var channels int
	for _, line := range strings.Split(string(out), "\n") {
		key, value, _ := strings.Cut(strings.TrimSpace(line), "=")
		switch key {
		case "duration":
			duration, _ = strconv.ParseFloat(value, 64)
		case "channels":
			channels, _ = strconv.Atoi(value)
		}
	}
	if duration <= 0 || channels <= 0 {
		return 0, 0, fmt.Errorf("ffprobe: invalid audio info %q", strings.TrimSpace(string(out)))
	}
	return duration, channels, nil
}

var reMaxVolume = regexp.MustCompile(`max_volume: (-?[\d.]+|-inf) dB`)

// isDualMono reports whether both channels of a stereo file carry the same signal.
func isDualMono(ctx context.Context, audioPath string) (bool, error) {
	cmd := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-nostats",
		"-i", audioPath, "-af", "pan=mono|c0=c0-c1,volumedetect", "-f", "null", "-",
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("ffmpeg volumedetect: %w: %s", err, lastLine(stderr.String()))
	}

	m := reMaxVolume.FindStringSubmatch(stderr.String())
	if m == nil {
		return false, fmt.Errorf("ffmpeg volumedetect: max_volume not found")
	}
	if m[1] == "-inf" {
		return true, nil
	}
	diff, _ := strconv.ParseFloat(m[1], 64)
	return diff < -60, nil
}

func channelFilter(channel int) string {
	return fmt.Sprintf("pan=mono|c0=c%d", channel)
}

var (
	reSilenceStart = regexp.MustCompile(`silence_start: (-?[\d.]+)`)
	reSilenceEnd   = regexp.MustCompile(`silence_end: (-?[\d.]+)`)
)

func detectSilences(ctx context.Context, audioPath string, channel int, opt Options) ([]Chunk, error) {
	filter := fmt.Sprintf("silencedetect=noise=%s:d=%s", opt.SilenceNoise, formatSeconds(opt.SilenceDuration))
	if channel != MixedChannels {
		filter = channelFilter(channel) + "," + filter
	}
	cmd := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-nostats",
		"-i", audioPath, "-af", filter, "-f", "null", "-",
	)
	// silencedetect writes its results to stderr
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg silencedetect: %w: %s", err, lastLine(stderr.String()))
	}

	return parseSilences(stderr.String()), nil
}

func parseSilences(output string) []Chunk {
	var silences []Chunk
	start := -1.0
	for _, line := range strings.Split(output, "\n") {
		if m := reSilenceStart.FindStringSubmatch(line); m != nil {
			start, _ = strconv.ParseFloat(m[1], 64)
			start = max(start, 0)
		} else if m := reSilenceEnd.FindStringSubmatch(line); m != nil && start >= 0 {
			end, _ := strconv.ParseFloat(m[1], 64)
			silences = append(silences, Chunk{Start: start, End: end})
			start = -1
		}
	}
	// silence lasting till the end of the file has no silence_end
	if start >= 0 {
		silences = append(silences, Chunk{Start: start, End: -1})
	}
	return silences
}

// split turns silences into speech chunks between MinDuration and MaxDuration.
// Neighbouring speech parts are merged while they fit into MaxDuration
// and the pause between them is not longer than MaxMergeGap; parts longer than MaxDuration are cut into equal pieces.
func split(total float64, silences []Chunk, opt Options) []Chunk {
	var speech []Chunk
	pos := 0.0
	for _, s := range silences {
		end := s.End
		if end < 0 || end > total {
			end = total
		}
		if s.Start > pos {
			speech = append(speech, Chunk{Start: pos, End: s.Start})
		}
		pos = max(pos, end)
	}
	if pos < total {
		speech = append(speech, Chunk{Start: pos, End: total})
	}

	var chunks []Chunk
	flush := func(c Chunk) {
		if c.Duration() <= opt.MaxDuration {
			if c.Duration() >= opt.MinDuration {
				chunks = append(chunks, c)
			}
			return
		}
		parts := int(c.Duration()/opt.MaxDuration) + 1
		step := c.Duration() / float64(parts)
		for i := 0; i < parts; i++ {
			chunks = append(chunks, Chunk{Start: c.Start + step*float64(i), End: c.Start + step*float64(i+1)})
		}
	}

	if len(speech) == 0 {
		return nil
	}
	cur := speech[0]
	for _, s := range speech[1:] {
		if s.End-cur.Start <= opt.MaxDuration && s.Start-cur.End <= opt.MaxMergeGap {
			cur.End = s.End
			continue
		}
		flush(cur)
		cur = s
	}
	flush(cur)

	return pad(chunks, total, opt)
}

// pad widens chunks by Padding without overlapping neighbours or exceeding MaxDuration.
func pad(chunks []Chunk, total float64, opt Options) []Chunk {
	prevEnd := 0.0
	for i := range chunks {
		c := &chunks[i]
		c.Start = max(c.Start-opt.Padding, prevEnd)
		c.End = min(c.End+opt.Padding, total, c.Start+opt.MaxDuration)
		if i+1 < len(chunks) {
			c.End = min(c.End, chunks[i+1].Start)
		}
		prevEnd = c.End
	}
	return chunks
}

func formatSeconds(s float64) string {
	return strconv.FormatFloat(s, 'f', 3, 64)
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return lines[len(lines)-1]
}
