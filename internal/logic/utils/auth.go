package utils

import (
	"context"
	"gaap-api/internal/dao"
	"gaap-api/internal/middleware"
	"gaap-api/internal/model/entity"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/google/uuid"
)

func RequireUserId(ctx context.Context) string {
	id, ok := ctx.Value(middleware.UserIdKey).(string)
	if !ok || id == "" {
		// In production, this error should be handled by Recovery middleware
		g.Log().Panicf(ctx, "user id not found in context")
	}
	return id
}

// FieldAccessor
type FieldAccessor[T any] struct {
	Model           func(ctx context.Context) *gdb.Model
	IdGetter        func(*T) uuid.UUID
	UserIdGetter    func(*T) uuid.UUID
	IdColumn        string
	DeletedAtColumn string
	ResourceName    string
}

// Pre-defined accessors
var (
	AccountAccessor = FieldAccessor[entity.Accounts]{
		Model:           func(ctx context.Context) *gdb.Model { return dao.Accounts.Ctx(ctx) },
		IdGetter:        func(a *entity.Accounts) uuid.UUID { return a.Id },
		UserIdGetter:    func(a *entity.Accounts) uuid.UUID { return a.UserId },
		IdColumn:        dao.Accounts.Columns().Id,
		DeletedAtColumn: dao.Accounts.Columns().DeletedAt,
		ResourceName:    "account",
	}

	TransferAccessor = FieldAccessor[entity.Transactions]{
		Model:           func(ctx context.Context) *gdb.Model { return dao.Transactions.Ctx(ctx) },
		IdGetter:        func(t *entity.Transactions) uuid.UUID { return t.Id },
		UserIdGetter:    func(t *entity.Transactions) uuid.UUID { return t.UserId },
		IdColumn:        dao.Transactions.Columns().Id,
		DeletedAtColumn: dao.Transactions.Columns().DeletedAt,
		ResourceName:    "transfer",
	}

	UserAccessor = FieldAccessor[entity.Users]{
		Model:           func(ctx context.Context) *gdb.Model { return dao.Users.Ctx(ctx) },
		IdGetter:        func(u *entity.Users) uuid.UUID { return u.Id },
		UserIdGetter:    func(u *entity.Users) uuid.UUID { return u.Id }, // 用户验证自己
		IdColumn:        dao.Users.Columns().Id,
		DeletedAtColumn: dao.Users.Columns().DeletedAt,
		ResourceName:    "user",
	}
)

// GetAndVerify Use this function to get and verify a resource
func GetAndVerify[T any](ctx context.Context, accessor FieldAccessor[T], resourceId uuid.UUID) (*T, error) {
	userId := RequireUserId(ctx)

	var resource T
	model := accessor.Model(ctx).Where(accessor.IdColumn, resourceId)
	if accessor.DeletedAtColumn != "" {
		model = model.WhereNull(accessor.DeletedAtColumn)
	}
	err := model.Scan(&resource)
	if err != nil {
		return nil, gerror.Wrapf(err, "failed to get %s", accessor.ResourceName)
	}

	if accessor.IdGetter(&resource) == uuid.Nil {
		return nil, gerror.Newf("%s not found", accessor.ResourceName)
	}

	if accessor.UserIdGetter(&resource).String() != userId {
		return nil, gerror.Newf("%s does not belong to user", accessor.ResourceName)
	}

	return &resource, nil
}
