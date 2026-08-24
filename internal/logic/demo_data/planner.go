package demo_data

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"gaap-api/internal/logic/utils"
	"gaap-api/internal/model"
	"gaap-api/internal/model/entity"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type accountState struct {
	account entity.Accounts
	balance decimal.Decimal
}

type plannedTransaction struct {
	input  model.TransactionCreateInput
	amount decimal.Decimal
}

type transactionTemplate struct {
	typeValue    int
	fromType     int
	toType       int
	fromKeywords []string
	toKeywords   []string
	note         string
	minimumCents int64
	maximumCents int64
}

func planTransactions(user entity.Users, accounts []entity.Accounts, businessDate time.Time, location *time.Location) ([]plannedTransaction, error) {
	states := make([]*accountState, 0, len(accounts))
	for _, account := range accounts {
		balance := decimal.NewFromInt(account.BalanceUnits).Add(decimal.New(int64(account.BalanceNanos), -9))
		if (account.Type == utils.AccountTypeAsset || account.Type == utils.AccountTypeLiability) && balance.IsNegative() {
			return nil, fmt.Errorf("demo account %s has a negative financial balance", account.Id)
		}
		states = append(states, &accountState{account: account, balance: balance})
	}
	sort.Slice(states, func(i, j int) bool { return states[i].account.Id.String() < states[j].account.Id.String() })

	rng := newDateRandom(user.Id, businessDate)
	templates := recurringTemplates(businessDate)
	targetCount := rng.Intn(5)
	if targetCount < len(templates) {
		targetCount = len(templates)
	}
	if targetCount > 4 {
		targetCount = 4
	}
	for len(templates) < targetCount {
		templates = append(templates, randomTemplate(rng, businessDate.Weekday()))
	}

	planned := make([]plannedTransaction, 0, len(templates))
	for _, template := range templates {
		from := chooseAccount(rng, states, template.fromType, template.fromKeywords, uuid.Nil)
		if from == nil {
			continue
		}
		to := chooseAccount(rng, states, template.toType, template.toKeywords, from.account.Id)
		if to == nil {
			continue
		}
		amount := amountFromRange(rng, template.minimumCents, template.maximumCents)
		amount, ok := fitToAvailableBalance(amount, from)
		if !ok {
			continue
		}

		applySimulation(from, to, amount)
		units, nanos := decimalToMoney(amount)
		transactionTime := randomTransactionTime(rng, businessDate, location)
		planned = append(planned, plannedTransaction{
			input: model.TransactionCreateInput{
				UserId:        user.Id,
				Date:          transactionTime.Format(time.RFC3339),
				FromAccountId: from.account.Id,
				ToAccountId:   to.account.Id,
				CurrencyCode:  user.MainCurrency,
				BalanceUnits:  units,
				BalanceNanos:  nanos,
				Note:          template.note,
				Type:          template.typeValue,
			},
			amount: amount,
		})
	}
	return planned, nil
}

func newDateRandom(userID uuid.UUID, businessDate time.Time) *rand.Rand {
	digest := sha256.Sum256([]byte(userID.String() + "|" + businessDate.Format(time.DateOnly)))
	seed := int64(binary.BigEndian.Uint64(digest[:8]))
	return rand.New(rand.NewSource(seed))
}

func recurringTemplates(date time.Time) []transactionTemplate {
	result := make([]transactionTemplate, 0, 2)
	day := date.Day()
	if day == 1 {
		result = append(result,
			incomeTemplate([]string{"interest"}, []string{"saving"}, "Monthly savings interest", 1_701, 2_499),
			expenseTemplate([]string{"checking"}, []string{"rent"}, "Monthly apartment rent", 242_501, 247_599),
		)
	}
	if day == 3 {
		result = append(result, expenseTemplate([]string{"checking"}, []string{"subscription"}, "Monthly digital subscriptions", 4_201, 5_399))
	}
	if day == 5 {
		result = append(result, expenseTemplate([]string{"checking"}, []string{"insurance"}, "Monthly auto insurance premium", 13_901, 15_299))
	}
	if day == 7 {
		result = append(result, transferTemplate([]string{"checking"}, []string{"loan"}, "Monthly auto loan payment", 39_501, 42_499))
	}
	if day == 21 {
		result = append(result, transferTemplate([]string{"checking"}, []string{"saving"}, "Monthly transfer to savings", 75_001, 84_999))
	}
	anchor := time.Date(2026, time.April, 10, 0, 0, 0, 0, date.Location())
	dateUTC := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	anchorUTC := time.Date(anchor.Year(), anchor.Month(), anchor.Day(), 0, 0, 0, 0, time.UTC)
	daysFromAnchor := int(dateUTC.Sub(anchorUTC).Hours() / 24)
	if daysFromAnchor >= 0 && daysFromAnchor%14 == 0 {
		result = append(result, incomeTemplate([]string{"salary"}, []string{"checking"}, "Biweekly payroll deposit", 482_501, 487_599))
	}
	return result
}

func randomTemplate(rng *rand.Rand, weekday time.Weekday) transactionTemplate {
	roll := rng.Intn(100)
	if roll < 72 {
		expenses := []transactionTemplate{
			expenseTemplate(nil, []string{"grocer"}, "Grocery and household purchase", 1_201, 18_999),
			expenseTemplate(nil, []string{"dining"}, "Restaurant or coffee purchase", 501, 12_999),
			expenseTemplate(nil, []string{"gas", "transport"}, "Local transportation purchase", 701, 9_999),
			expenseTemplate(nil, []string{"shopping"}, "Retail purchase", 1_001, 24_999),
			expenseTemplate(nil, []string{"entertainment"}, "Entertainment purchase", 801, 14_999),
			expenseTemplate(nil, []string{"health", "fitness"}, "Health and fitness purchase", 1_001, 16_999),
		}
		if weekday == time.Saturday || weekday == time.Sunday {
			expenses = append(expenses, expenses[1], expenses[4])
		}
		return expenses[rng.Intn(len(expenses))]
	}
	if roll < 88 {
		return incomeTemplate(nil, []string{"checking", "saving"}, "Freelance project payment", 12_501, 125_999)
	}
	return transferTemplate([]string{"checking"}, []string{"saving"}, "Transfer between financial accounts", 2_501, 40_999)
}

func incomeTemplate(fromKeywords, toKeywords []string, note string, minCents, maxCents int64) transactionTemplate {
	return transactionTemplate{utils.TransactionTypeIncome, utils.AccountTypeIncome, utils.AccountTypeAsset, fromKeywords, toKeywords, note, minCents, maxCents}
}

func expenseTemplate(fromKeywords, toKeywords []string, note string, minCents, maxCents int64) transactionTemplate {
	return transactionTemplate{utils.TransactionTypeExpense, utils.AccountTypeAsset, utils.AccountTypeExpense, fromKeywords, toKeywords, note, minCents, maxCents}
}

func transferTemplate(fromKeywords, toKeywords []string, note string, minCents, maxCents int64) transactionTemplate {
	return transactionTemplate{utils.TransactionTypeTransfer, utils.AccountTypeAsset, 0, fromKeywords, toKeywords, note, minCents, maxCents}
}

func chooseAccount(rng *rand.Rand, states []*accountState, accountType int, keywords []string, exclude uuid.UUID) *accountState {
	candidates := filterAccounts(states, accountType, keywords, exclude)
	if len(candidates) == 0 && len(keywords) > 0 {
		candidates = filterAccounts(states, accountType, nil, exclude)
	}
	if len(candidates) == 0 {
		return nil
	}
	return candidates[rng.Intn(len(candidates))]
}

func filterAccounts(states []*accountState, accountType int, keywords []string, exclude uuid.UUID) []*accountState {
	result := make([]*accountState, 0)
	for _, state := range states {
		if state.account.Id == exclude || state.account.IsGroup || state.account.DeletedAt != nil {
			continue
		}
		if accountType == 0 {
			if state.account.Type != utils.AccountTypeAsset && state.account.Type != utils.AccountTypeLiability {
				continue
			}
		} else if state.account.Type != accountType {
			continue
		}
		if len(keywords) > 0 && !containsAny(strings.ToLower(state.account.Name), keywords) {
			continue
		}
		result = append(result, state)
	}
	return result
}

func containsAny(value string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(value, keyword) {
			return true
		}
	}
	return false
}

func amountFromRange(rng *rand.Rand, minimumCents, maximumCents int64) decimal.Decimal {
	cents := minimumCents
	if maximumCents > minimumCents {
		cents += rng.Int63n(maximumCents - minimumCents + 1)
	}
	if cents%100 == 0 {
		if cents < maximumCents {
			cents++
		} else {
			cents--
		}
	}
	return decimal.NewFromInt(cents).Shift(-2)
}

func fitToAvailableBalance(amount decimal.Decimal, from *accountState) (decimal.Decimal, bool) {
	if from.account.Type != utils.AccountTypeAsset && from.account.Type != utils.AccountTypeLiability {
		return amount, true
	}
	if !from.balance.IsPositive() {
		return decimal.Zero, false
	}
	if amount.GreaterThan(from.balance) {
		cents := from.balance.Mul(decimal.NewFromInt(100)).Floor().IntPart()
		if cents <= 0 {
			return decimal.Zero, false
		}
		if cents%100 == 0 {
			cents--
		}
		if cents <= 0 {
			return decimal.Zero, false
		}
		amount = decimal.NewFromInt(cents).Shift(-2)
	}
	if amount.Mul(decimal.NewFromInt(100)).IntPart()%100 == 0 {
		return decimal.Zero, false
	}
	return amount, amount.IsPositive()
}

func applySimulation(from, to *accountState, amount decimal.Decimal) {
	from.balance = from.balance.Sub(amount)
	to.balance = to.balance.Add(amount)
}

func decimalToMoney(amount decimal.Decimal) (int64, int) {
	units := amount.Truncate(0).IntPart()
	nanos := amount.Sub(decimal.NewFromInt(units)).Mul(decimal.NewFromInt(1_000_000_000)).IntPart()
	return units, int(nanos)
}

func randomTransactionTime(rng *rand.Rand, businessDate time.Time, location *time.Location) time.Time {
	hour := 7 + rng.Intn(15)
	minute := rng.Intn(60)
	second := rng.Intn(60)
	return time.Date(businessDate.Year(), businessDate.Month(), businessDate.Day(), hour, minute, second, 0, location)
}
