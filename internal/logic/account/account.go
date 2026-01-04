package account

import (
	"context"
	"fmt"
	"gaap-api/internal/dao"
	"gaap-api/internal/middleware"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"
	"gaap-api/internal/service"

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

func (s *sAccount) ListAccounts(ctx context.Context, in model.AccountQueryInput) (out []model.Account, total int, err error) {
	// Get userId from context for security filtering
	userId, _ := ctx.Value(middleware.UserIdKey).(string)

	m := dao.Accounts.Ctx(ctx)
	if userId != "" {
		m = m.Where("user_id", userId)
	}
	if in.Type != "" {
		m = m.Where("type", in.Type)
	}
	if in.ParentId != "" {
		m = m.Where("parent_id", in.ParentId)
	}
	total, err = m.Count()
	if err != nil {
		return
	}
	var entities []entity.Accounts
	err = m.Page(in.Page, in.Limit).Scan(&entities)
	if err != nil {
		return
	}
	// Convert entities to model.Account
	for _, e := range entities {
		out = append(out, model.Account{
			Id:             e.Id,
			ParentId:       e.ParentId,
			Name:           e.Name,
			Type:           e.Type,
			IsGroup:        e.IsGroup,
			Balance:        e.Balance,
			Currency:       e.Currency,
			DefaultChildId: e.DefaultChildId,
			Date:           e.Date.String(),
			Number:         e.Number,
			Remarks:        e.Remarks,
			CreatedAt:      e.CreatedAt,
			UpdatedAt:      e.UpdatedAt,
		})
	}
	return
}

func (s *sAccount) CreateAccount(ctx context.Context, in model.AccountCreateInput) (out *model.Account, err error) {
	// Handle empty UUID fields by converting to map and removing them
	// so they are inserted as NULL (if nullable) or default.
	data := gconv.Map(in)
	if in.ParentId == "" {
		delete(data, "ParentId")
	}
	if in.DefaultChildId == "" {
		delete(data, "DefaultChildId")
	}
	// Remove empty string fields that would cause DB errors (especially Date which expects valid date or NULL)
	if in.Date == "" {
		// Default to today's date
		data["Date"] = gtime.Now().Format("Y-m-d")
	}
	if in.Number == "" {
		delete(data, "Number")
	}
	if in.Remarks == "" {
		delete(data, "Remarks")
	}

	// Generate UUID7 for the new account (since InsertAndGetId doesn't support UUID)
	newId, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	data["Id"] = newId.String()

	// Insert the account
	_, err = dao.Accounts.Ctx(ctx).Data(data).Insert()
	if err != nil {
		return nil, err
	}

	// Retrieve the created account
	var e entity.Accounts
	err = dao.Accounts.Ctx(ctx).Where("id", newId.String()).Scan(&e)
	if err != nil {
		return nil, err
	}

	out = &model.Account{
		Id:             e.Id,
		ParentId:       e.ParentId,
		Name:           e.Name,
		Type:           e.Type,
		IsGroup:        e.IsGroup,
		Balance:        e.Balance,
		Currency:       e.Currency,
		DefaultChildId: e.DefaultChildId,
		Date:           e.Date.String(),
		Number:         e.Number,
		Remarks:        e.Remarks,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}
	return out, nil
}

func (s *sAccount) GetAccount(ctx context.Context, id string) (out *model.Account, err error) {
	// Get userId from context for security filtering
	userId, _ := ctx.Value(middleware.UserIdKey).(string)

	var e entity.Accounts
	m := dao.Accounts.Ctx(ctx).Where("id", id)
	if userId != "" {
		m = m.Where("user_id", userId)
	}
	err = m.Scan(&e)
	if err != nil {
		return
	}
	out = &model.Account{
		Id:             e.Id,
		ParentId:       e.ParentId,
		Name:           e.Name,
		Type:           e.Type,
		IsGroup:        e.IsGroup,
		Balance:        e.Balance,
		Currency:       e.Currency,
		DefaultChildId: e.DefaultChildId,
		Date:           e.Date.String(),
		Number:         e.Number,
		Remarks:        e.Remarks,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}
	return
}

func (s *sAccount) UpdateAccount(ctx context.Context, id string, in model.AccountUpdateInput) (out *model.Account, err error) {
	// Get userId from context for security filtering
	userId, _ := ctx.Value(middleware.UserIdKey).(string)

	// Fetch existing account to check type
	existing, err := s.GetAccount(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("account not found")
	}

	// Restrict balance update for EXPENSE and INCOME accounts
	// We only allow updating balance via transactions for these types.
	// Note: in.Balance is float64, so 0 is considered "empty" by OmitEmpty usually,
	// but here we strictly check if user provided a non-zero value to update.
	if (existing.Type == "EXPENSE" || existing.Type == "INCOME") && in.Balance != 0 {
		return nil, fmt.Errorf("cannot manually update balance for %s accounts", existing.Type)
	}

	m := dao.Accounts.Ctx(ctx).Where("id", id)
	if userId != "" {
		m = m.Where("user_id", userId)
	}
	_, err = m.OmitEmpty().Data(in).Update()
	if err != nil {
		return
	}
	return s.GetAccount(ctx, id)
}

func (s *sAccount) DeleteAccount(ctx context.Context, id string, migrationTargets map[string]string) (taskId string, err error) {
	// Get userId from context for security filtering
	userId, _ := ctx.Value(middleware.UserIdKey).(string)

	// Verify account exists and belongs to user
	account, err := s.GetAccount(ctx, id)
	if err != nil {
		return "", err
	}
	if account == nil || account.Id == "" {
		return "", fmt.Errorf("account not found")
	}

	// Get child accounts if this is a group
	var childAccountIds []string
	if account.IsGroup {
		var children []entity.Accounts
		err = dao.Accounts.Ctx(ctx).Where("parent_id", id).Scan(&children)
		if err != nil {
			return "", err
		}
		for _, child := range children {
			childAccountIds = append(childAccountIds, child.Id)
		}
	}

	// Check if any account has transactions
	accountIds := append([]string{id}, childAccountIds...)
	var totalTransactionCount int
	for _, accId := range accountIds {
		count, err := dao.Transactions.Ctx(ctx).
			Where("from_account_id = ? OR to_account_id = ?", accId, accId).
			Count()
		if err != nil {
			return "", err
		}
		totalTransactionCount += count
	}

	// If no transactions, directly soft-delete without creating a task
	if totalTransactionCount == 0 {
		for _, accId := range accountIds {
			_, err = dao.Accounts.Ctx(ctx).
				Where("id", accId).
				Where("user_id", userId).
				Data(g.Map{"deleted_at": gtime.Now()}).
				Update()
			if err != nil {
				return "", err
			}
		}
		return "", nil // Return empty taskId to indicate direct deletion
	}

	// Has transactions - create migration task
	payload := model.AccountMigrationPayload{
		AccountId:        id,
		ChildAccountIds:  childAccountIds,
		MigrationTargets: migrationTargets,
	}

	task, err := service.Task().CreateTask(ctx, model.TaskCreateInput{
		UserId:  userId,
		Type:    model.TaskTypeAccountMigration,
		Payload: payload,
	})
	if err != nil {
		return "", err
	}

	return task.Id, nil
}

// GetAccountTransactionCount returns the number of transactions involving this account
func (s *sAccount) GetAccountTransactionCount(ctx context.Context, id string) (count int, err error) {
	// Verify account exists and belongs to user
	account, err := s.GetAccount(ctx, id)
	if err != nil {
		return 0, err
	}
	if account == nil || account.Id == "" {
		return 0, fmt.Errorf("account not found")
	}

	// Count transactions where this account is from or to
	count, err = dao.Transactions.Ctx(ctx).
		Where("from_account_id = ? OR to_account_id = ?", id, id).
		Count()
	return count, err
}
