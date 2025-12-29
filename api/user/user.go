package user

import (
	"context"
	v1 "gaap-api/api/user/v1"
)

type IUserV1 interface {
	GetUserProfile(ctx context.Context, req *v1.GetUserProfileReq) (res *v1.GetUserProfileRes, err error)
	UpdateUserProfile(ctx context.Context, req *v1.UpdateUserProfileReq) (res *v1.UpdateUserProfileRes, err error)
	UpdateThemePreference(ctx context.Context, req *v1.UpdateThemePreferenceReq) (res *v1.UpdateThemePreferenceRes, err error)
}
