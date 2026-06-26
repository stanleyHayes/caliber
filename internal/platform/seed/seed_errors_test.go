package seed_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	"github.com/xcreativs/caliber/internal/mocks"
	"github.com/xcreativs/caliber/internal/platform/seed"
)

type failingHasher struct{}

func (failingHasher) Hash(string) (string, error) { return "", errors.New("hash failed") }

type staticHasher struct{}

func (staticHasher) Hash(string) (string, error) { return "demo-hash", nil }

func TestLoad_PropagatesHasherError(t *testing.T) {
	repos, _ := newRepos()
	_, err := seed.Load(context.Background(), repos, failingHasher{}, time.Unix(1, 0))
	assert.Error(t, err, "a hashing failure aborts the seed")
}

func TestLoad_PropagatesRepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	users := mocks.NewMockUserRepository(ctrl)
	users.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errors.New("db down")).AnyTimes()

	repos := seed.Repositories{
		Users:      users,
		Candidates: memory.NewCandidateRepo(),
		Profiles:   memory.NewTalentProfileRepo(),
		Roles:      memory.NewRoleRepo(),
	}
	_, err := seed.Load(context.Background(), repos, staticHasher{}, time.Unix(1, 0))
	assert.Error(t, err, "a repository failure aborts the seed")
}
