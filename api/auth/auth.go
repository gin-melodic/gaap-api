package auth

import (
	"context"
	v1 "gaap-api/api/auth/v1"
)

type IAuthV1 interface {
	Login(ctx context.Context, req *v1.LoginReq) (res *v1.LoginRes, err error)
	Register(ctx context.Context, req *v1.RegisterReq) (res *v1.RegisterRes, err error)
	Logout(ctx context.Context, req *v1.LogoutReq) (res *v1.LogoutRes, err error)
	Generate2FA(ctx context.Context, req *v1.Generate2FAReq) (res *v1.Generate2FARes, err error)
	Enable2FA(ctx context.Context, req *v1.Enable2FAReq) (res *v1.Enable2FARes, err error)
	Disable2FA(ctx context.Context, req *v1.Disable2FAReq) (res *v1.Disable2FARes, err error)
}
