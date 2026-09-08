package middleware

import "testing"

func TestDeferredBetaPaths(t *testing.T) {
	tests := map[string]bool{
		"/v1/auth/login":                      false,
		"/v1/auth/update-password":            true,
		"/v1/config/list-currencies":          false,
		"/v1/config/add-currency":             false,
		"/v1/config/delete-currency":          false,
		"/v1/config/get-exchange-rates":       false,
		"/v1/config/set-exchange-rate":        false,
		"/v1/account/create-account":          false,
		"/v1/transaction/create-transaction":  false,
		"/v1/dashboard/get-dashboard-summary": false,
		"/v1/task/list-tasks":                 true,
		"/v1/data/export-data":                true,
		// DEF-029: update-profile carries the base-currency switch and must work in the beta
		// runtime; only the deferred theme preference update stays 404.
		"/v1/user/update-profile": false,
		"/v1/user/update-theme":   true,
	}

	for path, expected := range tests {
		if actual := isDeferredBetaPath(path); actual != expected {
			t.Fatalf("isDeferredBetaPath(%q) = %t, want %t", path, actual, expected)
		}
	}
}
