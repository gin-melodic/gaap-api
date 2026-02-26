// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"
	"gaap-api/internal/model"
)

type (
	IUser interface {
		// GetUserProfile returns the current user's profile with caching.
		GetUserProfile(ctx context.Context) (out *model.UserProfile, err error)
		// UpdateUserProfile updates the user profile and invalidates the cache.
		UpdateUserProfile(ctx context.Context, in model.UserUpdateInput) (out *model.UserProfile, err error)
		// UpdateThemePreference updates the user's theme preference and invalidates the cache.
		UpdateThemePreference(ctx context.Context, in model.Theme) (out *model.Theme, err error)
	}
)

var (
	localUser IUser
)

func User() IUser {
	if localUser == nil {
		panic("implement not found for interface IUser, forgot register?")
	}
	return localUser
}

func RegisterUser(i IUser) {
	localUser = i
}
