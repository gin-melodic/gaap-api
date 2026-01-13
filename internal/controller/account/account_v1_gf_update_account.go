package account

import (
	"context"

	v1 "gaap-api/api/account/v1"
	"gaap-api/api/base"
	"gaap-api/internal/service"

	"github.com/google/uuid"
)

func (c *ControllerV1) GfUpdateAccount(ctx context.Context, req *v1.GfUpdateAccountReq) (res *v1.GfUpdateAccountRes, err error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, err
	}

	input := protoInputToUpdateInput(req.GetInput())

	account, err := service.Account().UpdateAccount(ctx, id, input)
	if err != nil {
		return nil, err
	}

	return &v1.UpdateAccountRes{
		Account: entityToProto(account),
		Base:    &base.BaseResponse{Message: "success"},
	}, nil
}
