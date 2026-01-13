// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package user

import (
	"context"

	"gaap-api/api/user/v1"
)

type IUserV1 interface {
	GfGetProfile(ctx context.Context, req *v1.GfGetProfileReq) (res *v1.GfGetProfileRes, err error)
	GfUpdateProfile(ctx context.Context, req *v1.GfUpdateProfileReq) (res *v1.GfUpdateProfileRes, err error)
	GfUpdateTheme(ctx context.Context, req *v1.GfUpdateThemeReq) (res *v1.GfUpdateThemeRes, err error)
}
