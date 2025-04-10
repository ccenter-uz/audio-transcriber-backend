package repo

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/mirjalilova/voice_transcribe/config"
	"github.com/mirjalilova/voice_transcribe/internal/entity"
	"github.com/mirjalilova/voice_transcribe/pkg/logger"
	"github.com/mirjalilova/voice_transcribe/pkg/postgres"
)

type AudioSegmentRepo struct {
	pg     *postgres.Postgres
	config *config.Config
	logger *logger.Logger
}

// New -.
func NewAudioSegmentRepo(pg *postgres.Postgres, config *config.Config, logger *logger.Logger) *AudioSegmentRepo {
	return &AudioSegmentRepo{
		pg:     pg,
		config: config,
		logger: logger,
	}
}

func (r *AudioSegmentRepo) GetById(ctx context.Context, id int) (*entity.AudioSegment, error) {
	var createdAt time.Time

	query := `
	SELECT 
		s.id,
		s.audio_id,
		a.filename,
		s.status,
		s.created_at
	FROM audio_file_segments s
	JOIN audio_files a ON s.audio_id = a.id
	WHERE s.id = $1 AND a.deleted_at = 0 AND s.deleted_at = 0
	`
	segment := &entity.AudioSegment{}
	err := r.pg.Pool.QueryRow(ctx, query, id).Scan(
		&segment.Id,
		&segment.AudioId,
		&segment.AudioName,
		&segment.Status,
		&createdAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get segment: %w", err)
	}

	segment.CreatedAt = createdAt.Format("2006-01-02 15:04:05")

	return segment, nil
}

func (r *AudioSegmentRepo) GetList(ctx context.Context, req *entity.GetAudioSegmentReq) (*entity.AudioSegmentList, error) {
	query := `
	SELECT 
	COUNT(s.id) OVER () AS total_count,
	s.id,
	s.audio_id,
	a.filename,
	s.status,
	s.created_at
	FROM audio_file_segments s
	JOIN audio_files a ON s.audio_id = a.id
	WHERE a.deleted_at = 0 AND s.deleted_at = 0
	`

	var conditions []string
	var args []interface{}

	if req.AudioId != "" {
		conditions = append(conditions, "s.audio_id = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.AudioId)
	}

	if req.Status != "" {
		conditions = append(conditions, "s.status = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.Status)
	}

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	query += ` ORDER BY s.created_at DESC OFFSET $` + strconv.Itoa(len(args)+1) + ` LIMIT $` + strconv.Itoa(len(args)+2)

	args = append(args, req.Filter.Offset, req.Filter.Limit)

	rows, err := r.pg.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get segment list: %w", err)
	}
	defer rows.Close()

	audioSegments := entity.AudioSegmentList{}
	for rows.Next() {
		var createdAt time.Time
		var count int
		transcript := entity.AudioSegment{}
		err := rows.Scan(
			&count,
			&transcript.Id,
			&transcript.AudioId,
			&transcript.AudioName,
			&transcript.Status,
			&createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan segment: %w", err)
		}
		transcript.CreatedAt = createdAt.Format("2006-01-02 15:04:05")
		audioSegments.AudioSegments = append(audioSegments.AudioSegments, transcript)
		audioSegments.Count = count
	}

	return &audioSegments, nil
}

func (r *AudioSegmentRepo) Delete(ctx context.Context, id int) error {
	query := `
		UPDATE audio_file_segments
		SET deleted_at = EXTRACT(EPOCH FROM NOW())
		WHERE id = $1 AND deleted_at = 0
		`
	_, err := r.pg.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete segment: %w", err)
	}

	return nil
}

func (r *AudioSegmentRepo) GetTranscriptPercent(ctx context.Context) (*[]entity.TranscriptPersent, error) {
	query := `SELECT * FROM calculate_transcription_percentage()`

	rows, err := r.pg.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := []entity.TranscriptPersent{}
	for rows.Next() {
		reps := entity.TranscriptPersent{}
		err := rows.Scan(
			&reps.AudioFileId,
			&reps.Filename,
			&reps.TotalSegments,
			&reps.CompletedSegments,
			&reps.Percent,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan calculate_transcription_percentage: %w", err)
		}

		reps.Percent = math.Round(reps.Percent*100) / 100
		res = append(res, reps)
	}

	return &res, nil
}
