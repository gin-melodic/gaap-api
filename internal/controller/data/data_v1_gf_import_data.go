package data

import (
	"context"

	"gaap-api/api/base"
	v1 "gaap-api/api/data/v1"
	"gaap-api/internal/model"
	"gaap-api/internal/service"
	utilproto "gaap-api/utility/proto"

	"github.com/gogf/gf/v2/net/ghttp"
)

func (c *ControllerV1) GfImportData(ctx context.Context, req *v1.GfImportDataReq) (res *v1.GfImportDataRes, err error) {
	// Parse protobuf from ALE context
	if err := utilproto.ParseFromALE(ctx, &req.ImportDataReq); err != nil {
		return nil, err
	}

	// Get request from context to handle file upload
	r := ghttp.RequestFromCtx(ctx)

	// Get file from multipart form
	file := r.GetUploadFile("file")
	if file == nil {
		return nil, nil
	}

	// Read file content
	f, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Read all file content
	content := make([]byte, file.Size)
	_, err = f.Read(content)
	if err != nil {
		return nil, err
	}

	input := model.DataImportInput{
		FileContent: content,
		FileName:    file.Filename,
	}

	output, err := service.Data().Import(ctx, input)
	if err != nil {
		return nil, err
	}

	return &v1.ImportDataRes{
		TaskId: output.TaskId,
		Base:   &base.BaseResponse{Message: "success"},
	}, nil
}
