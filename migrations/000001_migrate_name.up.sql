CREATE TYPE file_status AS ENUM('pending', 'processing', 'done', 'error');

CREATE TYPE transcript_status AS ENUM('done', 'reviewed');

CREATE TYPE role AS ENUM('admin', 'transcriber');

CREATE TABLE users (
    user_id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL, 
    role role NOT NULL,         
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE audio_files (
    audio_id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(user_id),
    filename VARCHAR(100) NOT NULL,
    file_path TEXT NOT NULL,
    status file_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE transcripts (
    transcript_id SERIAL PRIMARY KEY,
    audio_id INT NOT NULL REFERENCES audio_files(audio_id),
    text TEXT,
    status transcript_status NOT NULL DEFAULT 'done',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at BIGINT NOT NULL DEFAULT 0
);