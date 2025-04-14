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
		Login(ctx context.Context, req *entity.LoginReq) (*entity.User, error)
		Create(ctx context.Context, req *entity.CreateUser) error
		GetById(ctx context.Context, id int) (*entity.User, error)
		GetList(ctx context.Context, req *entity.GetUserReq) (*entity.UserList, error)
		Update(ctx context.Context, req *entity.UpdateUser) error
		Delete(ctx context.Context, id int) error
	}

	// TranscriptRepo -.
	TranscriptRepoI interface {
		GetById(ctx context.Context, id int) (*entity.Transcript, error)
		GetList(ctx context.Context, req *entity.GetTranscriptReq) (*entity.TranscriptList, error)
		Update(ctx context.Context, req *entity.UpdateTranscript) error
		UpdateStatus(ctx context.Context, id *int) error
		Delete(ctx context.Context, id int) error
	}

	// AudioSegmentRepo -.
	AudioSegmentRepoI interface {
		GetById(ctx context.Context, id int) (*entity.AudioSegment, error)
		GetList(ctx context.Context, req *entity.GetAudioSegmentReq) (*entity.AudioSegmentList, error)
		Delete(ctx context.Context, id int) error
		GetTranscriptPercent(ctx context.Context) (*[]entity.TranscriptPersent, error)
		GetUserTranscriptCount(ctx context.Context) (*[]entity.UserTranscriptCount, error)
	}

	// AudioFileRepo -.
	AudioFileRepoI interface {
		Create(ctx context.Context, req *entity.CreateAudioFile) error
	}
)
