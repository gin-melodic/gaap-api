package account

import (
	"context"
	"database/sql"
	"fmt"
	"gaap-api/internal/dao"
	"gaap-api/internal/logic/utils"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"
	"gaap-api/internal/service"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/google/uuid"
)

type sAccount struct{}

func init() {
	service.RegisterAccount(New())
}

func New() *sAccount {
	return &sAccount{}
}

func (s *sAccount) ListAccounts(ctx context.Context, in model.AccountQueryInput) (out []entity.Accounts, total int, err error) {
	// Get userId from context for security filtering
	userId := utils.RequireUserId(ctx)

	m := dao.Accounts.Ctx(ctx)
	if userId != "" {
		m = m.Where("user_id", userId)
	}
	if in.Type != 0 {
		m = m.Where("type", in.Type)
	}
	if in.ParentId != uuid.Nil {
		m = m.Where("parent_id", in.ParentId)
	}
	total, err = m.Count()
	if err != nil {
		return
	}
	var entities []entity.Accounts
	if err = m.Page(in.Page, in.Limit).Scan(&entities); err != nil {
		return
	}
	// Convert entities to model.Account
	if err = gconv.Scan(entities, &out); err != nil {
		return
	}
	return
}

func (s *sAccount) CreateAccount(ctx context.Context, in model.AccountCreateInput) (out *entity.Accounts, err error) {
	// Store the requested initial balance
	initialUnits := in.Units
	initialNanos := in.Nanos
	initialCurrency := in.CurrencyCode

	// Handle empty UUID fields by converting to map and removing them
	// so they are inserted as NULL (if nullable) or default.
	data := gconv.Map(in)
	if in.ParentId == uuid.Nil {
		delete(data, "ParentId")
	}
	if in.DefaultChildId == uuid.Nil {
		delete(data, "DefaultChildId")
	}
	// Remove empty string fields that would cause DB errors (especially Date which expects valid date or NULL)
	accountDate := in.Date
	if accountDate == "" {
		// Default to today's date
		accountDate = gtime.Now().Format("Y-m-d")
		data["Date"] = accountDate
	}
	if in.Number == "" {
		delete(data, "Number")
	}
	if in.Remarks == "" {
		delete(data, "Remarks")
	}

	// Set Balance to 0 initially - balance will be updated via opening balance transaction
	data["Units"] = 0
	data["Nanos"] = 0
	data["CurrencyCode"] = initialCurrency

	// Generate UUID7 for the new account (since InsertAndGetId doesn't support UUID)
	newId, err := uuid.NewV7()
	if err != nil {
		return nil, gerror.Wrap(err, "failed to generate UUID7 for new account")
	}
	data["Id"] = newId

	// Wrap everything in a single database transaction for atomicity
	err = g.DB().Transaction(ctx, func(ctx context.Context, dbTx gdb.TX) error {
		// Insert the account
		_, err := dbTx.Model(dao.Accounts.Table()).Data(data).Insert()
		if err != nil {
			return gerror.Wrap(err, "failed to insert account")
		}

		// If initial balance is non-zero, create opening balance transaction
		if (initialUnits != 0 && initialNanos != 0) && (in.Type == utils.AccountTypeAsset || in.Type == utils.AccountTypeLiability) {
			// Get or create equity account for this currency
			equityAccountId, err := s.getOrCreateOpeningBalanceEquityAccountInTx(ctx, dbTx, in.CurrencyCode, in.UserId)
			if err != nil {
				return gerror.Wrap(err, "failed to get/create equity account")
			}

			// Generate transaction ID
			txId, err := uuid.NewV7()
			if err != nil {
				return err
			}

			txData := entity.Transactions{
				Id:            txId,
				UserId:        in.UserId,
				Date:          gtime.NewFromStr(accountDate),
				FromAccountId: equityAccountId,
				ToAccountId:   newId,
				BalanceUnits:  initialUnits,
				BalanceNanos:  initialNanos,
				Note:          "Opening Balance - " + in.Name,
				Type:          utils.TransactionTypeOpeningBalance,
			}

			_, err = dbTx.Model(dao.Transactions.Table()).Save(txData)
			if err != nil {
				return gerror.Wrap(err, "failed to create opening balance transaction")
			}

			// Apply balance changes within the same transaction
			txInput := &model.TransactionCreateInput{
				UserId:        in.UserId,
				Date:          accountDate,
				FromAccountId: equityAccountId,
				ToAccountId:   newId,
				CurrencyCode:  initialCurrency,
				BalanceUnits:  initialUnits,
				BalanceNanos:  initialNanos,
				Note:          "Opening Balance - " + in.Name,
				Type:          utils.TransactionTypeOpeningBalance,
			}
			if err := service.Balance().ApplyTransactionInTx(ctx, dbTx, txInput); err != nil {
				return gerror.Wrap(err, "failed to apply opening balance")
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Retrieve the created account
	var e entity.Accounts
	err = dao.Accounts.Ctx(ctx).Where("id", newId).Scan(&e)
	if err != nil {
		return nil, err
	}

	return &e, nil
}

func (s *sAccount) GetAccount(ctx context.Context, id uuid.UUID) (out *entity.Accounts, err error) {
	userId := utils.RequireUserId(ctx)

	var e entity.Accounts
	m := dao.Accounts.Ctx(ctx).Where("id", id).Where("user_id", userId)
	err = m.Scan(&e)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to get account")
	}
	if e.Id == uuid.Nil {
		return nil, gerror.New("account not found")
	}
	return &e, nil
}

func (s *sAccount) UpdateAccount(ctx context.Context, id uuid.UUID, in model.AccountUpdateInput) (out *entity.Accounts, err error) {
	userId := utils.RequireUserId(ctx)

	// Fetch existing account to check type
	existing, err := s.GetAccount(ctx, id)
	if err != nil {
		return nil, gerror.Wrap(err, "failed to get account")
	}
	if existing == nil {
		return nil, gerror.New("account not found")
	}

	// Restrict balance update for EXPENSE and INCOME accounts
	if (existing.Type == utils.AccountTypeExpense || existing.Type == utils.AccountTypeIncome) && (in.Units != 0 && in.Nanos != 0) {
		return nil, gerror.New("cannot manually update balance for " + gconv.String(existing.Type) + " accounts")
	}

	m := dao.Accounts.Ctx(ctx).Where("id", id)
	if userId != "" {
		m = m.Where("user_id", userId)
	}
	_, err = m.OmitEmpty().Data(in).Update()
	if err != nil {
		return nil, gerror.Wrap(err, "failed to update account")
	}
	return s.GetAccount(ctx, id)
}

func (s *sAccount) DeleteAccount(ctx context.Context, id uuid.UUID, migrationTargets map[string]uuid.UUID) (taskId string, err error) {
	// Get userId from context for security filtering
	userId := utils.RequireUserId(ctx)

	// Verify account exists and belongs to user
	account, err := s.GetAccount(ctx, id)
	if err != nil {
		return "", err
	}
	if account == nil || account.Id == uuid.Nil {
		return "", gerror.New("account not found")
	}

	if account.UserId.String() != userId {
		return "", gerror.New("account does not belong to user")
	}

	// Get child accounts if this is a group
	var childAccountIds []uuid.UUID
	if account.IsGroup {
		var children []entity.Accounts
		err = dao.Accounts.Ctx(ctx).Where("parent_id", id).Scan(&children)
		if err != nil {
			return "", gerror.Wrap(err, "failed to get child accounts")
		}
		for _, child := range children {
			childAccountIds = append(childAccountIds, child.Id)
		}
	}

	// Check if account has transactions
	transactionCount, err := s.GetAccountTransactionCount(ctx, id)
	if err != nil {
		return "", err
	}

	if transactionCount == 0 {
		// No transactions - can delete directly
		// Soft delete account
		_, err = dao.Accounts.Ctx(ctx).Where("id", id).Data(entity.Accounts{
			DeletedAt: gtime.Now(),
		}).Update()
		if err != nil {
			return "", gerror.Wrap(err, "failed to delete account")
		}

		// Also delete child accounts if this is a group
		if account.IsGroup && len(childAccountIds) > 0 {
			_, err = dao.Accounts.Ctx(ctx).WhereIn("id", childAccountIds).Data(entity.Accounts{
				DeletedAt: gtime.Now(),
			}).Update()
			if err != nil {
				return "", gerror.Wrap(err, "failed to delete child accounts")
			}
		}
		return "", nil // Successfully deleted, no task needed
	}

	// Has transactions - create migration task
	payload := model.AccountMigrationPayload{
		Payload:          &model.Payload{UserId: account.UserId},
		AccountId:        id,
		ChildAccountIds:  childAccountIds,
		MigrationTargets: migrationTargets,
	}

	task, err := service.Task().CreateTask(ctx, model.TaskCreateInput{
		UserId:  account.UserId,
		Type:    model.TaskTypeAccountMigration,
		Payload: payload,
	})
	if err != nil {
		return "", err
	}

	return task.Id.String(), nil
}

// GetAccountTransactionCount returns the number of transactions involving this account
func (s *sAccount) GetAccountTransactionCount(ctx context.Context, id uuid.UUID) (count int, err error) {
	// Verify account exists and belongs to user
	account, err := s.GetAccount(ctx, id)
	if err != nil {
		return 0, err
	}
	if account == nil || account.Id == uuid.Nil {
		return 0, gerror.New("account not found")
	}

	// Count transactions where this account is from or to
	count, err = dao.Transactions.Ctx(ctx).
		Where("from_account_id = ? OR to_account_id = ?", id, id).
		Count()
	return count, err
}

// // getOrCreateOpeningBalanceEquityAccount gets or creates an opening balance equity account for a given currency.
// // Equity accounts are named "Opening Balance - {Currency}" and are used as the source for opening balance transactions.
// func (s *sAccount) getOrCreateOpeningBalanceEquityAccount(ctx context.Context, currency string, userId uuid.UUID) (uuid.UUID, error) {
// 	// Look for existing equity account for this currency and user
// 	equityAccountName := "Opening Balance - " + currency

// 	var existing entity.Accounts
// 	err := dao.Accounts.Ctx(ctx).
// 		Where("user_id", userId).
// 		Where("type", "EQUITY").
// 		Where("currency", currency).
// 		Where("name", equityAccountName).
// 		Where("deleted_at IS NULL").
// 		Scan(&existing)

// 	if err != nil {
// 		return uuid.Nil, gerror.Wrap(err, "failed to query existing equity account")
// 	}

// 	// If found, return its ID
// 	if existing.Id != uuid.Nil {
// 		return existing.Id, nil
// 	}

// 	// Create new equity account
// 	newId, err := uuid.NewV7()
// 	if err != nil {
// 		return uuid.Nil, gerror.Wrap(err, "failed to create equity account")
// 	}

// 	equityAccount := g.Map{
// 		"Id":       newId.String(),
// 		"UserId":   userId,
// 		"Name":     equityAccountName,
// 		"Type":     "EQUITY",
// 		"IsGroup":  false,
// 		"Balance":  0,
// 		"Currency": currency,
// 		"Date":     gtime.Now().Format("Y-m-d"),
// 	}

// 	_, err = dao.Accounts.Ctx(ctx).Data(equityAccount).Insert()
// 	if err != nil {
// 		return uuid.Nil, gerror.Wrap(err, "failed to create equity account")
// 	}

// 	return newId, nil
// }

// getOrCreateOpeningBalanceEquityAccountInTx gets or creates an opening balance equity account within a transaction.
func (s *sAccount) getOrCreateOpeningBalanceEquityAccountInTx(ctx context.Context, dbTx gdb.TX, currency string, userId uuid.UUID) (uuid.UUID, error) {
	// Ensure EQUITY type exists in account_types to prevent FK violation
	exists, err := dbTx.Model("account_types").Where("type", utils.AccountTypeEquity).Exist()
	if err != nil && err != sql.ErrNoRows {
		g.Log().Errorf(ctx, "Failed to check EQUITY type: %v", err)
		return uuid.Nil, fmt.Errorf("failed to check EQUITY account type: %w", err)
	}
	if !exists {
		g.Log().Info(ctx, "EQUITY type missing, creating it automatically")
		_, err = dbTx.Model("account_types").Save(entity.AccountTypes{
			Type:  utils.AccountTypeEquity,
			Label: "Equity",
			Color: "text-purple-600",
			Bg:    "bg-purple-100",
			Icon:  "Landmark",
		})
		if err != nil {
			g.Log().Errorf(ctx, "Failed to insert EQUITY type: %v", err)
			return uuid.Nil, fmt.Errorf("failed to create EQUITY account type: %w", err)
		}
	}

	// Look for existing equity account for this currency and user
	equityAccountName := "Opening Balance - " + currency

	var existing entity.Accounts
	err = dbTx.Model(dao.Accounts.Table()).
		Where("user_id", userId).
		Where("type", utils.AccountTypeEquity).
		Where("currency", currency).
		Where("name", equityAccountName).
		Where("deleted_at IS NULL").
		Scan(&existing)

	if err != nil && err != sql.ErrNoRows {
		g.Log().Errorf(ctx, "Failed to scan equity account: %v", err)
		return uuid.Nil, fmt.Errorf("failed to query existing equity account: %w", err)
	}

	// If found, return its ID
	if existing.Id != uuid.Nil {
		return existing.Id, nil
	}

	// Create new equity account
	newId, err := uuid.NewV7()
	if err != nil {
		g.Log().Errorf(ctx, "Failed to generate UUID: %v", err)
		return uuid.Nil, err
	}

	equityAccount := entity.Accounts{
		Id:           newId,
		UserId:       userId,
		Name:         equityAccountName,
		Type:         utils.AccountTypeEquity,
		IsGroup:      false,
		BalanceUnits: 0,
		BalanceNanos: 0,
		CurrencyCode: currency,
		Date:         gtime.Now(),
	}

	_, err = dbTx.Model(dao.Accounts.Table()).Save(equityAccount)
	if err != nil {
		g.Log().Errorf(ctx, "Failed to insert equity account: %v", err)
		return uuid.Nil, fmt.Errorf("failed to create equity account: %w", err)
	}

	return newId, nil
}
