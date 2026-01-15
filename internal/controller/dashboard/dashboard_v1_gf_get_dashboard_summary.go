package dashboard

import (
	"context"

	"gaap-api/api/base"
	v1 "gaap-api/api/dashboard/v1"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"
)

func (c *ControllerV1) GfGetDashboardSummary(ctx context.Context, req *v1.GfGetDashboardSummaryReq) (res *v1.GfGetDashboardSummaryRes, err error) {
	// Parse protobuf from ALE context
	if err := utilproto.ParseFromALE(ctx, &req.GetDashboardSummaryReq); err != nil {
		return nil, err
	}

	summary, err := service.Dashboard().GetDashboardSummary(ctx)
	if err != nil {
		return nil, err
	}

	return &v1.GetDashboardSummaryRes{
		Summary: summaryToProto(summary),
		Base:    &base.BaseResponse{Message: "success"},
	}, nil
}
