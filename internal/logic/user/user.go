package user

import (
	"context"

	"gaap-api/internal/dao"
	"gaap-api/internal/logic/utils"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"
	"gaap-api/internal/service"

	"github.com/gogf/gf/v2/errors/gerror"
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
	userId := utils.RequireUserId(ctx)

	var user *entity.Users
	err = dao.Users.Ctx(ctx).Where(dao.Users.Columns().Id, userId).Scan(&user)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to get user")
	}
	if user == nil {
		return nil, gerror.New("user not found")
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
	userId := utils.RequireUserId(ctx)

	_, err = dao.Users.Ctx(ctx).Data(in).Where(dao.Users.Columns().Id, userId).Update()
	if err != nil {
		return nil, gerror.Wrap(err, "failed to update user profile")
	}
	return s.GetUserProfile(ctx)
}

func (s *sUser) UpdateThemePreference(ctx context.Context, in model.Theme) (out *model.Theme, err error) {
	userId := utils.RequireUserId(ctx)

	// Validate that the theme exists
	var theme *entity.Themes
	err = dao.Themes.Ctx(ctx).Where(dao.Themes.Columns().Id, in.Id).WhereNull(dao.Themes.Columns().DeletedAt).Scan(&theme)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to get theme")
	}
	if theme == nil {
		return nil, gerror.New("theme not found")
	}

	_, err = dao.Users.Ctx(ctx).Where(dao.Users.Columns().Id, userId).Data(g.Map{dao.Users.Columns().ThemeId: theme.Id}).Update()

	// Return the updated theme
	out = &model.Theme{
		Id:     theme.Id,
		Name:   theme.Name,
		IsDark: theme.IsDark,
	}
	return
}
