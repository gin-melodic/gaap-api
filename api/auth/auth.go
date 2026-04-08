// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package auth

import (
	"context"

	"gaap-api/api/auth/v1"
)

type IAuthV1 interface {
	GfLogin(ctx context.Context, req *v1.GfLoginReq) (res *v1.GfLoginRes, err error)
	GfRegister(ctx context.Context, req *v1.GfRegisterReq) (res *v1.GfRegisterRes, err error)
	GfLogout(ctx context.Context, req *v1.GfLogoutReq) (res *v1.GfLogoutRes, err error)
	GfRefreshToken(ctx context.Context, req *v1.GfRefreshTokenReq) (res *v1.GfRefreshTokenRes, err error)
	GfGenerate2FA(ctx context.Context, req *v1.GfGenerate2FAReq) (res *v1.GfGenerate2FARes, err error)
	GfEnable2FA(ctx context.Context, req *v1.GfEnable2FAReq) (res *v1.GfEnable2FARes, err error)
	GfDisable2FA(ctx context.Context, req *v1.GfDisable2FAReq) (res *v1.GfDisable2FARes, err error)
	GfUpdatePassword(ctx context.Context, req *v1.GfUpdatePasswordReq) (res *v1.GfUpdatePasswordRes, err error)
	GfGetCurrencyList(ctx context.Context, req *v1.GfGetCurrencyListReq) (res *v1.GfGetCurrencyListRes, err error)
}
