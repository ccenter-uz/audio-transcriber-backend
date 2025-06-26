CREATE TYPE file_status AS ENUM('pending', 'processing', 'done', 'error');

CREATE TYPE transcript_status AS ENUM('ready', 'invalid', 'done');

CREATE TYPE role AS ENUM('admin', 'transcriber');

CREATE TABLE users (
    id UUID NOT NULL PRIMARY KEY,
    login VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL, 
    role role NOT NULL DEFAULT 'transcriber',  
    username VARCHAR(100) NOT NULL,       
    first_number VARCHAR(13) NOT NULL,
    service_name VARCHAR(50),
    image_url TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at BIGINT NOT NULL DEFAULT 0,
    CONSTRAINT unique_username_deleted_at UNIQUE (deleted_at, username)
);

CREATE INDEX idx_users_username ON users (username);

CREATE TABLE audio_files (
    id SERIAL PRIMARY KEY,
    filename VARCHAR(200) NOT NULL,
    file_path TEXT NOT NULL,
    status file_status NOT NULL DEFAULT 'pending',
    user_id UUID REFERENCES users(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX idx_audio_files_id ON audio_files (id);

CREATE TABLE audio_file_segments (
    id SERIAL PRIMARY KEY,
    audio_id INT NOT NULL REFERENCES audio_files(id),
    filename VARCHAR(200) NOT NULL,
    duration FLOAT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX idx_audio_file_segments_audio_id ON audio_file_segments (audio_id);

CREATE TABLE transcripts (
    id SERIAL PRIMARY KEY,
    segment_id INT NOT NULL REFERENCES audio_file_segments(id),
    user_id UUID ,
    ai_text TEXT,
    transcribe_text TEXT,
    report_text TEXT,
    status transcript_status NOT NULL DEFAULT 'ready',
    viewed_at TIMESTAMP DEFAULT '2025-05-01 00:00:00',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at BIGINT NOT NULL DEFAULT 0,

    UNIQUE (segment_id, deleted_at) 
);

CREATE OR REPLACE FUNCTION calculate_transcription_percentage() 
RETURNS TABLE (audio_id INT, filename VARCHAR, total_segments BIGINT, completed_segments BIGINT, percentage FLOAT) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        af.id AS audio_id,
        af.filename,
        COUNT(afs.id) AS total_segments, 
        COUNT(t.id) AS completed_segments, 
        (COUNT(t.id)::FLOAT / COUNT(afs.id) * 100) AS percentage 
    FROM 
        audio_files af
    JOIN 
        audio_file_segments afs ON af.id = afs.audio_id
    LEFT JOIN 
        transcripts t ON afs.id = t.segment_id AND t.status = 'done'
    WHERE 
        af.deleted_at = 0 
        AND afs.deleted_at = 0  
    GROUP BY 
        af.id, af.filename;
END;
$$ LANGUAGE plpgsql;


CREATE OR REPLACE FUNCTION get_user_transcription_statistics(p_user_id UUID)
RETURNS TABLE (
    total_audio_files BIGINT,
    total_segments BIGINT,
    total_minutes NUMERIC,
    weekly_audio_files BIGINT,
    weekly_segments BIGINT,
    daily_segments JSONB
) AS $$
DECLARE
    user_created_at DATE;
BEGIN
    SELECT created_at::DATE INTO user_created_at FROM users WHERE id = p_user_id;

    RETURN QUERY
    WITH user_transcripts AS (
        SELECT 
            t.id AS transcript_id,
            t.created_at::DATE AS transcript_created_at,
            afs.audio_id,
            afs.id AS segment_id,
            afs.duration AS duration
        FROM 
            transcripts t
        JOIN audio_file_segments afs ON t.segment_id = afs.id
        JOIN audio_files af ON af.id = afs.audio_id
        WHERE 
            t.user_id = p_user_id
            AND t.deleted_at = 0
            AND af.deleted_at = 0
            AND afs.deleted_at = 0
            AND t.status = 'done'
    ),
    this_week AS (
        SELECT audio_id, segment_id
        FROM user_transcripts
        WHERE transcript_created_at >= date_trunc('week', CURRENT_DATE)
    ),
    daily_counts AS (
        SELECT
            d.day,
            COALESCE(COUNT(ut.transcript_id), 0) AS segments_per_day
        FROM (
            SELECT generate_series(user_created_at, CURRENT_DATE, '1 day') AS day
        ) d
        LEFT JOIN user_transcripts ut
            ON ut.transcript_created_at = d.day
        GROUP BY d.day
    )
    SELECT 
        (SELECT COUNT(DISTINCT audio_id) FROM user_transcripts),
        (SELECT COUNT(*) FROM user_transcripts),
        (SELECT COALESCE(ROUND(SUM(duration)::NUMERIC / 60.0, 2), 0) FROM user_transcripts),
        (SELECT COUNT(DISTINCT audio_id) FROM this_week),
        (SELECT COUNT(*) FROM this_week),
        (SELECT jsonb_object_agg(to_char(day, 'YYYY-MM-DD'), segments_per_day) FROM daily_counts);
END;
$$ LANGUAGE plpgsql;




CREATE OR REPLACE FUNCTION update_audio_file_status_from_transcripts()
RETURNS TRIGGER AS $$
DECLARE
    segment_audio_id INT;
    done_count INT;
    invalid_count INT;
    total_count INT;
    ready_count INT;
BEGIN
    SELECT afs.audio_id INTO segment_audio_id
    FROM audio_file_segments afs
    WHERE afs.id = NEW.segment_id AND afs.deleted_at = 0;

    IF segment_audio_id IS NULL THEN
        RETURN NEW;
    END IF;

    SELECT COUNT(*) INTO total_count
    FROM audio_file_segments
    WHERE audio_id = segment_audio_id AND deleted_at = 0;

    SELECT COUNT(*) INTO invalid_count
    FROM transcripts t
    JOIN audio_file_segments afs ON afs.id = t.segment_id
    WHERE afs.audio_id = segment_audio_id
      AND afs.deleted_at = 0
      AND t.status = 'invalid';

    SELECT COUNT(*) INTO ready_count
    FROM transcripts t
    JOIN audio_file_segments afs ON afs.id = t.segment_id
    WHERE afs.audio_id = segment_audio_id
      AND afs.deleted_at = 0
      AND t.status = 'ready';
      
    SELECT COUNT(*) INTO done_count
    FROM transcripts t
    JOIN audio_file_segments afs ON afs.id = t.segment_id
    WHERE afs.audio_id = segment_audio_id
      AND afs.deleted_at = 0
      AND t.status = 'done';

    IF invalid_count = total_count THEN
        UPDATE audio_files
        SET status = 'error',
            updated_at = NOW()
        WHERE id = segment_audio_id;
    ELSIF done_count + invalid_count = total_count THEN
        UPDATE audio_files
        SET status = 'done',
            updated_at = NOW()
        WHERE id = segment_audio_id;
    ELSIF ready_count = total_count THEN
        UPDATE audio_files
        SET status = 'pending',
            updated_at = NOW()
        WHERE id = segment_audio_id;
    ELSE
        UPDATE audio_files
        SET status = 'processing',
            updated_at = NOW()
        WHERE id = segment_audio_id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


CREATE TRIGGER trg_update_audio_status
AFTER INSERT OR UPDATE ON transcripts
FOR EACH ROW
EXECUTE FUNCTION update_audio_file_status_from_transcripts();











CREATE OR REPLACE FUNCTION get_audio_transcript_stats_by_range(
    from_date date,
    to_date date
)
RETURNS TABLE(
    stat_date date,
    done_segments bigint,
    invalid_segments bigint,
    done_files bigint,
    error_files bigint,
    active_operators numeric
)
AS $$
BEGIN
RETURN QUERY
WITH RECURSIVE 
days AS (
    SELECT generate_series(from_date, to_date, INTERVAL '1 day')::date AS day
),
done_seg AS (
    SELECT updated_at::date AS day, COUNT(*) AS count
    FROM transcripts
    WHERE status = 'done' AND deleted_at = 0
    GROUP BY updated_at::date
),
invalid_seg AS (
    SELECT updated_at::date AS day, COUNT(*) AS count
    FROM transcripts
    WHERE status = 'invalid' AND deleted_at = 0
    GROUP BY updated_at::date
),
done_af AS (
    SELECT updated_at::date AS day, COUNT(*) AS count
    FROM audio_files
    WHERE status = 'done' AND deleted_at = 0
    GROUP BY updated_at::date
),
error_af AS (
    SELECT updated_at::date AS day, COUNT(*) AS count
    FROM audio_files
    WHERE status = 'error' AND deleted_at = 0
    GROUP BY updated_at::date
),
first_transcript AS (
    SELECT
        user_id,
        DATE(updated_at) AS work_day,
        MIN(updated_at) AS start_time
    FROM transcripts
    WHERE status = 'done' AND deleted_at = 0 AND user_id IS NOT NULL
    GROUP BY user_id, DATE(updated_at)
),
blocks AS (
    SELECT
        ft.user_id,
        ft.work_day,
        ft.start_time AS block_start,
        ft.start_time + INTERVAL '30 minutes' AS block_end
    FROM first_transcript ft

    UNION ALL

    SELECT
        b.user_id,
        b.work_day,
        b.block_end AS block_start,
        b.block_end + INTERVAL '30 minutes' AS block_end
    FROM blocks b
    WHERE b.block_end < (b.work_day + INTERVAL '1 day')
),
block_activity AS (
    SELECT DISTINCT
        b.work_day,
        b.user_id,
        b.block_start
    FROM blocks b
    JOIN transcripts t
        ON t.user_id = b.user_id
        AND t.updated_at >= b.block_start
        AND t.updated_at < b.block_end
        AND t.status = 'done'
        AND t.deleted_at = 0
),
active_per_day AS (
    SELECT
        work_day AS day,
        COUNT(*)::numeric / 18.0 AS count
    FROM block_activity
    GROUP BY work_day
)
SELECT
    d.day AS stat_date,
    COALESCE(ds.count, 0),
    COALESCE(isg.count, 0),
    COALESCE(daf.count, 0),
    COALESCE(eaf.count, 0),
    ROUND(COALESCE(apd.count, 0), 2)
FROM days d
LEFT JOIN done_seg ds ON ds.day = d.day
LEFT JOIN invalid_seg isg ON isg.day = d.day
LEFT JOIN done_af daf ON daf.day = d.day
LEFT JOIN error_af eaf ON eaf.day = d.day
LEFT JOIN active_per_day apd ON apd.day = d.day
ORDER BY d.day;
END;
$$ LANGUAGE plpgsql;























CREATE OR REPLACE FUNCTION get_daily_active_blocks_per_user(
    from_date date,
    to_date date
)
RETURNS TABLE(
    stat_date date,
    operator_id uuid,
    username text,
    active_blocks numeric
)
AS $$
BEGIN
RETURN QUERY
WITH RECURSIVE
first_transcript AS (
    SELECT
        t.user_id,
        DATE(t.updated_at) AS work_day,
        MIN(t.updated_at) AS start_time
    FROM transcripts t
    WHERE t.status = 'done' AND t.deleted_at = 0 AND t.user_id IS NOT NULL
        AND t.updated_at::date BETWEEN from_date AND to_date
    GROUP BY t.user_id, DATE(t.updated_at)
),
blocks AS (
    SELECT
        ft.user_id,
        ft.work_day,
        ft.start_time AS block_start,
        ft.start_time + INTERVAL '30 minutes' AS block_end
    FROM first_transcript ft

    UNION ALL

    SELECT
        b.user_id,
        b.work_day,
        b.block_end AS block_start,
        b.block_end + INTERVAL '30 minutes' AS block_end
    FROM blocks b
    WHERE b.block_end < (b.work_day + INTERVAL '1 day')
),
block_activity AS (
    SELECT DISTINCT
        b.work_day,
        b.user_id,
        b.block_start
    FROM blocks b
    JOIN transcripts t
        ON t.user_id = b.user_id
        AND t.updated_at >= b.block_start
        AND t.updated_at < b.block_end
        AND t.status = 'done'
        AND t.deleted_at = 0
),
active_blocks_per_user AS (
    SELECT
        ba.work_day AS stat_date,
        ba.user_id AS operator_id,
        COUNT(*)::numeric / 18.0 AS active_blocks
    FROM block_activity ba
    GROUP BY ba.work_day, ba.user_id
)
SELECT
    abpu.stat_date,
    abpu.operator_id,
    u.username,
    ROUND(abpu.active_blocks, 2)
FROM active_blocks_per_user abpu
LEFT JOIN users u ON u.id = abpu.operator_id
ORDER BY abpu.stat_date, u.username;
END;
$$ LANGUAGE plpgsql;








WITH raw_data AS (
  SELECT
    t.user_id,
    DATE(t.updated_at) AS day,
    COUNT(*) AS done_chunks
  FROM transcripts t
  WHERE t.status = 'done'
    AND t.deleted_at = 0
    AND t.updated_at >= NOW() - INTERVAL '7 days'
  GROUP BY t.user_id, DATE(t.updated_at)
),
avg_chunk_time AS (
  SELECT 0.27::numeric AS avg_chunk_minute
),
kpi_table AS (
  SELECT
    r.user_id,
    r.day,
    r.done_chunks,
    ROUND(480 / a.avg_chunk_minute) AS expected_kpi,
    ROUND((r.done_chunks::numeric / (480 / a.avg_chunk_minute)) * 100, 1) AS kpi_percent
  FROM raw_data r
  CROSS JOIN avg_chunk_time a
)
SELECT
  u.username,
  k.day,
  k.done_chunks,
  k.expected_kpi,
  k.kpi_percent
FROM kpi_table k
JOIN users u ON u.id = k.user_id
ORDER BY k.day, u.username;
voice_transcribe=# WITH raw_data AS (
  SELECT
    t.id,
    t.user_id,
    a.duration,
    EXTRACT(EPOCH FROM (t.updated_at - t.viewed_at)) AS spent_seconds
  FROM transcripts t
  JOIN audio_file_segments a ON a.id = t.segment_id
  WHERE t.viewed_at IS NOT NULL
    AND t.updated_at > t.viewed_at
    AND t.status = 'done'
    AND t.deleted_at = 0
),                                        
bounds AS (
  SELECT
    PERCENTILE_CONT(0.25) WITHIN GROUP (ORDER BY spent_seconds) AS q1,
    PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY spent_seconds) AS q3
  FROM raw_data
),                
filtered AS (                                       
  SELECT r.*                                                                            
  FROM raw_data r
  JOIN bounds b ON r.spent_seconds BETWEEN (b.q1 - 1.5 * (b.q3 - b.q1)) AND (b.q3 + 1.5 * (b.q3 - b.q1))
),
weighted AS (
  SELECT
    SUM(spent_seconds * duration) AS weighted_spent_sum,
    SUM(duration) AS total_weight
  FROM filtered
)              
SELECT
  ROUND((weighted_spent_sum / total_weight)::numeric, 2) AS weighted_avg_spent_per_chunk_sec,
  ROUND(((weighted_spent_sum / total_weight) / 60.0)::numeric, 2) AS weighted_avg_spent_per_chunk_min
FROM weighted;






