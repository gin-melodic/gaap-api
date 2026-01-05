package data

import (
	"context"

	v1 "gaap-api/api/data/v1"
	dataLogic "gaap-api/internal/logic/data"

	"github.com/gogf/gf/v2/net/ghttp"
)

func (c *ControllerV1) DownloadExport(ctx context.Context, req *v1.DownloadExportReq) (res *v1.DownloadExportRes, err error) {
	r := ghttp.RequestFromCtx(ctx)
	err = dataLogic.Data().Download(ctx, req, r)
	return nil, err
}
