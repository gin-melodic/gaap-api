package utils

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// SoftDeleteOptions Configuration for soft delete operation
type SoftDeleteOptions struct {
	TableName      string                                     // Table name
	WhereCondition interface{}                                // Delete condition (map/struct/string)
	WhereArgs      []interface{}                              // Condition parameters
	CascadeFunc    func(ctx context.Context, tx gdb.TX) error // Cascade function
}

// SoftDelete Common soft delete operation. Using this function directly is not recommended.
func SoftDelete(ctx context.Context, tx gdb.TX, opts SoftDeleteOptions) error {
	var (
		deleteTx       = tx
		shouldManageTx = false
		err            error
	)

	// If not give transaction, begin one.
	if deleteTx == nil {
		deleteTx, err = g.DB().Begin(ctx)
		if err != nil {
			return gerror.Wrap(err, "failed to begin transaction")
		}
		shouldManageTx = true
	}

	// Rollback or commit transaction when error occurs or not.
	if shouldManageTx {
		defer func() {
			if err != nil {
				deleteTx.Rollback()
			} else {
				deleteTx.Commit()
			}
		}()
	}

	model := deleteTx.Model(opts.TableName).Where(opts.WhereCondition, opts.WhereArgs...)

	_, err = model.Data(g.Map{"deleted_at": gtime.Now()}).Update()
	if err != nil {
		return gerror.Wrapf(err, "failed to soft delete from %s", opts.TableName)
	}

	// Cascade delete if needed
	if opts.CascadeFunc != nil {
		err = opts.CascadeFunc(ctx, deleteTx)
		if err != nil {
			return gerror.Wrapf(err, "failed to cascade delete from %s", opts.TableName)
		}
	}

	return nil
}
