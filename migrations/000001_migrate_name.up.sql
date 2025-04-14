CREATE TYPE file_status AS ENUM('pending', 'processing', 'done', 'error');

CREATE TYPE transcript_status AS ENUM('ready', 'done');

CREATE TYPE role AS ENUM('admin', 'transcriber');

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    password_hash VARCHAR(255) NOT NULL, 
    role role NOT NULL,         
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at BIGINT NOT NULL DEFAULT 0,
    CONSTRAINT unique_username_deleted_at UNIQUE (deleted_at, username)
);

CREATE INDEX idx_users_username ON users (username);

CREATE TABLE audio_files (
    id SERIAL PRIMARY KEY,
    filename VARCHAR(100) NOT NULL,
    file_path TEXT NOT NULL,
    status file_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX idx_audio_files_id ON audio_files (id);

CREATE TABLE audio_file_segments (
    id SERIAL PRIMARY KEY,
    audio_id INT NOT NULL REFERENCES audio_files(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX idx_audio_file_segments_audio_id ON audio_file_segments (audio_id);

CREATE TABLE transcripts (
    id SERIAL PRIMARY KEY,
    segment_id INT NOT NULL REFERENCES audio_file_segments(id),
    user_id INT NOT NULL REFERENCES users(id),
    ai_text TEXT,
    transcribe_text TEXT,
    status transcript_status NOT NULL DEFAULT 'ready',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at BIGINT NOT NULL DEFAULT 0
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


INSERT INTO audio_files (filename, file_path) VALUES
    ('audio1', 'path/audio1'),
    ('audio2', 'path/audio2'),
    ('audio3', 'path/audio3');


INSERT INTO audio_file_segments (audio_id) VALUES
    ('1'),
    ('1'),
    ('3');

INSERT INTO transcripts (segment_id, user_id, ai_text) VALUES
    (2, 2, 'qdhgwqjqwuhjdh'),
    (3, 2, 'wqjdhijqwhdj qiwhdkj qwkjhqkjwh'),
    (4, 2, 'qhjwn e wq ehwq ejqh djhdjjhm ');


