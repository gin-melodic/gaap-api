package user_test

import (
	"context"
	"testing"

	v1 "gaap-api/api/user/v1"
	"gaap-api/internal/controller/user"
	"gaap-api/internal/model"
	"gaap-api/internal/service"

	"github.com/gogf/gf/v2/test/gtest"
)

// mockUserService implements service.IUser for testing
type mockUserService struct {
	getUserProfileFunc        func(ctx context.Context) (out *model.UserProfile, err error)
	updateUserProfileFunc     func(ctx context.Context, in model.UserUpdateInput) (out *model.UserProfile, err error)
	updateThemePreferenceFunc func(ctx context.Context, in model.Theme) (out *model.Theme, err error)
}

func (m *mockUserService) GetUserProfile(ctx context.Context) (out *model.UserProfile, err error) {
	if m.getUserProfileFunc != nil {
		return m.getUserProfileFunc(ctx)
	}
	return nil, nil
}

func (m *mockUserService) UpdateUserProfile(ctx context.Context, in model.UserUpdateInput) (out *model.UserProfile, err error) {
	if m.updateUserProfileFunc != nil {
		return m.updateUserProfileFunc(ctx, in)
	}
	return nil, nil
}

func (m *mockUserService) UpdateThemePreference(ctx context.Context, in model.Theme) (out *model.Theme, err error) {
	if m.updateThemePreferenceFunc != nil {
		return m.updateThemePreferenceFunc(ctx, in)
	}
	return nil, nil
}

func Test_ControllerV1_GetUserProfile(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		ctx := context.Background()

		// Mock service
		mock := &mockUserService{
			getUserProfileFunc: func(ctx context.Context) (*model.UserProfile, error) {
				return &model.UserProfile{
					Email:    "test@example.com",
					Nickname: "Test User",
					Plan:     "FREE",
				}, nil
			},
		}
		service.RegisterUser(mock)

		c := user.NewV1()
		res, err := c.GetUserProfile(ctx, &v1.GetUserProfileReq{})
		t.AssertNil(err)
		t.AssertNE(res, nil)
		t.Assert(res.User.Email, "test@example.com")
		t.Assert(res.User.Nickname, "Test User")
	})
}

func Test_ControllerV1_UpdateUserProfile(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		ctx := context.Background()

		// Mock service
		mock := &mockUserService{
			updateUserProfileFunc: func(ctx context.Context, in model.UserUpdateInput) (*model.UserProfile, error) {
				return &model.UserProfile{
					Email:    "test@example.com",
					Nickname: in.Nickname,
					Plan:     "PRO",
				}, nil
			},
		}
		service.RegisterUser(mock)

		c := user.NewV1()
		req := &v1.UpdateUserProfileReq{
			UserInput: &v1.UserInput{
				Nickname: "Updated User",
				Plan:     "PRO",
			},
		}
		res, err := c.UpdateUserProfile(ctx, req)
		t.AssertNil(err)
		t.AssertNE(res, nil)
		t.Assert(res.User.Nickname, "Updated User")
		t.Assert(res.User.Plan, "PRO")
	})
}
