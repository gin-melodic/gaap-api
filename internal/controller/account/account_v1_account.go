package account

import (
	"context"
	v1 "gaap-api/api/account/v1"
	common "gaap-api/api/common/v1"
	"gaap-api/internal/middleware"
	"gaap-api/internal/model"
	"gaap-api/internal/service"
)

func (c *ControllerV1) ListAccounts(ctx context.Context, req *v1.ListAccountsReq) (res *v1.ListAccountsRes, err error) {
	out, total, err := service.Account().ListAccounts(ctx, model.AccountQueryInput{
		Page:     req.Page,
		Limit:    req.Limit,
		Type:     req.Type,
		ParentId: req.ParentId,
	})
	if err != nil {
		return &v1.ListAccountsRes{
			BaseResponse: &common.BaseResponse{
				Message: err.Error(),
			},
		}, nil
	}
	var accounts []v1.Account
	for _, a := range out {
		var parentId *string
		if a.ParentId != "" {
			parentId = &a.ParentId
		}
		var defaultChildId *string
		if a.DefaultChildId != "" {
			defaultChildId = &a.DefaultChildId
		}
		accounts = append(accounts, v1.Account{
			Id:             a.Id,
			ParentId:       parentId,
			Name:           a.Name,
			Type:           a.Type,
			IsGroup:        a.IsGroup,
			Balance:        a.Balance,
			Currency:       a.Currency,
			DefaultChildId: defaultChildId,
			Date:           a.Date,
			Number:         a.Number,
			Remarks:        a.Remarks,
			CreatedAt:      a.CreatedAt,
			UpdatedAt:      a.UpdatedAt,
		})
	}
	res = &v1.ListAccountsRes{
		PaginatedResponse: common.PaginatedResponse{
			Total: total,
			Page:  req.Page,
			Limit: req.Limit,
		},
		Data: accounts,
	}
	return
}

func (c *ControllerV1) CreateAccount(ctx context.Context, req *v1.CreateAccountReq) (res *v1.CreateAccountRes, err error) {
	// Get userId from context (injected by AuthMiddleware)
	userId, _ := ctx.Value(middleware.UserIdKey).(string)

	in := model.AccountCreateInput{
		UserId:   userId,
		Name:     req.Name,
		Type:     req.Type,
		Currency: req.Currency,
		IsGroup:  req.IsGroup,
		Balance:  req.Balance,
		Date:     req.Date,
		Number:   req.Number,
		Remarks:  req.Remarks,
	}
	if req.ParentId != nil {
		in.ParentId = *req.ParentId
	}
	if req.DefaultChildId != nil {
		in.DefaultChildId = *req.DefaultChildId
	}

	out, err := service.Account().CreateAccount(ctx, in)
	if err != nil {
		return &v1.CreateAccountRes{
			BaseResponse: &common.BaseResponse{
				Message: err.Error(),
			},
		}, nil
	}
	if out == nil {
		return nil, nil
	}

	var parentId *string
	if out.ParentId != "" {
		parentId = &out.ParentId
	}
	var defaultChildId *string
	if out.DefaultChildId != "" {
		defaultChildId = &out.DefaultChildId
	}

	res = &v1.CreateAccountRes{
		Account: &v1.Account{
			Id:             out.Id,
			ParentId:       parentId,
			Name:           out.Name,
			Type:           out.Type,
			IsGroup:        out.IsGroup,
			Balance:        out.Balance,
			Currency:       out.Currency,
			DefaultChildId: defaultChildId,
			Date:           out.Date,
			Number:         out.Number,
			Remarks:        out.Remarks,
			CreatedAt:      out.CreatedAt,
			UpdatedAt:      out.UpdatedAt,
		},
	}
	return
}

func (c *ControllerV1) GetAccount(ctx context.Context, req *v1.GetAccountReq) (res *v1.GetAccountRes, err error) {
	out, err := service.Account().GetAccount(ctx, req.Id)
	if err != nil {
		return &v1.GetAccountRes{
			BaseResponse: &common.BaseResponse{
				Message: err.Error(),
			},
		}, nil
	}
	if out == nil {
		return nil, nil
	}

	var parentId *string
	if out.ParentId != "" {
		parentId = &out.ParentId
	}
	var defaultChildId *string
	if out.DefaultChildId != "" {
		defaultChildId = &out.DefaultChildId
	}

	res = &v1.GetAccountRes{
		Account: &v1.Account{
			Id:             out.Id,
			ParentId:       parentId,
			Name:           out.Name,
			Type:           out.Type,
			IsGroup:        out.IsGroup,
			Balance:        out.Balance,
			Currency:       out.Currency,
			DefaultChildId: defaultChildId,
			Date:           out.Date,
			Number:         out.Number,
			Remarks:        out.Remarks,
			CreatedAt:      out.CreatedAt,
			UpdatedAt:      out.UpdatedAt,
		},
	}
	return
}

func (c *ControllerV1) UpdateAccount(ctx context.Context, req *v1.UpdateAccountReq) (res *v1.UpdateAccountRes, err error) {
	in := model.AccountUpdateInput{
		Name:     req.Name,
		Type:     req.Type,
		Currency: req.Currency,
		IsGroup:  req.IsGroup,
		Balance:  req.Balance,
		Date:     req.Date,
		Number:   req.Number,
		Remarks:  req.Remarks,
	}
	if req.ParentId != nil {
		in.ParentId = *req.ParentId
	}
	if req.DefaultChildId != nil {
		in.DefaultChildId = *req.DefaultChildId
	}

	out, err := service.Account().UpdateAccount(ctx, req.Id, in)
	if err != nil {
		return &v1.UpdateAccountRes{
			BaseResponse: &common.BaseResponse{
				Message: err.Error(),
			},
		}, nil
	}
	if out == nil {
		return nil, nil
	}

	var parentId *string
	if out.ParentId != "" {
		parentId = &out.ParentId
	}
	var defaultChildId *string
	if out.DefaultChildId != "" {
		defaultChildId = &out.DefaultChildId
	}

	res = &v1.UpdateAccountRes{
		Account: &v1.Account{
			Id:             out.Id,
			ParentId:       parentId,
			Name:           out.Name,
			Type:           out.Type,
			IsGroup:        out.IsGroup,
			Balance:        out.Balance,
			Currency:       out.Currency,
			DefaultChildId: defaultChildId,
			Date:           out.Date,
			Number:         out.Number,
			Remarks:        out.Remarks,
			CreatedAt:      out.CreatedAt,
			UpdatedAt:      out.UpdatedAt,
		},
	}
	return
}

func (c *ControllerV1) DeleteAccount(ctx context.Context, req *v1.DeleteAccountReq) (res *v1.DeleteAccountRes, err error) {
	taskId, err := service.Account().DeleteAccount(ctx, req.Id, req.MigrationTargets)
	if err != nil {
		return &v1.DeleteAccountRes{
			BaseResponse: &common.BaseResponse{
				Message: err.Error(),
			},
		}, nil
	}
	res = &v1.DeleteAccountRes{
		TaskId: taskId,
	}
	return
}
