package account

import (
	"context"
	"database/sql"
	"fmt"
	"gaap-api/internal/dao"
	"gaap-api/internal/logic/dashboard"
	"gaap-api/internal/logic/utils"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"
	"gaap-api/internal/service"
	"strings"

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
	userId := utils.RequireUserId(ctx)
	parsedUserId, err := uuid.Parse(userId)
	if err != nil {
		return nil, gerror.New("invalid authenticated user")
	}
	in.UserId = parsedUserId
	if strings.TrimSpace(in.Name) == "" {
		return nil, gerror.New("account name is required")
	}
	if err := validateAccountType(in.Type); err != nil {
		return nil, err
	}
	// Store the requested initial balance
	initialUnits := in.Units
	initialNanos := in.Nanos
	initialCurrency := in.CurrencyCode

	accountDate := in.Date
	if accountDate == "" {
		accountDate = gtime.Now().Format("Y-m-d")
	}

	// Generate UUID7 for the new account (since InsertAndGetId doesn't support UUID)
	newId, err := uuid.NewV7()
	if err != nil {
		return nil, gerror.Wrap(err, "failed to generate UUID7 for new account")
	}

	// Wrap everything in a single database transaction for atomicity
	err = g.DB().Transaction(ctx, func(ctx context.Context, dbTx gdb.TX) error {
		if accessErr := validateUserAccountHierarchyAccess(ctx, dbTx, in.UserId, in.IsGroup, in.ParentId); accessErr != nil {
			return accessErr
		}
		resolvedCurrency, currencyErr := resolveUserBaseCurrency(ctx, dbTx, in.UserId, initialCurrency)
		if currencyErr != nil {
			return currencyErr
		}
		initialCurrency = resolvedCurrency
		in.CurrencyCode = resolvedCurrency
		if parentErr := validateParentAccount(ctx, dbTx, in.ParentId, in.UserId, resolvedCurrency); parentErr != nil {
			return parentErr
		}

		data := g.Map{
			dao.Accounts.Columns().Id:           newId,
			dao.Accounts.Columns().UserId:       in.UserId,
			dao.Accounts.Columns().Name:         strings.TrimSpace(in.Name),
			dao.Accounts.Columns().Type:         in.Type,
			dao.Accounts.Columns().IsGroup:      in.IsGroup,
			dao.Accounts.Columns().CurrencyCode: resolvedCurrency,
			dao.Accounts.Columns().BalanceUnits: 0,
			dao.Accounts.Columns().BalanceNanos: 0,
			dao.Accounts.Columns().Date:         gtime.NewFromStr(accountDate),
			dao.Accounts.Columns().Number:       in.Number,
			dao.Accounts.Columns().Remarks:      in.Remarks,
		}
		if in.ParentId != uuid.Nil {
			data[dao.Accounts.Columns().ParentId] = in.ParentId
		}
		if in.DefaultChildId != uuid.Nil {
			data[dao.Accounts.Columns().DefaultChildId] = in.DefaultChildId
		}
		// Insert the account
		_, err := dbTx.Model(dao.Accounts.Table()).FieldsEx(dao.Accounts.Columns().BalanceDecimal).Data(data).Insert()
		if err != nil {
			return gerror.Wrap(err, "failed to insert initial account")
		}

		// If initial balance is non-zero, create opening balance transaction
		if (initialUnits != 0 || initialNanos != 0) && (in.Type == utils.AccountTypeAsset || in.Type == utils.AccountTypeLiability) {
			// Get or create equity account for this currency
			equityAccountId, err := s.getOrCreateOpeningBalanceEquityAccountInTx(ctx, dbTx, resolvedCurrency, in.UserId, newId)
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
				Note:          fmt.Sprintf("Opening Balance - %s - %s", in.Name, resolvedCurrency),
				Type:          utils.TransactionTypeOpeningBalance,
			}
			_, err = service.Transaction().CreateTransaction(ctx, txData, dbTx)
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

	// Invalidate cache for the new account
	_ = utils.InvalidateCache(ctx, utils.AccountCacheKey(newId.String()))
	dashboard.PublishDashboardRefresh(ctx, userId, "account_create")

	return &e, nil
}

// GetAccount returns an account by ID with caching.
func (s *sAccount) GetAccount(ctx context.Context, id uuid.UUID) (out *entity.Accounts, err error) {
	return s.loadAccountFromDB(ctx, id)
}

// loadAccountFromDB fetches an account directly from the database with verification.
func (s *sAccount) loadAccountFromDB(ctx context.Context, id uuid.UUID) (*entity.Accounts, error) {
	return utils.GetAndVerify(ctx, utils.AccountAccessor, id)
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
	if strings.TrimSpace(in.Name) == "" {
		return nil, gerror.New("account name is required")
	}
	if err := validateAccountType(in.Type); err != nil {
		return nil, err
	}

	// Restrict balance update for EXPENSE, INCOME, and EQUITY accounts
	// These account types should only have their balances modified through transactions
	if (existing.Type == utils.AccountTypeExpense || existing.Type == utils.AccountTypeIncome || existing.Type == utils.AccountTypeEquity) && (in.BalanceUnits != nil || in.BalanceNanos != nil) {
		return nil, gerror.New("cannot manually update balance for " + gconv.String(existing.Type) + " accounts")
	}

	inBalanceUnits := int64(0)
	if in.BalanceUnits != nil {
		inBalanceUnits = *in.BalanceUnits
	}
	inBalanceNanos := 0
	if in.BalanceNanos != nil {
		inBalanceNanos = int(*in.BalanceNanos)
	}

	inMoney := utils.NewMoneyFromUnitsAndNanos(inBalanceUnits, int32(inBalanceNanos), existing.CurrencyCode)
	existsAccountMoney := utils.NewFromEntity(existing)

	// Wrap in transaction
	err = g.DB().Transaction(ctx, func(ctx context.Context, dbTx gdb.TX) error {
		if accessErr := validateUserAccountHierarchyAccess(ctx, dbTx, existing.UserId, in.IsGroup, in.ParentId); accessErr != nil {
			return accessErr
		}
		resolvedCurrency, currencyErr := resolveUserBaseCurrency(ctx, dbTx, existing.UserId, in.CurrencyCode)
		if currencyErr != nil {
			return currencyErr
		}
		if parentErr := validateParentAccount(ctx, dbTx, in.ParentId, existing.UserId, resolvedCurrency); parentErr != nil {
			return parentErr
		}

		data := g.Map{
			dao.Accounts.Columns().Name:     strings.TrimSpace(in.Name),
			dao.Accounts.Columns().Type:     in.Type,
			dao.Accounts.Columns().IsGroup:  in.IsGroup,
			dao.Accounts.Columns().ParentId: nil,
			dao.Accounts.Columns().Number:   in.Number,
			dao.Accounts.Columns().Remarks:  in.Remarks,
		}
		if in.ParentId != uuid.Nil {
			data[dao.Accounts.Columns().ParentId] = in.ParentId
		}
		if in.DefaultChildId != uuid.Nil {
			data[dao.Accounts.Columns().DefaultChildId] = in.DefaultChildId
		}
		if strings.TrimSpace(in.Date) != "" {
			data[dao.Accounts.Columns().Date] = gtime.NewFromStr(in.Date)
		}

		m := dbTx.Model(dao.Accounts.Table()).Where(dao.Accounts.Columns().Id, id)
		if userId != "" {
			m = m.Where(dao.Accounts.Columns().UserId, userId)
		}
		// Only update meta
		result, updateErr := m.Data(data).Update()
		if updateErr != nil {
			return gerror.Wrap(updateErr, "failed to update account")
		}
		rows, rowsErr := result.RowsAffected()
		if rowsErr != nil || rows != 1 {
			return gerror.New("account update did not affect exactly one row")
		}

		// If not update balance, return
		if in.BalanceUnits == nil && in.BalanceNanos == nil {
			return nil
		}

		// Get opening equity account
		equityAccountId, err := s.getOrCreateOpeningBalanceEquityAccountInTx(ctx, dbTx, existing.CurrencyCode, existing.UserId, id)
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

		_, err = service.Transaction().CreateTransaction(ctx, tran, dbTx)
		if err != nil {
			return gerror.Wrap(err, "failed to create transaction")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Invalidate cache after update
	_ = utils.InvalidateCache(ctx, utils.AccountCacheKey(id.String()))
	dashboard.PublishDashboardRefresh(ctx, userId, "account_update")

	return s.GetAccount(ctx, id)
}

func (s *sAccount) DeleteAccount(ctx context.Context, id uuid.UUID, migrationTargets map[string]uuid.UUID) (taskId string, err error) {
	_ = migrationTargets
	// Verify account exists and belongs to user
	account, err := utils.GetAndVerify(ctx, utils.AccountAccessor, id)
	if err != nil {
		return "", gerror.Wrap(err, "failed to get account")
	}

	var childAccountIds []uuid.UUID
	accountIds := []uuid.UUID{id}
	err = g.DB().Transaction(ctx, func(ctx context.Context, dbTx gdb.TX) error {
		if account.IsGroup {
			scanErr := dbTx.Model(dao.Accounts.Table()).
				Fields(dao.Accounts.Columns().Id).
				Where(dao.Accounts.Columns().ParentId, id).
				WhereNull(dao.Accounts.Columns().DeletedAt).
				Scan(&childAccountIds)
			if scanErr != nil {
				return gerror.Wrap(scanErr, "failed to get child accounts")
			}
		}
		accountIds = append(accountIds, childAccountIds...)
		totalTxCount, countErr := dbTx.Model(dao.Transactions.Table()).
			Where(fmt.Sprintf("%s IN(?) OR %s IN(?)",
				dao.Transactions.Columns().FromAccountId,
				dao.Transactions.Columns().ToAccountId),
				accountIds, accountIds).
			WhereNot(dao.Transactions.Columns().Type, utils.TransactionTypeOpeningBalance).
			WhereNull(dao.Transactions.Columns().DeletedAt).
			Count()
		if countErr != nil {
			return gerror.Wrap(countErr, "failed to get transactions count")
		}
		if totalTxCount != 0 {
			return gerror.New("account with transactions cannot be deleted in beta")
		}

		now := gtime.Now()
		_, deleteTxErr := dbTx.Model(dao.Transactions.Table()).
			Where(fmt.Sprintf("%s IN(?) OR %s IN(?)",
				dao.Transactions.Columns().FromAccountId,
				dao.Transactions.Columns().ToAccountId),
				accountIds, accountIds).
			Where(dao.Transactions.Columns().Type, utils.TransactionTypeOpeningBalance).
			WhereNull(dao.Transactions.Columns().DeletedAt).
			Data(g.Map{dao.Transactions.Columns().DeletedAt: now}).
			Update()
		if deleteTxErr != nil {
			return gerror.Wrap(deleteTxErr, "failed to delete opening balance transactions")
		}

		_, equityErr := dbTx.Model(dao.Accounts.Table()).
			WhereIn(dao.Accounts.Columns().EquityAccountId, accountIds).
			Where(dao.Accounts.Columns().Type, utils.AccountTypeEquity).
			WhereNull(dao.Accounts.Columns().DeletedAt).
			Data(g.Map{dao.Accounts.Columns().DeletedAt: now}).
			Update()
		if equityErr != nil {
			return gerror.Wrap(equityErr, "failed to delete opening balance equity accounts")
		}

		result, deleteAccountErr := dbTx.Model(dao.Accounts.Table()).
			WhereIn(dao.Accounts.Columns().Id, accountIds).
			Where(dao.Accounts.Columns().UserId, account.UserId).
			WhereNull(dao.Accounts.Columns().DeletedAt).
			Data(g.Map{dao.Accounts.Columns().DeletedAt: now}).
			Update()
		if deleteAccountErr != nil {
			return gerror.Wrap(deleteAccountErr, "failed to delete account")
		}
		rows, rowsErr := result.RowsAffected()
		if rowsErr != nil || rows != int64(len(accountIds)) {
			return gerror.New("account delete did not affect the expected rows")
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	for _, accountId := range accountIds {
		_ = utils.InvalidateCache(ctx, utils.AccountCacheKey(accountId.String()))
	}
	dashboard.PublishDashboardRefresh(ctx, account.UserId.String(), "account_delete")
	return "", nil
}

// GetAccountTransactionCount returns the number of transactions involving this account, and the number of transactions involving this account without equity
func (s *sAccount) GetAccountTransactionCount(ctx context.Context, id uuid.UUID) (count int, countWithoutEquity int, err error) {
	// Verify account exists and belongs to user
	account, err := s.GetAccount(ctx, id)
	if err != nil {
		return 0, 0, err
	}
	if account == nil || account.Id == uuid.Nil {
		return 0, 0, gerror.New("account not found")
	}

	// Use a single query to fetch both counts for efficiency
	var result struct {
		Count              int `orm:"count"`
		CountWithoutEquity int `orm:"count_without_equity"`
	}

	err = dao.Transactions.Ctx(ctx).
		Fields(fmt.Sprintf(
			"COUNT(*) as count, COUNT(CASE WHEN %s != %d THEN 1 END) as count_without_equity",
			dao.Transactions.Columns().Type,
			utils.TransactionTypeOpeningBalance,
		)).
		Where(fmt.Sprintf("%s = ? OR %s = ?",
			dao.Transactions.Columns().FromAccountId,
			dao.Transactions.Columns().ToAccountId),
			id, id,
		).
		WhereNull(dao.Transactions.Columns().DeletedAt).
		Scan(&result)

	if err != nil {
		return 0, 0, err
	}

	return result.Count, result.CountWithoutEquity, nil
}

// getOrCreateOpeningBalanceEquityAccountInTx gets or creates an opening balance equity account within a transaction.
// It uses equity_account_id FK for efficient lookup and sets bidirectional linking.
func (s *sAccount) getOrCreateOpeningBalanceEquityAccountInTx(ctx context.Context, dbTx gdb.TX, currency string, userId uuid.UUID, sourceAccountId uuid.UUID) (uuid.UUID, error) {
	// Look for existing equity account linked via equity_account_id
	var existing entity.Accounts
	err := dbTx.Model(dao.Accounts.Table()).
		Where(dao.Accounts.Columns().EquityAccountId, sourceAccountId).
		Where(dao.Accounts.Columns().Type, utils.AccountTypeEquity).
		WhereNull(dao.Accounts.Columns().DeletedAt).
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

	equityAccountName := "Opening Balance - " + currency + " - " + userId.String()

	equityAccount := entity.Accounts{
		Id:              newId,
		UserId:          userId,
		Name:            equityAccountName,
		Type:            utils.AccountTypeEquity,
		IsGroup:         false,
		BalanceUnits:    0,
		BalanceNanos:    0,
		CurrencyCode:    currency,
		Date:            gtime.Now(),
		EquityAccountId: sourceAccountId, // Link equity -> source
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

	// Set bidirectional link: source -> equity
	_, err = dbTx.Model(dao.Accounts.Table()).
		Where(dao.Accounts.Columns().Id, sourceAccountId).
		Data(g.Map{dao.Accounts.Columns().EquityAccountId: newId}).
		Update()
	if err != nil {
		g.Log().Errorf(ctx, "Failed to link source account to equity account: %v", err)
		return uuid.Nil, gerror.Wrap(err, "failed to link source account to equity account")
	}

	return newId, nil
}
