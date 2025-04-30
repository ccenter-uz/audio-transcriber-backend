package repo

import (
	"context"
	"database/sql"
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

func (r *AudioSegmentRepo) Create(ctx context.Context, req *entity.CreateAudioSegment) error {

	tr, err := r.pg.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	// defer func() {
	// 	if err := tr.Rollback(ctx); err != nil {
	// 		// r.logger.Error("failed to rollback transaction: %v", err)
	// 	}
	// }()

	query := `
	INSERT INTO audio_file_segments (audio_id, filename, duration)
	VALUES ($1, $2, $3)
	RETURNING id
	`

	var id int
	_ = tr.QueryRow(ctx, query, req.AudioId, req.FileName, req.Duration).Scan(&id)
	if err != nil {
		tr.Rollback(ctx)
		return fmt.Errorf("failed to create audio segment: %w", err)
	}

	query = `
	INSERT INTO transcripts (segment_id)
	VALUES ($1)
	`

	_, err = tr.Exec(ctx, query, id)
	if err != nil {
		tr.Rollback(ctx)
		return fmt.Errorf("failed to create transcript: %w", err)
	}
	if err := tr.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (r *AudioSegmentRepo) GetById(ctx context.Context, id int) (*entity.AudioSegment, error) {
	var createdAt time.Time

	query := `
	SELECT 
		s.id,
		s.audio_id,
		a.filename,
		s.filename,
		t.status,
		s.created_at
	FROM audio_file_segments s
	JOIN audio_files a ON s.audio_id = a.id
	JOIN 
		transcripts t ON t.segment_id = s.id
	WHERE s.id = $1 AND a.deleted_at = 0 AND s.deleted_at = 0
	`
	segment := &entity.AudioSegment{}
	err := r.pg.Pool.QueryRow(ctx, query, id).Scan(
		&segment.Id,
		&segment.AudioId,
		&segment.AudioName,
		&segment.FilePath,
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
		s.filename,
		t.status,
		s.created_at
	FROM 
		audio_file_segments s
	JOIN 
		audio_files a ON s.audio_id = a.id
	JOIN 
		transcripts t ON t.segment_id = s.id
	WHERE 
		a.deleted_at = 0 AND s.deleted_at = 0
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

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	} else {
		query += `
		AND a.id = (
			SELECT id FROM audio_files
			WHERE status = 'pending' AND deleted_at = 0
			ORDER BY created_at ASC
			LIMIT 1
		)
		AND t.status = 'ready'
		`
	}

	query += ` ORDER BY s.id `

	rows, err := r.pg.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get segment list: %w", err)
	}
	defer rows.Close()

	audioSegments := entity.AudioSegmentList{}
	for rows.Next() {
		var createdAt time.Time
		var count int
		var audioName sql.NullString
		var status sql.NullString
		transcript := entity.AudioSegment{}
		err := rows.Scan(
			&count,
			&transcript.Id,
			&transcript.AudioId,
			&audioName,
			&transcript.FilePath,
			&status,
			&createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan segment: %w", err)
		}

		if audioName.Valid {
			transcript.AudioName = audioName.String
		} else {
			transcript.AudioName = ""
		}

		if status.Valid {
			transcript.Status = status.String
		} else {
			transcript.Status = ""
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

func (r *AudioSegmentRepo) GetUserTranscriptStatictics(ctx context.Context, user_id string) (*entity.UserTranscriptStatictics, error) {
	res := entity.UserTranscriptStatictics{}

	query := `SELECT username FROM users WHERE id = $1`

	err := r.pg.Pool.QueryRow(ctx, query, user_id).Scan(&res.Username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get username: %w", err)
	}

	query = `SELECT * FROM get_user_transcription_statistics($1)`

	err = r.pg.Pool.QueryRow(ctx, query, user_id).Scan(
		&res.TotalAudioFiles,
		&res.TotalChunks,
		&res.TotalMinutes,
		&res.WeeklyAudioFiles,
		&res.WeeklyChunks,
		&res.DailyChunks,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan user transcript statistics: %w", err)
	}

	return &res, nil
}

// func (r *AudioSegmentRepo) GetUserTranscriptCount(ctx context.Context) (*[]entity.UserTranscriptCount, error) {
// 	query := `SELECT
// 				t.user_id,
// 				u.username,
// 				COUNT(t.id) AS done_count
// 			FROM
// 				transcripts t
// 			JOIN
// 				users u ON t.user_id = u.id
// 			WHERE
// 				t.status = 'done'
// 				AND t.transcribe_text IS NOT NULL
// 				AND TRIM(t.transcribe_text) <> ''
// 				AND t.deleted_at = 0
// 			GROUP BY
// 				t.user_id, u.username
// `

// 	rows, err := r.pg.Pool.Query(ctx, query)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	res := []entity.UserTranscriptCount{}
// 	for rows.Next() {
// 		reps := entity.UserTranscriptCount{}
// 		err := rows.Scan(
// 			&reps.UserId,
// 			&reps.Username,
// 			&reps.TotalSegments,
// 		)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to scan user transcript count: %w", err)
// 		}

// 		res = append(res, reps)
// 	}

// 	return &res, nil
// }
