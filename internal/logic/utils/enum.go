package utils

type AccountType = int

const (
	AccountTypeUnspecified AccountType = iota
	AccountTypeAsset
	AccountTypeLiability
	AccountTypeIncome
	AccountTypeExpense
	AccountTypeEquity
)

type TransactionType = int

const (
	TransactionTypeUnspecified TransactionType = iota
	TransactionTypeIncome
	TransactionTypeExpense
	TransactionTypeTransfer
	TransactionTypeOpeningBalance
)

type UserLevel = int

const (
	UserLevelUnspecified UserLevel = iota
	UserLevelFree
	UserLevelPro
)
