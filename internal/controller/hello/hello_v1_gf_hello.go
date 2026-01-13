package hello

import (
	"context"

	"gaap-api/api/base"
	v1 "gaap-api/api/hello/v1"
)

func (c *ControllerV1) GfHello(ctx context.Context, req *v1.GfHelloReq) (res *v1.GfHelloRes, err error) {
	return &v1.HelloRes{
		Base: &base.BaseResponse{Message: "success"},
	}, nil
}
