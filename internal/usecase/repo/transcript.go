package repo

import (
	"context"
	"fmt"
	"log/slog"
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
	SET
		status = 'done',
		`

	var conditions []string
	var args []interface{}

	tr, err := r.pg.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	conditions = append(conditions, " user_id = $"+strconv.Itoa(len(args)+1))
	args = append(args, req.UserID)

	if req.TranscriptText != "" && req.TranscriptText != "string" {
		conditions = append(conditions, " transcribe_text = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.TranscriptText)
	}
	if req.ReportText != "" && req.ReportText != "string" {
		conditions = append(conditions, " report_text = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.ReportText)
	}

	if len(conditions) == 0 {
		slog.Warn("nothing to update")
		return nil
	}

	conditions = append(conditions, " updated_at = now()")
	query += strings.Join(conditions, ", ")
	query += " WHERE segment_id = $" + strconv.Itoa(len(args)+1) + " AND deleted_at = 0"
	args = append(args, req.Id)

	_, err = tr.Exec(ctx, query, args...)
	if err != nil {
		tr.Rollback(ctx)
		return fmt.Errorf("failed to update transcript: %w", err)
	}

	if req.ReportText != "" && req.ReportText != "string" {
		if req.EntireAudioInvalid {
			query1 := `
					WITH target_audio_file AS (
						SELECT audio_id
						FROM audio_file_segments
						WHERE id = $1
					),
					segments_of_audio AS (
						SELECT id
						FROM audio_file_segments
						WHERE audio_id = (SELECT audio_id FROM target_audio_file)
					)
					UPDATE transcripts
					SET status = 'invalid',
						updated_at = NOW()
					WHERE segment_id IN (SELECT id FROM segments_of_audio)
				`

				_, err := tr.Exec(ctx, query1, req.Id)
				if err != nil {
					tr.Rollback(ctx)
					return fmt.Errorf("failed to update transcript status: %w", err)
				}

				query2 := `
					UPDATE audio_files
					SET status = 'error',
						updated_at = NOW()
					WHERE id = (
						SELECT audio_id
						FROM audio_file_segments
						WHERE id = $1
					)
				`

				_, err = tr.Exec(ctx, query2, req.Id)
				if err != nil {
					tr.Rollback(ctx)
					return fmt.Errorf("failed to update audio file status: %w", err)
				}
		}

		query = `UPDATE transcripts SET status = 'invalid', updated_at = now() WHERE segment_id = $1 AND deleted_at = 0`
		_, err = tr.Exec(ctx, query, req.Id)
		if err != nil {
			tr.Rollback(ctx)
			return fmt.Errorf("failed to update transcript status: %w", err)
		}

	}

	if err := tr.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// func (r *TranscriptRepo) UpdateStatus(ctx context.Context, id *int, user_id string) error {
// 	query := `
// 	UPDATE
// 		transcripts
// 	SET
// 		status = 'done',
// 		user_id = $2,
// 		updated_at = now()
// 	WHERE
// 		segment_id = $1 AND deleted_at = 0`

// 	_, err := r.pg.Pool.Exec(ctx, query, id, user_id)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

func (r *TranscriptRepo) GetById(ctx context.Context, id int) (*entity.Transcript, error) {
	var createdAt time.Time

	query := `
	SELECT
		t.id,
		s.audio_id,
		a.filename,
		t.segment_id,
		COALESCE(t.user_id::text, '') AS user_id,
		COALESCE(NULLIF(u.username, ''), '') AS username,
		COALESCE(t.ai_text::text, '') AS ai_text,
		COALESCE(NULLIF(t.transcribe_text, ''), '') AS transcribe_text,
		COALESCE(NULLIF(t.report_text, ''), '') AS report_text,
		t.status,
		t.created_at
	FROM transcripts t
	LEFT JOIN users u ON t.user_id = u.id
	JOIN audio_file_segments s ON t.segment_id = s.id
	JOIN audio_files a ON s.audio_id = a.id
	WHERE t.segment_id = $1 AND t.deleted_at = 0 AND s.deleted_at = 0
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
	WHERE segment_id = $1 AND deleted_at = 0
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
		COALESCE(t.user_id::text, '') AS user_id,
		COALESCE(NULLIF(u.username, ''), '') AS username,
		COALESCE(t.ai_text::text, '') AS ai_text,
		COALESCE(NULLIF(t.transcribe_text, ''), '') AS transcribe_text,
		COALESCE(NULLIF(t.report_text, ''), '') AS report_text,
		t.status,
		t.created_at
	FROM transcripts t
	LEFT JOIN users u ON t.user_id = u.id
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

	query += ` ORDER BY t.segment_id OFFSET $` + strconv.Itoa(len(args)+1)
	args = append(args, req.Filter.Offset)
	if req.Filter.Limit != 0 {
		query += " LIMIT $" + strconv.Itoa(len(args)+1)
		args = append(args, req.Filter.Limit)
	}
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
