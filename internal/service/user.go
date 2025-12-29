package service

import (
	"context"
	"gaap-api/internal/model"
)

type IUser interface {
	GetUserProfile(ctx context.Context) (out *model.UserProfile, err error)
	UpdateUserProfile(ctx context.Context, in model.UserUpdateInput) (out *model.UserProfile, err error)
	UpdateThemePreference(ctx context.Context, in model.Theme) (out *model.Theme, err error)
}

var localUser IUser

func User() IUser {
	if localUser == nil {
		panic("implement not found for interface IUser, forgot register?")
	}
	return localUser
}

func RegisterUser(i IUser) {
	localUser = i
}
