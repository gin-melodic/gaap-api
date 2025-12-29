package transaction

import (
	"context"
	common "gaap-api/api/common/v1"
	v1 "gaap-api/api/transaction/v1"
	"gaap-api/internal/middleware"
	"gaap-api/internal/model"
	"gaap-api/internal/service"
)

func (c *ControllerV1) ListTransactions(ctx context.Context, req *v1.ListTransactionsReq) (res *v1.ListTransactionsRes, err error) {
	out, total, err := service.Transaction().ListTransactions(ctx, model.TransactionQueryInput{
		Page:      req.Page,
		Limit:     req.Limit,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		AccountId: req.AccountId,
		Type:      req.Type,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	})
	if err != nil {
		return &v1.ListTransactionsRes{
			BaseResponse: &common.BaseResponse{
				Message: err.Error(),
			},
		}, nil
	}
	var transactions []v1.Transaction
	for _, t := range out {
		transactions = append(transactions, v1.Transaction{
			Id:        t.Id,
			Date:      t.Date,
			From:      t.From,
			To:        t.To,
			Amount:    t.Amount,
			Currency:  t.Currency,
			Note:      t.Note,
			Type:      t.Type,
			CreatedAt: t.CreatedAt,
			UpdatedAt: t.UpdatedAt,
		})
	}
	res = &v1.ListTransactionsRes{
		PaginatedResponse: common.PaginatedResponse{
			Total: total,
			Page:  req.Page,
			Limit: req.Limit,
		},
		Data: transactions,
	}
	return
}

func (c *ControllerV1) CreateTransaction(ctx context.Context, req *v1.CreateTransactionReq) (res *v1.CreateTransactionRes, err error) {
	// Get userId from context (injected by AuthMiddleware)
	userId, _ := ctx.Value(middleware.UserIdKey).(string)

	in := model.TransactionCreateInput{
		UserId:   userId,
		Date:     req.Date,
		From:     req.From,
		To:       req.To,
		Amount:   req.Amount,
		Currency: req.Currency,
		Note:     req.Note,
		Type:     req.Type,
	}
	out, err := service.Transaction().CreateTransaction(ctx, in)
	if err != nil {
		return &v1.CreateTransactionRes{
			BaseResponse: &common.BaseResponse{
				Message: err.Error(),
			},
		}, nil
	}
	if out == nil {
		return nil, nil
	}
	res = &v1.CreateTransactionRes{
		Transaction: &v1.Transaction{
			Id:        out.Id,
			Date:      out.Date,
			From:      out.From,
			To:        out.To,
			Amount:    out.Amount,
			Currency:  out.Currency,
			Note:      out.Note,
			Type:      out.Type,
			CreatedAt: out.CreatedAt,
			UpdatedAt: out.UpdatedAt,
		},
	}
	return
}

func (c *ControllerV1) GetTransaction(ctx context.Context, req *v1.GetTransactionReq) (res *v1.GetTransactionRes, err error) {
	out, err := service.Transaction().GetTransaction(ctx, req.Id)
	if err != nil {
		return &v1.GetTransactionRes{
			BaseResponse: &common.BaseResponse{
				Message: err.Error(),
			},
		}, nil
	}
	if out == nil {
		return nil, nil
	}
	res = &v1.GetTransactionRes{
		Transaction: &v1.Transaction{
			Id:        out.Id,
			Date:      out.Date,
			From:      out.From,
			To:        out.To,
			Amount:    out.Amount,
			Currency:  out.Currency,
			Note:      out.Note,
			Type:      out.Type,
			CreatedAt: out.CreatedAt,
			UpdatedAt: out.UpdatedAt,
		},
	}
	return
}

func (c *ControllerV1) UpdateTransaction(ctx context.Context, req *v1.UpdateTransactionReq) (res *v1.UpdateTransactionRes, err error) {
	in := model.TransactionUpdateInput{
		Date:     req.Date,
		From:     req.From,
		To:       req.To,
		Amount:   req.Amount,
		Currency: req.Currency,
		Note:     req.Note,
		Type:     req.Type,
	}
	out, err := service.Transaction().UpdateTransaction(ctx, req.Id, in)
	if err != nil {
		return &v1.UpdateTransactionRes{
			BaseResponse: &common.BaseResponse{
				Message: err.Error(),
			},
		}, nil
	}
	if out == nil {
		return nil, nil
	}
	res = &v1.UpdateTransactionRes{
		Transaction: &v1.Transaction{
			Id:        out.Id,
			Date:      out.Date,
			From:      out.From,
			To:        out.To,
			Amount:    out.Amount,
			Currency:  out.Currency,
			Note:      out.Note,
			Type:      out.Type,
			CreatedAt: out.CreatedAt,
			UpdatedAt: out.UpdatedAt,
		},
	}
	return
}

func (c *ControllerV1) DeleteTransaction(ctx context.Context, req *v1.DeleteTransactionReq) (res *v1.DeleteTransactionRes, err error) {
	err = service.Transaction().DeleteTransaction(ctx, req.Id)
	if err != nil {
		return &v1.DeleteTransactionRes{
			BaseResponse: &common.BaseResponse{
				Message: err.Error(),
			},
		}, nil
	}
	res = &v1.DeleteTransactionRes{}
	return
}
