package account_test

import (
	"context"
	"testing"

	v1 "gaap-api/api/account/v1"
	"gaap-api/internal/controller/account"
	"gaap-api/internal/model"
	"gaap-api/internal/service"

	"github.com/gogf/gf/v2/test/gtest"
)

// mockAccountService implements service.IAccount for testing
type mockAccountService struct {
	listAccountsFunc               func(ctx context.Context, in model.AccountQueryInput) (out []model.Account, total int, err error)
	createAccountFunc              func(ctx context.Context, in model.AccountCreateInput) (out *model.Account, err error)
	getAccountFunc                 func(ctx context.Context, id string) (out *model.Account, err error)
	updateAccountFunc              func(ctx context.Context, id string, in model.AccountUpdateInput) (out *model.Account, err error)
	deleteAccountFunc              func(ctx context.Context, id string, migrationTargets map[string]string) (taskId string, err error)
	getAccountTransactionCountFunc func(ctx context.Context, id string) (count int, err error)
}

func (m *mockAccountService) ListAccounts(ctx context.Context, in model.AccountQueryInput) (out []model.Account, total int, err error) {
	if m.listAccountsFunc != nil {
		return m.listAccountsFunc(ctx, in)
	}
	return nil, 0, nil
}

func (m *mockAccountService) CreateAccount(ctx context.Context, in model.AccountCreateInput) (out *model.Account, err error) {
	if m.createAccountFunc != nil {
		return m.createAccountFunc(ctx, in)
	}
	return nil, nil
}

func (m *mockAccountService) GetAccount(ctx context.Context, id string) (out *model.Account, err error) {
	if m.getAccountFunc != nil {
		return m.getAccountFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockAccountService) UpdateAccount(ctx context.Context, id string, in model.AccountUpdateInput) (out *model.Account, err error) {
	if m.updateAccountFunc != nil {
		return m.updateAccountFunc(ctx, id, in)
	}
	return nil, nil
}

func (m *mockAccountService) DeleteAccount(ctx context.Context, id string, migrationTargets map[string]string) (taskId string, err error) {
	if m.deleteAccountFunc != nil {
		return m.deleteAccountFunc(ctx, id, migrationTargets)
	}
	return "", nil
}

func (m *mockAccountService) GetAccountTransactionCount(ctx context.Context, id string) (count int, err error) {
	if m.getAccountTransactionCountFunc != nil {
		return m.getAccountTransactionCountFunc(ctx, id)
	}
	return 0, nil
}

func Test_ControllerV1_ListAccounts(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		ctx := context.Background()

		mock := &mockAccountService{
			listAccountsFunc: func(ctx context.Context, in model.AccountQueryInput) ([]model.Account, int, error) {
				return []model.Account{
					{Id: "1", Name: "Account 1", Type: "ASSET"},
					{Id: "2", Name: "Account 2", Type: "LIABILITY"},
				}, 2, nil
			},
		}
		service.RegisterAccount(mock)

		c := account.NewV1()
		req := &v1.ListAccountsReq{
			AccountQuery: v1.AccountQuery{
				Page:  1,
				Limit: 10,
			},
		}
		res, err := c.ListAccounts(ctx, req)
		t.AssertNil(err)
		t.AssertNE(res, nil)
		t.Assert(res.Total, 2)
		t.Assert(len(res.Data), 2)
		t.Assert(res.Data[0].Name, "Account 1")
	})
}

func Test_ControllerV1_CreateAccount(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		ctx := context.Background()

		mock := &mockAccountService{
			createAccountFunc: func(ctx context.Context, in model.AccountCreateInput) (*model.Account, error) {
				return &model.Account{
					Id:       "1",
					Name:     in.Name,
					Type:     in.Type,
					Currency: in.Currency,
				}, nil
			},
		}
		service.RegisterAccount(mock)

		c := account.NewV1()
		req := &v1.CreateAccountReq{
			AccountInput: &v1.AccountInput{
				Name:     "New Account",
				Type:     "ASSET",
				Currency: "USD",
			},
		}
		res, err := c.CreateAccount(ctx, req)
		t.AssertNil(err)
		t.AssertNE(res, nil)
		t.Assert(res.Account.Name, "New Account")
		t.Assert(res.Account.Type, "ASSET")
	})
}

func Test_ControllerV1_GetAccount(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		ctx := context.Background()

		mock := &mockAccountService{
			getAccountFunc: func(ctx context.Context, id string) (*model.Account, error) {
				return &model.Account{
					Id:   id,
					Name: "Test Account",
				}, nil
			},
		}
		service.RegisterAccount(mock)

		c := account.NewV1()
		req := &v1.GetAccountReq{
			Id: "123",
		}
		res, err := c.GetAccount(ctx, req)
		t.AssertNil(err)
		t.AssertNE(res, nil)
		t.Assert(res.Account.Id, "123")
		t.Assert(res.Account.Name, "Test Account")
	})
}

func Test_ControllerV1_UpdateAccount(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		ctx := context.Background()

		mock := &mockAccountService{
			updateAccountFunc: func(ctx context.Context, id string, in model.AccountUpdateInput) (*model.Account, error) {
				return &model.Account{
					Id:   id,
					Name: in.Name,
				}, nil
			},
		}
		service.RegisterAccount(mock)

		c := account.NewV1()
		req := &v1.UpdateAccountReq{
			Id: "123",
			AccountInput: &v1.AccountInput{
				Name: "Updated Account",
			},
		}
		res, err := c.UpdateAccount(ctx, req)
		t.AssertNil(err)
		t.AssertNE(res, nil)
		t.Assert(res.Account.Name, "Updated Account")
	})
}

func Test_ControllerV1_DeleteAccount(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		ctx := context.Background()

		mock := &mockAccountService{
			deleteAccountFunc: func(ctx context.Context, id string, migrationTargets map[string]string) (string, error) {
				return "task-123", nil
			},
		}
		service.RegisterAccount(mock)

		c := account.NewV1()
		req := &v1.DeleteAccountReq{
			Id: "123",
		}
		res, err := c.DeleteAccount(ctx, req)
		t.AssertNil(err)
		t.AssertNE(res, nil)
	})
}
