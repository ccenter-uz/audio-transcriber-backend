// Package usecase implements application business logic. Each logic group in own file.
package usecase

import (
	"context"

	"github.com/mirjalilova/voice_transcribe/internal/entity"
)

//go:generate mockgen -source=interfaces.go -destination=./mocks_test.go -package=usecase_test

type (
	// AuthRepo -.
	AuthRepoI interface {
		Login(ctx context.Context, req *entity.LoginReq) (*entity.UserInfo, error)
		Create(ctx context.Context, req *entity.UserInfo) error
		// GetById(ctx context.Context, id int) (*entity.User, error)
		// GetList(ctx context.Context, req *entity.GetUserReq) (*entity.UserList, error)
		// Update(ctx context.Context, req *entity.UpdateUser) error
		// Delete(ctx context.Context, id int) error
	}

	// TranscriptRepo -.
	TranscriptRepoI interface {
		Create(ctx context.Context, req *entity.CreateTranscript) error
		GetById(ctx context.Context, id int) (*entity.Transcript, error)
		GetList(ctx context.Context, req *entity.GetTranscriptReq) (*entity.TranscriptList, error)
		Update(ctx context.Context, req *entity.UpdateTranscript) error
		// UpdateStatus(ctx context.Context, id *int, user_id string) error
		Delete(ctx context.Context, id int) error
	}

	// AudioSegmentRepo -.
	AudioSegmentRepoI interface {
		Create(ctx context.Context, req *entity.CreateAudioSegment) error
		GetById(ctx context.Context, id int) (*entity.AudioSegment, error)
		GetList(ctx context.Context, req *entity.GetAudioSegmentReq) (*entity.AudioSegmentList, error)
		Delete(ctx context.Context, id int) error
		GetTranscriptPercent(ctx context.Context) (*[]entity.TranscriptPersent, error)
		GetUserTranscriptStatictics(ctx context.Context, user_id string) (*entity.UserTranscriptStatictics, error)
	}

	// AudioFileRepo -.
	AudioFileRepoI interface {
		Create(ctx context.Context, req *entity.CreateAudioFile) (*int, error)
		GetById(ctx context.Context, id int) (*entity.AudioFile, error)
	}
)
