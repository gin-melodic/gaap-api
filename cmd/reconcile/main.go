package main

import (
	"encoding/json"
	"fmt"
	"os"

	"gaap-api/internal/boot"
	"gaap-api/internal/logic/reconciliation"

	_ "github.com/gogf/gf/contrib/drivers/pgsql/v2"
	"github.com/gogf/gf/v2/os/gctx"
)

func main() {
	ctx := gctx.GetInitCtx()
	boot.InitConfig(ctx)
	boot.InitDatabaseConfig(ctx)

	report, err := reconciliation.Run(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reconciliation failed: %v\n", err)
		os.Exit(1)
	}

	output, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode reconciliation report: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(output))
	if !report.Passed {
		os.Exit(2)
	}
}
