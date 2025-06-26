package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
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
	defer func() {
		if err != nil {
			_ = tr.Rollback(ctx)
		}
	}()

	query := `
	INSERT INTO audio_file_segments (audio_id, filename, duration)
	VALUES ($1, $2, $3)
	RETURNING id
	`

	var id int
	row := tr.QueryRow(ctx, query, req.AudioId, req.FileName, req.Duration)
	err = row.Scan(&id)
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

	var audio_id int
	query := `
	SELECT id FROM audio_files
	WHERE status = 'processing' AND deleted_at = 0 AND user_id = $1
	ORDER BY created_at ASC
	LIMIT 1`

	fmt.Println("UserID:", req.UserID)
	err := r.pg.Pool.QueryRow(ctx, query, req.UserID).Scan(&audio_id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			query = `
			SELECT id FROM audio_files
			WHERE status = 'pending' AND deleted_at = 0
			ORDER BY created_at ASC
			LIMIT 1`

			err = r.pg.Pool.QueryRow(ctx, query).Scan(&audio_id)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return nil, fmt.Errorf("no pending audio files available")
				}
				return nil, fmt.Errorf("failed to get pending audio file: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get processing audio file: %w", err)
		}
	}

	query = `
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

	if req.UserID != "" {
		conditions = append(conditions, "s.audio_id = $"+strconv.Itoa(len(args)+1))
		args = append(args, audio_id)
	}

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
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

	query = `SELECT user_id FROM audio_files WHERE id = $1 AND deleted_at = 0`
	var userId string

	err = r.pg.Pool.QueryRow(ctx, query, audio_id).Scan(&userId)
	if userId == "" {
		_, err = r.pg.Pool.Exec(ctx, "UPDATE audio_files SET status = 'processing', user_id = $2, updated_at = now() WHERE id = $1 AND deleted_at = 0", audio_id, req.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to update file: %w", err)
		}
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

func (r *AudioSegmentRepo) DatasetViewer(ctx context.Context, req *entity.Filter, user_id string, report bool) (*entity.DatasetViewerListResponse, error) {
	baseQuery := `
		FROM audio_files af
		JOIN audio_file_segments afs ON af.id = afs.audio_id
		JOIN transcripts t ON afs.id = t.segment_id
		LEFT JOIN users u ON af.user_id = u.id
		JOIN (
			SELECT
				af.id AS audio_id,
				STRING_AGG(t2.transcribe_text, ' ' ORDER BY t2.id) AS all_transcripts
			FROM audio_files af
			JOIN audio_file_segments afs2 ON af.id = afs2.audio_id
			JOIN transcripts t2 ON afs2.id = t2.segment_id
			WHERE t2.deleted_at = 0
			GROUP BY af.id
		) aggregated_segments ON af.id = aggregated_segments.audio_id
		WHERE
			af.deleted_at = 0
			AND afs.deleted_at = 0
			AND t.deleted_at = 0
			AND af.status <> 'pending'
	`

	conditions := []string{}
	args := []interface{}{}
	argIdx := 1

	if user_id != "" {
		conditions = append(conditions, fmt.Sprintf("u.id = $%d", argIdx))
		args = append(args, user_id)
		argIdx++
	}

	statusCondition := "t.status = 'done'"
	if report {
		statusCondition = "t.status = 'invalid'"
	}
	baseQuery += fmt.Sprintf(" AND %s", statusCondition)

	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	err := r.pg.Pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count dataset viewer rows: %w", err)
	}

	selectQuery := `
		SELECT
			af.id AS audio_id,
			af.filename AS audio_filename,
			afs.id AS chunk_id,
			afs.filename AS segment_filename,
			afs.duration,
			t.transcribe_text AS chunk_text,
			LAG(t.transcribe_text) OVER (PARTITION BY af.id ORDER BY afs.id) AS previous_text,
			LEAD(t.transcribe_text) OVER (PARTITION BY af.id ORDER BY afs.id) AS next_text,
			aggregated_segments.all_transcripts,
			t.report_text,
			u.username,
			u.id,
			EXTRACT(EPOCH FROM t.updated_at - t.viewed_at) / 60 AS minutes_spent
	` + baseQuery + `
		ORDER BY af.id, afs.id
		LIMIT $` + fmt.Sprint(argIdx) + ` OFFSET $` + fmt.Sprint(argIdx+1)
	args = append(args, req.Limit, req.Offset)

	rows, err := r.pg.Pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get dataset viewer: %w", err)
	}
	defer rows.Close()

	res := []entity.DatasetViewerList{}
	for rows.Next() {
		reps := entity.DatasetViewerList{}
		err := rows.Scan(
			&reps.AudioID,
			&reps.AudioUrl,
			&reps.ChunkID,
			&reps.ChunkUrl,
			&reps.Duration,
			&reps.ChunkText,
			&reps.PreviouText,
			&reps.NextText,
			&reps.Sentence,
			&reps.ReportText,
			&reps.Transcriber,
			&reps.TranscriberID,
			&reps.MinutesSpent,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan dataset viewer: %w", err)
		}
		res = append(res, reps)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate over dataset viewer rows: %w", err)
	}

	return &entity.DatasetViewerListResponse{
		Total: total,
		Data:  res,
	}, nil
}

func (r *AudioSegmentRepo) GetStatistics(ctx context.Context) (*entity.Statistics, error) {
	query := `
	WITH transcribed AS (
		SELECT 
			u.username,
			t.transcribe_text,
			afs.duration,
			LAG(t.transcribe_text) OVER (PARTITION BY t.user_id ORDER BY afs.id) AS previous_text,
			LEAD(t.transcribe_text) OVER (PARTITION BY t.user_id ORDER BY afs.id) AS next_text
		FROM 
			transcripts t
		JOIN audio_file_segments afs ON t.segment_id = afs.id
		JOIN users u ON t.user_id = u.id
		WHERE t.deleted_at = 0 AND afs.deleted_at = 0 AND t.status = 'done'
	)
	SELECT * FROM transcribed;
	`

	rows, err := r.pg.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	durationStats := make(map[string]int)
	textStats := make(map[string]int)
	prevTextStats := make(map[string]int)
	nextTextStats := make(map[string]int)
	transcriberStats := make(map[string]int)

	for rows.Next() {
		var username string
		var transcribeText, previousText, nextText sql.NullString
		var duration sql.NullFloat64

		if err := rows.Scan(&username, &transcribeText, &duration, &previousText, &nextText); err != nil {
			return nil, err
		}

		transcriberStats[username]++

		if duration.Valid {
			dur := duration.Float64
			var bucket string
			switch {
			case dur <= 1.0:
				bucket = "0-1s"
			case dur <= 2.0:
				bucket = "1-2s"
			case dur <= 3.0:
				bucket = "2-3s"
			case dur <= 4.0:
				bucket = "3-4s"
			case dur <= 5.0:
				bucket = "4-5s"
			case dur <= 6.0:
				bucket = "5-6s"
			case dur <= 7.0:
				bucket = "6-7s"
			case dur <= 8.0:
				bucket = "7-8s"
			case dur <= 9.0:
				bucket = "8-9s"
			case dur <= 10.0:
				bucket = "9-10s"
			case dur <= 11.0:
				bucket = "10-11s"
			case dur <= 12.0:
				bucket = "11-12s"
			case dur <= 13.0:
				bucket = "12-13s"
			case dur <= 14.0:
				bucket = "13-14s"
			case dur <= 15.0:
				bucket = "14-15s"
			case dur <= 16.0:
				bucket = "15-16s"
			case dur <= 17.0:
				bucket = "16-17s"
			case dur <= 18.0:
				bucket = "17-18s"
			case dur <= 19.0:
				bucket = "18-19s"
			case dur <= 20.0:
				bucket = "19-20s"
			case dur <= 21.0:
				bucket = "20-21s"
			case dur <= 22.0:
				bucket = "21-22s"
			default:
				bucket = "22s+"
			}
			durationStats[bucket]++
		}

		bucketByLength := func(text string) string {
			length := len(text)
			switch {
			case length <= 15:
				return "0-15"
			case length <= 30:
				return "16-30"
			case length <= 45:
				return "31-45"
			case length <= 60:
				return "46-60"
			case length <= 75:
				return "61-75"
			case length <= 90:
				return "76-90"
			case length <= 105:
				return "91-105"
			case length <= 120:
				return "106-120"
			default:
				return "121+"
			}
		}

		if transcribeText.Valid {
			textStats[bucketByLength(transcribeText.String)]++
		}
		if previousText.Valid {
			prevTextStats[bucketByLength(previousText.String)]++
		}
		if nextText.Valid {
			nextTextStats[bucketByLength(nextText.String)]++
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &entity.Statistics{
		Duration:    durationStats,
		Text:        textStats,
		PreviouText: prevTextStats,
		NextText:    nextTextStats,
		Transcriber: transcriberStats,
	}, nil
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

func (r *AudioSegmentRepo) GetAudioTranscriptStats(ctx context.Context, fromDate, toDate time.Time) (*[]entity.TranscriptStatictics, error) {
	query := `SELECT * FROM get_audio_transcript_stats_by_range($1, $2)`

	var stats []entity.TranscriptStatictics

	rows, err := r.pg.Pool.Query(ctx, query, fromDate, toDate)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no daily audio transcript stats found: %w", err)
		}
		return nil, fmt.Errorf("failed to get daily audio transcript stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var stateDate time.Time
		var stat entity.TranscriptStatictics

		err := rows.Scan(
			&stateDate,
			&stat.DoneChunks,
			&stat.InvalidChunks,
			&stat.DoneAudioFiles,
			&stat.ErrorAudioFiles,
			&stat.ActiveOperators,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audio transcript stats: %w", err)
		}

		stat.StateDate = stateDate.Format("2006-01-02")
		stats = append(stats, stat)
	}

	query = "SELECT * FROM get_daily_active_blocks_per_user($1, $2)"
	rows, err = r.pg.Pool.Query(ctx, query, fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily active operators: %w", err)
	}
	defer rows.Close()

	blocksMap := make(map[string][]entity.DailyActiveBlock)

	for rows.Next() {
		var block entity.DailyActiveBlock
		var statDate time.Time

		if err := rows.Scan(&statDate, &block.OperatorID, &block.Username, &block.ActiveBlocks); err != nil {
			return nil, fmt.Errorf("failed to scan daily active operator: %w", err)
		}

		block.StatDate = statDate.Format("2006-01-02")
		blocksMap[block.StatDate] = append(blocksMap[block.StatDate], block)
	}

	for i, stat := range stats {
		if blocks, ok := blocksMap[stat.StateDate]; ok {
			stats[i].ActiveOperatorsBlock = blocks
		}
	}

	return &stats, nil
}

func (r *AudioSegmentRepo) GetHourlyTranscripts(ctx context.Context, userId string, date time.Time) (*entity.ListDailyTranscriptResponse, error) {
	query := `SELECT * FROM get_hourly_transcripts($1, $2);`

	rows, err := r.pg.Pool.Query(ctx, query, userId, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	resp := entity.ListDailyTranscriptResponse{
		Data: []entity.DailyTranscriptResponse{},
	}
	userMap := make(map[string]*entity.DailyTranscriptResponse)

	found := false

	for rows.Next() {
		found = true
		var (
			userID     string
			username   string
			hourRange  string
			chunkCount int
			total      int
		)

		if err := rows.Scan(&userID, &username, &hourRange, &chunkCount, &total); err != nil {
			return nil, err
		}

		if _, ok := userMap[userID]; !ok {
			userMap[userID] = &entity.DailyTranscriptResponse{
				UserId:           userID,
				Username:         username,
				DailyTranscripts: []entity.DailyTranscript{},
				TotalCount:       total,
			}
		}

		userMap[userID].DailyTranscripts = append(userMap[userID].DailyTranscripts, entity.DailyTranscript{
			HourRange: hourRange,
			Count:     chunkCount,
		})
	}

	if !found {
		const fallbackQuery = `SELECT username FROM users WHERE id = $1`
		var username string
		err := r.pg.Pool.QueryRow(ctx, fallbackQuery, userId).Scan(&username)
		if err != nil {
			return nil, err
		}

		resp.Data = append(resp.Data, entity.DailyTranscriptResponse{
			UserId:           userId,
			Username:         username,
			DailyTranscripts: []entity.DailyTranscript{},
			TotalCount:       0,
		})

		return &resp, nil
	}

	for _, v := range userMap {
		resp.Data = append(resp.Data, *v)
	}

	return &resp, nil
}

