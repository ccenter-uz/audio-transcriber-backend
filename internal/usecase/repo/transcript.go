package repo

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mirjalilova/voice_transcribe/config"
	"github.com/mirjalilova/voice_transcribe/internal/entity"
	"github.com/mirjalilova/voice_transcribe/pkg/logger"
	"github.com/mirjalilova/voice_transcribe/pkg/postgres"
)

type TranscriptRepo struct {
	pg     *postgres.Postgres
	config *config.Config
	logger *logger.Logger
}

// New -.
func NewTranscriptRepo(pg *postgres.Postgres, config *config.Config, logger *logger.Logger) *TranscriptRepo {
	return &TranscriptRepo{
		pg:     pg,
		config: config,
		logger: logger,
	}
}

func (r *TranscriptRepo) Create(ctx context.Context, req *entity.CreateTranscript) error {
	query := `
	INSERT INTO transcripts (segment_id, user_id, ai_text, transcribe_text, report_text)
	VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.pg.Pool.Exec(ctx, query, req.SegmentId, req.AIText)
	if err != nil {
		return fmt.Errorf("failed to create transcript: %w", err)
	}

	return nil
}

func (r *TranscriptRepo) Update(ctx context.Context, req *entity.UpdateTranscript) error {
	query := `
	UPDATE
		transcripts
	SET`

	var conditions []string
	var args []interface{}

	if req.TranscriptText != "" && req.TranscriptText != "string" {
		conditions = append(conditions, " transcribe_text = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.TranscriptText)
	}
	if req.ReportText != "" && req.ReportText != "string" {
		conditions = append(conditions, " report_text = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.ReportText)
	}

	if len(conditions) == 0 {
		return errors.New("nothing to update")
	}

	conditions = append(conditions, " updated_at = now()")
	query += strings.Join(conditions, ", ")
	query += " WHERE id = $" + strconv.Itoa(len(args)+1) + " AND deleted_at = 0"
	args = append(args, req.Id)

	_, err := r.pg.Pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	return nil
}

func (r *TranscriptRepo) UpdateStatus(ctx context.Context, id *int) error {
	query := `
	UPDATE
		transcripts
	SET
		status = 'done',
		updated_at = now()
	WHERE
		id = $1 AND deleted_at = 0`

	_, err := r.pg.Pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	return nil
}

func (r *TranscriptRepo) GetById(ctx context.Context, id int) (*entity.Transcript, error) {
	var createdAt time.Time

	query := `
	SELECT
		t.id,
		s.audio_id,
		a.filename,
		t.segment_id,
		t.user_id,
		u.username,
		t.ai_text,
		COALESCE(NULLIF(t.transcribe_text, ''), 'no_text') AS transcribe_text,
		COALESCE(NULLIF(t.report_text, ''), 'no_text') AS report_text,
		t.status,
		t.created_at
	FROM transcripts t
	JOIN users u ON t.user_id = u.id
	JOIN audio_file_segments s ON t.segment_id = s.id
	JOIN audio_files a ON s.audio_id = a.id
	WHERE t.id = $1 AND t.deleted_at = 0 AND s.deleted_at = 0
	`
	transcript := &entity.Transcript{}
	err := r.pg.Pool.QueryRow(ctx, query, id).Scan(
		&transcript.Id,
		&transcript.AudioId,
		&transcript.AudioName,
		&transcript.SegmentId,
		&transcript.UserId,
		&transcript.Username,
		&transcript.AIText,
		&transcript.TranscriptText,
		&transcript.ReportText,
		&transcript.Status,
		&createdAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get transcripts: %w", err)
	}

	transcript.CreatedAt = createdAt.Format("2006-01-02 15:04:05")

	return transcript, nil
}

func (r *TranscriptRepo) Delete(ctx context.Context, id int) error {
	query := `
	UPDATE transcripts
	SET deleted_at = EXTRACT(EPOCH FROM NOW())
	WHERE id = $1 AND deleted_at = 0
	`
	_, err := r.pg.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete transcript: %w", err)
	}

	return nil
}

func (r *TranscriptRepo) GetList(ctx context.Context, req *entity.GetTranscriptReq) (*entity.TranscriptList, error) {
	query := `
	SELECT 
		COUNT(t.id) OVER () AS total_count,
		t.id, 
		s.audio_id,
		a.filename,
		t.segment_id, 
		t.user_id, 
		u.username,
		t.ai_text,
		COALESCE(NULLIF(t.transcribe_text, ''), 'no_text') AS transcribe_text,
		COALESCE(NULLIF(t.report_text, ''), 'no_text') AS report_text,
		t.status,
		t.created_at
	FROM transcripts t
	JOIN users u ON t.user_id = u.id
	JOIN audio_file_segments s ON t.segment_id = s.id
	JOIN audio_files a ON s.audio_id = a.id
	WHERE t.deleted_at = 0 AND s.deleted_at = 0
	`

	var conditions []string
	var args []interface{}

	if req.AudioId != "" {
		conditions = append(conditions, "s.audio_id = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.AudioId)
	}

	if req.Status != "" {
		conditions = append(conditions, "t.status = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.Status)
	}

	if req.UserId != "" {
		conditions = append(conditions, "t.user_id = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.UserId)
	}

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	query += ` ORDER BY t.created_at DESC OFFSET $` + strconv.Itoa(len(args)+1) + ` LIMIT $` + strconv.Itoa(len(args)+2)

	args = append(args, req.Filter.Offset, req.Filter.Limit)

	rows, err := r.pg.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get transcript list: %w", err)
	}
	defer rows.Close()

	transcripts := entity.TranscriptList{}
	for rows.Next() {
		var createdAt time.Time
		var count int
		transcript := entity.Transcript{}
		err := rows.Scan(
			&count,
			&transcript.Id,
			&transcript.AudioId,
			&transcript.AudioName,
			&transcript.SegmentId,
			&transcript.UserId,
			&transcript.Username,
			&transcript.AIText,
			&transcript.TranscriptText,
			&transcript.ReportText,
			&transcript.Status,
			&createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transcript: %w", err)
		}
		transcript.CreatedAt = createdAt.Format("2006-01-02 15:04:05")
		transcripts.Transcripts = append(transcripts.Transcripts, transcript)
		transcripts.Count = count
	}

	return &transcripts, nil
}
