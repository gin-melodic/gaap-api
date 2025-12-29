package transaction

import (
	"context"
	"fmt"
	"gaap-api/internal/dao"
	"gaap-api/internal/middleware"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"
	"gaap-api/internal/service"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

type sTransaction struct{}

func init() {
	service.RegisterTransaction(New())
}

func New() *sTransaction {
	return &sTransaction{}
}

func (s *sTransaction) ListTransactions(ctx context.Context, in model.TransactionQueryInput) (out []model.Transaction, total int, err error) {
	// Get userId from context for security filtering
	userId, _ := ctx.Value(middleware.UserIdKey).(string)

	m := dao.Transactions.Ctx(ctx)
	if userId != "" {
		m = m.Where("user_id", userId)
	}
	if in.StartDate != "" {
		m = m.Where("date >=", in.StartDate)
	}
	if in.EndDate != "" {
		m = m.Where("date <=", in.EndDate)
	}
	if in.AccountId != "" {
		m = m.Where("from_account_id = ? OR to_account_id = ?", in.AccountId, in.AccountId)
	}
	if in.Type != "" {
		m = m.Where("type", in.Type)
	}

	// Sort
	if in.SortBy != "" {
		order := in.SortBy
		if in.SortOrder == "desc" {
			order += " DESC"
		} else {
			order += " ASC"
		}
		m = m.Order(order)
	}

	total, err = m.Count()
	if err != nil {
		return
	}
	var entities []entity.Transactions
	err = m.Page(in.Page, in.Limit).Scan(&entities)
	if err != nil {
		return
	}

	for _, e := range entities {
		out = append(out, model.Transaction{
			Id:        e.Id,
			Date:      e.Date.String(),
			From:      e.FromAccountId,
			To:        e.ToAccountId,
			Amount:    e.Amount,
			Currency:  e.Currency,
			Note:      e.Note,
			Type:      e.Type,
			CreatedAt: e.CreatedAt,
			UpdatedAt: e.UpdatedAt,
		})
	}
	return
}

func (s *sTransaction) CreateTransaction(ctx context.Context, in model.TransactionCreateInput) (out *model.Transaction, err error) {
	// Use database transaction to ensure atomicity
	err = g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		// 1. Insert transaction record
		_, insertErr := tx.Model(dao.Transactions.Table()).Data(in).Insert()
		if insertErr != nil {
			return fmt.Errorf("failed to insert transaction: %w", insertErr)
		}

		// 2. Apply balance changes
		balanceErr := service.Balance().ApplyTransactionInTx(ctx, tx, &in)
		if balanceErr != nil {
			return fmt.Errorf("failed to apply balance changes: %w", balanceErr)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (s *sTransaction) GetTransaction(ctx context.Context, id string) (out *model.Transaction, err error) {
	// Get userId from context for security filtering
	userId, _ := ctx.Value(middleware.UserIdKey).(string)

	var e entity.Transactions
	m := dao.Transactions.Ctx(ctx).Where("id", id)
	if userId != "" {
		m = m.Where("user_id", userId)
	}
	err = m.Scan(&e)
	if err != nil {
		return
	}
	out = &model.Transaction{
		Id:        e.Id,
		Date:      e.Date.String(),
		From:      e.FromAccountId,
		To:        e.ToAccountId,
		Amount:    e.Amount,
		Currency:  e.Currency,
		Note:      e.Note,
		Type:      e.Type,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
	return
}

// getTransactionById is an internal helper that gets a transaction by ID within a transaction.
func (s *sTransaction) getTransactionByIdInTx(ctx context.Context, tx gdb.TX, id string, userId string) (*model.Transaction, error) {
	var e entity.Transactions
	m := tx.Model(dao.Transactions.Table()).Where("id", id)
	if userId != "" {
		m = m.Where("user_id", userId)
	}
	err := m.Scan(&e)
	if err != nil {
		return nil, err
	}
	if e.Id == "" {
		return nil, fmt.Errorf("transaction not found: %s", id)
	}
	return &model.Transaction{
		Id:        e.Id,
		Date:      e.Date.String(),
		From:      e.FromAccountId,
		To:        e.ToAccountId,
		Amount:    e.Amount,
		Currency:  e.Currency,
		Note:      e.Note,
		Type:      e.Type,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}, nil
}

func (s *sTransaction) UpdateTransaction(ctx context.Context, id string, in model.TransactionUpdateInput) (out *model.Transaction, err error) {
	// Get userId from context for security filtering
	userId, _ := ctx.Value(middleware.UserIdKey).(string)

	// Use database transaction to ensure atomicity
	err = g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		// 1. Get the original transaction to reverse its balance effect
		original, getErr := s.getTransactionByIdInTx(ctx, tx, id, userId)
		if getErr != nil {
			return fmt.Errorf("failed to get original transaction: %w", getErr)
		}

		// 2. Reverse the original balance effect
		reverseErr := service.Balance().ReverseTransactionInTx(ctx, tx, original)
		if reverseErr != nil {
			return fmt.Errorf("failed to reverse original balance: %w", reverseErr)
		}

		// 3. Update the transaction record
		m := tx.Model(dao.Transactions.Table()).Where("id", id)
		if userId != "" {
			m = m.Where("user_id", userId)
		}
		_, updateErr := m.Data(in).Update()
		if updateErr != nil {
			return fmt.Errorf("failed to update transaction: %w", updateErr)
		}

		// 4. Get the updated transaction and apply new balance effect
		updated, getUpdatedErr := s.getTransactionByIdInTx(ctx, tx, id, userId)
		if getUpdatedErr != nil {
			return fmt.Errorf("failed to get updated transaction: %w", getUpdatedErr)
		}

		// Convert to TransactionCreateInput for ApplyTransactionInTx
		newInput := &model.TransactionCreateInput{
			From:     updated.From,
			To:       updated.To,
			Amount:   updated.Amount,
			Currency: updated.Currency,
			Type:     updated.Type,
		}

		applyErr := service.Balance().ApplyTransactionInTx(ctx, tx, newInput)
		if applyErr != nil {
			return fmt.Errorf("failed to apply new balance: %w", applyErr)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.GetTransaction(ctx, id)
}

func (s *sTransaction) DeleteTransaction(ctx context.Context, id string) (err error) {
	// Get userId from context for security filtering
	userId, _ := ctx.Value(middleware.UserIdKey).(string)

	// Use database transaction to ensure atomicity
	err = g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		// 1. Get the transaction to reverse its balance effect
		original, getErr := s.getTransactionByIdInTx(ctx, tx, id, userId)
		if getErr != nil {
			return fmt.Errorf("failed to get transaction: %w", getErr)
		}

		// 2. Reverse the balance effect
		reverseErr := service.Balance().ReverseTransactionInTx(ctx, tx, original)
		if reverseErr != nil {
			return fmt.Errorf("failed to reverse balance: %w", reverseErr)
		}

		// 3. Delete the transaction record
		m := tx.Model(dao.Transactions.Table()).Where("id", id)
		if userId != "" {
			m = m.Where("user_id", userId)
		}
		_, deleteErr := m.Unscoped().Delete()
		if deleteErr != nil {
			return fmt.Errorf("failed to delete transaction: %w", deleteErr)
		}

		return nil
	})

	return err
}
