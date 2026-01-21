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
		m = m.Where(dao.Accounts.Columns().UserId, userId)
	}
	if in.Type != 0 {
		m = m.Where(dao.Accounts.Columns().Type, in.Type)
	}
	if in.ParentId != uuid.Nil {
		m = m.Where(dao.Accounts.Columns().ParentId, in.ParentId)
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
	if initialCurrency == "" {
		initialCurrency = "USD"
	}
	data["CurrencyCode"] = initialCurrency

	// Generate UUID7 for the new account (since InsertAndGetId doesn't support UUID)
	newId, err := uuid.NewV7()
	if err != nil {
		return nil, gerror.Wrap(err, "failed to generate UUID7 for new account")
	}
	data["Id"] = newId

	// Always delete calculated fields
	delete(data, "BalanceDecimal")

	// Wrap everything in a single database transaction for atomicity
	err = g.DB().Transaction(ctx, func(ctx context.Context, dbTx gdb.TX) error {
		// Insert the account
		_, err := dbTx.Model(dao.Accounts.Table()).Data(data).Insert()
		if err != nil {
			return gerror.Wrap(err, "failed to insert initial account")
		}

		// If initial balance is non-zero, create opening balance transaction
		if (initialUnits != 0 || initialNanos != 0) && (in.Type == utils.AccountTypeAsset || in.Type == utils.AccountTypeLiability) {
			// Get or create equity account for this currency
			equityAccountId, err := s.getOrCreateOpeningBalanceEquityAccountInTx(ctx, dbTx, in.CurrencyCode, in.UserId)
			if err != nil {
				return gerror.Wrap(err, "failed to get/create equity account")
			}
			// Create transaction
			txData := model.TransactionCreateInput{
				UserId:        in.UserId,
				Date:          gtime.NewFromStr(accountDate).Format("Y-m-d"),
				FromAccountId: equityAccountId,
				ToAccountId:   newId,
				BalanceUnits:  initialUnits,
				BalanceNanos:  initialNanos,
				CurrencyCode:  initialCurrency,
				Note:          fmt.Sprintf("Opening Balance - %s - %s", in.Name, in.CurrencyCode),
				Type:          utils.TransactionTypeOpeningBalance,
			}
			_, err = service.Transaction().CreateTransaction(ctx, txData, &dbTx)
			if err != nil {
				return gerror.Wrap(err, "failed to create opening balance transaction")
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Retrieve the created account
	var e entity.Accounts
	err = dao.Accounts.Ctx(ctx).Where(dao.Accounts.Columns().Id, newId).Scan(&e)
	if err != nil {
		return nil, err
	}

	return &e, nil
}

func (s *sAccount) GetAccount(ctx context.Context, id uuid.UUID) (out *entity.Accounts, err error) {
	userId := utils.RequireUserId(ctx)

	var e entity.Accounts
	m := dao.Accounts.Ctx(ctx).Where(dao.Accounts.Columns().Id, id).Where(dao.Accounts.Columns().UserId, userId)
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

	if existing.UserId.String() != userId {
		return nil, gerror.New("account does not belong to user")
	}

	// Restrict balance update for EXPENSE and INCOME accounts
	if (existing.Type == utils.AccountTypeExpense || existing.Type == utils.AccountTypeIncome) && (in.BalanceUnits != nil || in.BalanceNanos != nil) {
		return nil, gerror.New("cannot manually update balance for " + gconv.String(existing.Type) + " accounts")
	}

	// Convert input to map to verify handling of zero values
	data := gconv.Map(in)
	if in.ParentId == uuid.Nil {
		delete(data, "ParentId")
	}
	if in.DefaultChildId == uuid.Nil {
		delete(data, "DefaultChildId")
	}

	// Prevent direct update of balance fields
	delete(data, "BalanceUnits")
	delete(data, "BalanceNanos")
	delete(data, "BalanceDecimal")
	delete(data, "CurrencyCode")

	inBalanceUnits := 0
	if in.BalanceUnits != nil {
		inBalanceUnits = int(*in.BalanceUnits)
	}
	inBalanceNanos := 0
	if in.BalanceNanos != nil {
		inBalanceNanos = int(*in.BalanceNanos)
	}

	inMoney := utils.NewMoneyFromUnitsAndNanos(int64(inBalanceUnits), int32(inBalanceNanos), existing.CurrencyCode)
	existsAccountMoney := utils.NewFromEntity(existing)

	// Wrap in transaction
	err = g.DB().Transaction(ctx, func(ctx context.Context, dbTx gdb.TX) error {
		m := dao.Accounts.Ctx(ctx).Where(dao.Accounts.Columns().Id, id)
		if userId != "" {
			m = m.Where(dao.Accounts.Columns().UserId, userId)
		}
		// Only update meta
		if len(data) > 0 {
			_, err := m.Data(data).Update()
			if err != nil {
				return gerror.Wrap(err, "failed to update account")
			}
		}

		// If not update balance, return
		if in.BalanceUnits == nil && in.BalanceNanos == nil {
			return nil
		}

		// Get opening equity account
		equityAccountId, err := s.getOrCreateOpeningBalanceEquityAccountInTx(ctx, dbTx, existing.CurrencyCode, existing.UserId)
		if err != nil {
			return gerror.Wrap(err, "failed to get/create opening equity account")
		}

		tran := model.TransactionCreateInput{
			UserId:        existing.UserId,
			Date:          gtime.Now().Format("Y-m-d"),
			FromAccountId: equityAccountId,
			ToAccountId:   id,
			CurrencyCode:  existing.CurrencyCode,
			Note:          fmt.Sprintf("Update Balance - %s - %s", existing.Name, existing.CurrencyCode),
			Type:          utils.TransactionTypeOpeningBalance,
		}

		// Confirm transfer amount
		addMoney, err := inMoney.Sub(existsAccountMoney)
		if err != nil {
			return gerror.Wrap(err, "failed to calculate transfer amount")
		}

		// If delta money is zero, return
		if addMoney.IsZero() {
			return nil
		}

		units, nanos := addMoney.ToEntityValues()
		tran.BalanceUnits = units
		tran.BalanceNanos = int(nanos)

		_, err = service.Transaction().CreateTransaction(ctx, tran, &dbTx)
		if err != nil {
			return gerror.Wrap(err, "failed to create transaction")
		}

		return nil
	})

	if err != nil {
		return nil, err
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
		err = dao.Accounts.Ctx(ctx).Where(dao.Accounts.Columns().ParentId, id).Scan(&children)
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
		_, err = dao.Accounts.Ctx(ctx).Where(dao.Accounts.Columns().Id, id).Data(g.Map{
			dao.Accounts.Columns().DeletedAt: gtime.Now(),
		}).Update()
		if err != nil {
			return "", gerror.Wrap(err, "failed to delete account")
		}

		// Also delete child accounts if this is a group
		if account.IsGroup && len(childAccountIds) > 0 {
			_, err = dao.Accounts.Ctx(ctx).WhereIn(dao.Accounts.Columns().Id, childAccountIds).FieldsEx(dao.Accounts.Columns().BalanceDecimal).Data(entity.Accounts{
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

// getOrCreateOpeningBalanceEquityAccountInTx gets or creates an opening balance equity account within a transaction.
func (s *sAccount) getOrCreateOpeningBalanceEquityAccountInTx(ctx context.Context, dbTx gdb.TX, currency string, userId uuid.UUID) (uuid.UUID, error) {
	// Look for existing equity account for this currency and user
	equityAccountName := "Opening Balance - " + currency + " - " + userId.String()

	var existing entity.Accounts
	err := dbTx.Model(dao.Accounts.Table()).
		Where(dao.Accounts.Columns().UserId, userId).
		Where(dao.Accounts.Columns().Type, utils.AccountTypeEquity).
		Where(dao.Accounts.Columns().CurrencyCode, currency).
		Where(dao.Accounts.Columns().Name, equityAccountName).
		Where(dao.Accounts.Columns().DeletedAt + " IS NULL").
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
		return uuid.Nil, gerror.Wrap(err, "failed to generate UUID")
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

	_, err = dbTx.Model(dao.Accounts.Table()).
		FieldsEx(
			dao.Accounts.Columns().BalanceDecimal,
			dao.Accounts.Columns().ParentId,
			dao.Accounts.Columns().DefaultChildId,
		).
		Insert(equityAccount)
	if err != nil {
		g.Log().Errorf(ctx, "Failed to insert equity account: %v", err)
		return uuid.Nil, gerror.Wrap(err, "failed to create equity account")
	}

	return newId, nil
}
