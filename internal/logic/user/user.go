package user

import (
	"context"
	"errors"

	"gaap-api/internal/dao"
	"gaap-api/internal/middleware"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"
	"gaap-api/internal/service"

	"github.com/gogf/gf/v2/frame/g"
)

type sUser struct{}

func init() {
	service.RegisterUser(New())
}

func New() *sUser {
	return &sUser{}
}

func (s *sUser) GetUserProfile(ctx context.Context) (out *model.UserProfile, err error) {
	userId := ctx.Value(middleware.UserIdKey)
	if userId == nil {
		return nil, errors.New("unauthorized")
	}

	var user *entity.Users
	err = dao.Users.Ctx(ctx).Where("id", userId).Scan(&user)
	if err != nil {
		return
	}
	if user == nil {
		return nil, errors.New("user not found")
	}
	out = &model.UserProfile{
		Email:            user.Email,
		Nickname:         user.Nickname,
		Avatar:           user.Avatar,
		Plan:             user.Plan,
		TwoFactorEnabled: user.TwoFactorEnabled,
		MainCurrency:     user.MainCurrency,
	}
	return
}

func (s *sUser) UpdateUserProfile(ctx context.Context, in model.UserUpdateInput) (out *model.UserProfile, err error) {
	userId := ctx.Value(middleware.UserIdKey)
	if userId == nil {
		return nil, errors.New("unauthorized")
	}

	_, err = dao.Users.Ctx(ctx).Data(in).Where("id", userId).Update()
	if err != nil {
		return
	}
	return s.GetUserProfile(ctx)
}

func (s *sUser) UpdateThemePreference(ctx context.Context, in model.Theme) (out *model.Theme, err error) {
	userId := ctx.Value(middleware.UserIdKey)
	if userId == nil {
		return nil, errors.New("unauthorized")
	}

	// Validate that the theme exists
	var theme *entity.Themes
	err = dao.Themes.Ctx(ctx).Where("id", in.Id).WhereNull("deleted_at").Scan(&theme)
	if err != nil {
		return nil, err
	}
	if theme == nil {
		return nil, errors.New("theme not found")
	}

	// Update user's theme_id
	_, err = dao.Users.Ctx(ctx).Where("id", userId).Data(g.Map{
		"theme_id": in.Id,
	}).Update()
	if err != nil {
		return nil, err
	}

	// Return the updated theme
	out = &model.Theme{
		Id:     theme.Id,
		Name:   theme.Name,
		IsDark: theme.IsDark,
	}
	return
}
