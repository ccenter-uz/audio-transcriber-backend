package audiochunk

import (
	"math"
	"reflect"
	"testing"
)

func TestPad(t *testing.T) {
	opt := DefaultOptions
	got := pad([]Chunk{{0.1, 5}, {5.3, 25.3}, {29.9, 30}}, 30, opt)
	want := []Chunk{{0, 5.2}, {5.2, 25.2}, {29.7, 30}}
	for i := range want {
		if math.Abs(got[i].Start-want[i].Start) > 1e-9 || math.Abs(got[i].End-want[i].End) > 1e-9 {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestParseSilences(t *testing.T) {
	output := `[silencedetect @ 0x1] silence_start: -0.01
[silencedetect @ 0x1] silence_end: 2.97 | silence_duration: 2.98
size=N/A time=00:00:05.00
[silencedetect @ 0x1] silence_start: 4.5
[silencedetect @ 0x1] silence_end: 5 | silence_duration: 0.5
[silencedetect @ 0x1] silence_start: 9.25`

	got := parseSilences(output)
	want := []Chunk{{0, 2.97}, {4.5, 5}, {9.25, -1}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestSplit(t *testing.T) {
	opt := DefaultOptions
	opt.Padding = 0

	tests := []struct {
		name     string
		total    float64
		silences []Chunk
		want     []Chunk
	}{
		{
			name:  "no silence, short audio",
			total: 8,
			want:  []Chunk{{0, 8}},
		},
		{
			name:  "no silence, long audio is cut into equal parts",
			total: 50,
			want:  []Chunk{{0, 50.0 / 3}, {50.0 / 3, 100.0 / 3}, {100.0 / 3, 50}},
		},
		{
			name:     "short pauses are merged, leading and trailing silence dropped",
			total:    12,
			silences: []Chunk{{0, 1}, {3, 3.5}, {6, 6.4}, {10, -1}},
			want:     []Chunk{{1, 10}},
		},
		{
			name:     "long pause is not merged",
			total:    18,
			silences: []Chunk{{2, 15}},
			want:     []Chunk{{0, 2}, {15, 18}},
		},
		{
			name:     "merge stops at max duration",
			total:    30,
			silences: []Chunk{{9, 9.5}, {18, 18.5}, {25, 25.5}},
			want:     []Chunk{{0, 18}, {18.5, 30}},
		},
		{
			name:     "speech shorter than min duration is dropped",
			total:    10,
			silences: []Chunk{{0, 4}, {4.5, 10}},
			want:     nil,
		},
		{
			name:     "only silence",
			total:    5,
			silences: []Chunk{{0, -1}},
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := split(tt.total, tt.silences, opt)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if math.Abs(got[i].Start-tt.want[i].Start) > 1e-9 || math.Abs(got[i].End-tt.want[i].End) > 1e-9 {
					t.Fatalf("got %v, want %v", got, tt.want)
				}
				if got[i].Duration() > opt.MaxDuration+1e-9 {
					t.Fatalf("chunk %v is longer than %v", got[i], opt.MaxDuration)
				}
			}
		})
	}
}
