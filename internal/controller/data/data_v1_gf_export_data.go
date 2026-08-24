package data

import (
	"context"

	"gaap-api/api/base"
	v1 "gaap-api/api/data/v1"
	"gaap-api/internal/model"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"
)

func (c *ControllerV1) GfExportData(ctx context.Context, req *v1.GfExportDataReq) (res *v1.GfExportDataRes, err error) {
	// Parse protobuf from ALE context
	if err := utilproto.ParseFromALE(ctx, &req.ExportDataReq); err != nil {
		return nil, err
	}

	var startDate, endDate string
	params := req.ExportDataReq.GetParams()
	if params != nil {
		startDate = params.GetStartDate()
		endDate = params.GetEndDate()
	}

	input := model.DataExportInput{
		StartDate: startDate,
		EndDate:   endDate,
	}

	output, err := service.Data().Export(ctx, input)
	if err != nil {
		return nil, err
	}

	return &v1.ExportDataRes{
		TaskId: output.TaskId,
		Base:   &base.BaseResponse{Message: "success"},
	}, nil
}
