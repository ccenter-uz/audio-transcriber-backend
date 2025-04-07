// Package usecase implements application business logic. Each logic group in own file.
package usecase

import (
	"context"

	"github.com/mirjalilova/voice_transcribe/internal/entity"
)

//go:generate mockgen -source=interfaces.go -destination=./mocks_test.go -package=usecase_test

type (
	// TeamRepo -.
	RegionRepoI interface {
		Create(ctx context.Context, req *entity.CreateRegion) error
		GetById(ctx context.Context, id string) (*entity.Region, error)
		GetList(ctx context.Context, req entity.RegionSearch) (*entity.RegionList, error)
		Update(ctx context.Context, req entity.UpdateRegion) error
		Delete(ctx context.Context, id string) error
	}
)
