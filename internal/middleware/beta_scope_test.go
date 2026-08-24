package middleware

import "testing"

func TestDeferredBetaPaths(t *testing.T) {
	tests := map[string]bool{
		"/v1/auth/login":                      false,
		"/v1/auth/update-password":            true,
		"/v1/config/list-currencies":          false,
		"/v1/config/add-currency":             true,
		"/v1/account/create-account":          false,
		"/v1/transaction/create-transaction":  false,
		"/v1/dashboard/get-dashboard-summary": false,
		"/v1/task/list-tasks":                 true,
		"/v1/data/export-data":                true,
		"/v1/user/update-profile":             true,
	}

	for path, expected := range tests {
		if actual := isDeferredBetaPath(path); actual != expected {
			t.Fatalf("isDeferredBetaPath(%q) = %t, want %t", path, actual, expected)
		}
	}
}
