package auth

import (
	"context"
	"testing"

	internalauth "github.com/anzhiyu-c/anheyu-app/internal/pkg/auth"
	"github.com/anzhiyu-c/anheyu-app/pkg/constant"
	"github.com/anzhiyu-c/anheyu-app/pkg/domain/model"
	"github.com/anzhiyu-c/anheyu-app/pkg/domain/repository"
	"github.com/anzhiyu-c/anheyu-app/pkg/idgen"
	"github.com/stretchr/testify/require"
)

type tokenUserRepoStub struct {
	user *model.User
}

func (s *tokenUserRepoStub) FindByID(_ context.Context, _ uint) (*model.User, error) { return s.user, nil }
func (s *tokenUserRepoStub) FindByUsername(context.Context, string) (*model.User, error) { panic("unexpected call") }
func (s *tokenUserRepoStub) FindByEmail(context.Context, string) (*model.User, error) { panic("unexpected call") }
func (s *tokenUserRepoStub) FindByGroupID(context.Context, uint) ([]*model.User, error) { panic("unexpected call") }
func (s *tokenUserRepoStub) List(context.Context, int, int, string, *uint, *int) ([]*model.User, int64, error) {
	panic("unexpected call")
}
func (s *tokenUserRepoStub) Count(context.Context) (int64, error) { panic("unexpected call") }
func (s *tokenUserRepoStub) Transaction(_ context.Context, fn func(repo repository.UserRepository) error) error {
	return fn(s)
}
func (s *tokenUserRepoStub) Create(context.Context, *model.User) error { panic("unexpected call") }
func (s *tokenUserRepoStub) Update(context.Context, *model.User) error { panic("unexpected call") }
func (s *tokenUserRepoStub) Delete(context.Context, uint) error { panic("unexpected call") }

type tokenSettingServiceStub struct{}

func (s *tokenSettingServiceStub) LoadAllSettings(context.Context) error { panic("unexpected call") }
func (s *tokenSettingServiceStub) Get(key string) string {
	if key == constant.KeyJWTSecret.String() {
		return "test-secret"
	}
	return ""
}
func (s *tokenSettingServiceStub) GetBool(string) bool { panic("unexpected call") }
func (s *tokenSettingServiceStub) GetByKeys([]string) map[string]interface{} { panic("unexpected call") }
func (s *tokenSettingServiceStub) GetSiteConfig() map[string]interface{} { panic("unexpected call") }
func (s *tokenSettingServiceStub) GetConfigVersion() int64 { panic("unexpected call") }
func (s *tokenSettingServiceStub) UpdateSettings(context.Context, map[string]string) error { panic("unexpected call") }
func (s *tokenSettingServiceStub) RegisterPublicSettings([]string) { panic("unexpected call") }
func (s *tokenSettingServiceStub) IsPublicSetting(string) bool { panic("unexpected call") }

func TestTokenService_RefreshAccessToken_PreservesExternalUserID(t *testing.T) {
	require.NoError(t, idgen.InitSqidsEncoder())
	user := &model.User{ID: 1, Status: model.UserStatusActive, UserGroup: model.UserGroup{ID: 2, Permissions: model.NewBoolset()}}
	svc := NewTokenService(&tokenUserRepoStub{user: user}, &tokenSettingServiceStub{}, nil)

	refreshToken, err := internalauth.GenerateRefreshToken(user.ID, "oerx", []byte("test-secret"))
	require.NoError(t, err)

	accessToken, _, err := svc.RefreshAccessToken(context.Background(), refreshToken)
	require.NoError(t, err)

	claims, err := internalauth.ParseToken(accessToken, []byte("test-secret"))
	require.NoError(t, err)
	require.Equal(t, "oerx", claims.ExternalUserID)
}
