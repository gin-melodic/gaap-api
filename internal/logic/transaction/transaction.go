package transaction

import (
	"context"
	"gaap-api/internal/dao"
	"gaap-api/internal/logic/utils"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"
	"gaap-api/internal/service"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/google/uuid"
)

type sTransaction struct{}

func init() {
	service.RegisterTransaction(New())
}

func New() *sTransaction {
	return &sTransaction{}
}

func (s *sTransaction) ListTransactions(ctx context.Context, in model.TransactionQueryInput) (out []entity.Transactions, total int, err error) {
	// Get userId from context for security filtering
	userId := utils.RequireUserId(ctx)

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
	if in.AccountId != uuid.Nil {
		m = m.Where("from_account_id = ? OR to_account_id = ?", in.AccountId, in.AccountId)
	}
	if in.Type != 0 {
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
		return nil, 0, gerror.Wrap(err, "failed to count transactions")
	}

	var entities []entity.Transactions
	err = m.Page(in.Page, in.Limit).Scan(&entities)
	if err != nil {
		return nil, 0, gerror.Wrap(err, "failed to list transactions")
	}

	return entities, total, nil
}

func (s *sTransaction) CreateTransaction(ctx context.Context, in model.TransactionCreateInput) (out *entity.Transactions, err error) {
	// Generate UUID7 for the new transaction
	newId, err := uuid.NewV7()
	if err != nil {
		return nil, gerror.Wrap(err, "failed to generate UUID7 for new transaction")
	}

	txEntity := entity.Transactions{
		Id:            newId,
		UserId:        in.UserId,
		FromAccountId: in.FromAccountId,
		ToAccountId:   in.ToAccountId,
		CurrencyCode:  in.CurrencyCode,
		BalanceUnits:  in.BalanceUnits,
		BalanceNanos:  in.BalanceNanos,
		Note:          in.Note,
		Type:          in.Type,
	}

	// Use database transaction to ensure atomicity
	err = g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		// 1. Insert transaction record
		_, insertErr := tx.Model(dao.Transactions.Table()).Data(txEntity).Insert()
		if insertErr != nil {
			return gerror.Wrap(insertErr, "failed to insert transaction")
		}

		// 2. Apply balance changes
		balanceErr := service.Balance().ApplyTransactionInTx(ctx, tx, &in)
		if balanceErr != nil {
			return gerror.Wrap(balanceErr, "failed to apply balance changes")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Retrieve the created transaction
	var e entity.Transactions
	err = dao.Transactions.Ctx(ctx).Where("id", newId).Scan(&e)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to retrieve created transaction")
	}

	return &e, nil
}

func (s *sTransaction) GetTransaction(ctx context.Context, id uuid.UUID) (out *entity.Transactions, err error) {
	userId := utils.RequireUserId(ctx)

	var e entity.Transactions
	m := dao.Transactions.Ctx(ctx).Where("id", id)
	if userId != "" {
		m = m.Where("user_id", userId)
	}
	err = m.Scan(&e)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to get transaction")
	}
	if e.Id == uuid.Nil {
		return nil, gerror.New("transaction not found")
	}
	return &e, nil
}

// getTransactionByIdInTx is an internal helper that gets a transaction by ID within a transaction.
func (s *sTransaction) getTransactionByIdInTx(ctx context.Context, tx gdb.TX, id uuid.UUID, userId string) (*model.Transaction, error) {
	var e entity.Transactions
	m := tx.Model(dao.Transactions.Table()).Where("id", id)
	if userId != "" {
		m = m.Where("user_id", userId)
	}
	err := m.Scan(&e)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to get transaction")
	}
	if e.Id == uuid.Nil {
		return nil, gerror.Newf("transaction not found: %s", id)
	}
	return &model.Transaction{
		Id:            e.Id,
		UserId:        e.UserId,
		Date:          e.Date,
		FromAccountId: e.FromAccountId,
		ToAccountId:   e.ToAccountId,
		CurrencyCode:  e.CurrencyCode,
		BalanceUnits:  e.BalanceUnits,
		BalanceNanos:  e.BalanceNanos,
		Note:          e.Note,
		Type:          e.Type,
		CreatedAt:     e.CreatedAt,
		UpdatedAt:     e.UpdatedAt,
	}, nil
}

func (s *sTransaction) UpdateTransaction(ctx context.Context, id uuid.UUID, in model.TransactionUpdateInput) (out *entity.Transactions, err error) {
	userId := utils.RequireUserId(ctx)

	// Use database transaction to ensure atomicity
	err = g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		// 1. Get the original transaction to reverse its balance effect
		original, getErr := s.getTransactionByIdInTx(ctx, tx, id, userId)
		if getErr != nil {
			return gerror.Wrap(getErr, "failed to get original transaction")
		}

		// 2. Reverse the original balance effect
		reverseErr := service.Balance().ReverseTransactionInTx(ctx, tx, original)
		if reverseErr != nil {
			return gerror.Wrap(reverseErr, "failed to reverse original balance")
		}

		// 3. Update the transaction record
		updateData := entity.Transactions{
			FromAccountId: in.FromAccountId,
			ToAccountId:   in.ToAccountId,
			CurrencyCode:  in.CurrencyCode,
			BalanceUnits:  in.BalanceUnits,
			BalanceNanos:  in.BalanceNanos,
			Note:          in.Note,
			Type:          in.Type,
		}
		m := tx.Model(dao.Transactions.Table()).Where("id", id)
		if userId != "" {
			m = m.Where("user_id", userId)
		}
		_, updateErr := m.OmitEmpty().Data(updateData).Update()
		if updateErr != nil {
			return gerror.Wrap(updateErr, "failed to update transaction")
		}

		// 4. Get the updated transaction and apply new balance effect
		updated, getUpdatedErr := s.getTransactionByIdInTx(ctx, tx, id, userId)
		if getUpdatedErr != nil {
			return gerror.Wrap(getUpdatedErr, "failed to get updated transaction")
		}

		// Convert to TransactionCreateInput for ApplyTransactionInTx
		newInput := &model.TransactionCreateInput{
			FromAccountId: updated.FromAccountId,
			ToAccountId:   updated.ToAccountId,
			CurrencyCode:  updated.CurrencyCode,
			BalanceUnits:  updated.BalanceUnits,
			BalanceNanos:  updated.BalanceNanos,
			Type:          updated.Type,
		}

		applyErr := service.Balance().ApplyTransactionInTx(ctx, tx, newInput)
		if applyErr != nil {
			return gerror.Wrap(applyErr, "failed to apply new balance")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.GetTransaction(ctx, id)
}

func (s *sTransaction) DeleteTransaction(ctx context.Context, id uuid.UUID) (err error) {
	userId := utils.RequireUserId(ctx)

	// Use database transaction to ensure atomicity
	err = g.DB().Transaction(ctx, func(ctx context.Context, tx gdb.TX) error {
		// 1. Get the transaction to reverse its balance effect
		original, getErr := s.getTransactionByIdInTx(ctx, tx, id, userId)
		if getErr != nil {
			return gerror.Wrap(getErr, "failed to get transaction")
		}

		// 2. Reverse the balance effect
		reverseErr := service.Balance().ReverseTransactionInTx(ctx, tx, original)
		if reverseErr != nil {
			return gerror.Wrap(reverseErr, "failed to reverse balance")
		}

		// 3. Delete the transaction record
		m := tx.Model(dao.Transactions.Table()).Where("id", id)
		if userId != "" {
			m = m.Where("user_id", userId)
		}
		_, deleteErr := m.Unscoped().Delete()
		if deleteErr != nil {
			return gerror.Wrap(deleteErr, "failed to delete transaction")
		}

		return nil
	})

	return err
}
