package main

import (
	_ "gaap-api/internal/packed"

	"github.com/gogf/gf/v2/os/gctx"

	"gaap-api/internal/cmd"

	_ "github.com/gogf/gf/contrib/drivers/pgsql/v2"

	_ "gaap-api/internal/logic"
)

func main() {
	cmd.Main.Run(gctx.GetInitCtx())
}
