package account

import (
	"context"

	v1 "gaap-api/api/account/v1"
	"gaap-api/api/base"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"

	"github.com/google/uuid"
)

func (c *ControllerV1) GfDeleteAccount(ctx context.Context, req *v1.GfDeleteAccountReq) (res *v1.GfDeleteAccountRes, err error) {
	// Parse protobuf from ALE context
	if err := utilproto.ParseFromALE(ctx, &req.DeleteAccountReq); err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, err
	}

	// Convert migration targets from map[string]string to map[string]uuid.UUID
	migrationTargets := make(map[string]uuid.UUID)
	for currency, targetId := range req.GetMigrationTargets() {
		if parsedId, err := uuid.Parse(targetId); err == nil {
			migrationTargets[currency] = parsedId
		}
	}

	taskId, err := service.Account().DeleteAccount(ctx, id, migrationTargets)
	if err != nil {
		return nil, err
	}

	return &v1.DeleteAccountRes{
		TaskId: taskId,
		Base:   &base.BaseResponse{Message: "success"},
	}, nil
}
