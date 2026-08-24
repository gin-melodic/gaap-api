package middleware

import "testing"

func TestClassifyHandlerError(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantStatus int
		wantBody   string
	}{
		{name: "bad credentials", raw: "invalid email or password", wantStatus: 401, wantBody: "invalid email or password"},
		{name: "invite rejection", raw: "registration unavailable", wantStatus: 403, wantBody: "registration unavailable"},
		{name: "email length", raw: "email must not exceed 255 characters", wantStatus: 400, wantBody: "email must not exceed 255 characters"},
		{name: "password length", raw: "password must contain between 8 and 100 characters", wantStatus: 400, wantBody: "password must contain between 8 and 100 characters"},
		{name: "validation", raw: "amount must be greater than zero", wantStatus: 400, wantBody: "amount must be greater than zero"},
		{name: "not found", raw: "transaction not found", wantStatus: 404, wantBody: "transaction not found"},
		{name: "internal detail hidden", raw: "database connection refused", wantStatus: 500, wantBody: "internal server error"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status, body := classifyHandlerError(test.raw)
			if status != test.wantStatus || body != test.wantBody {
				t.Fatalf("classifyHandlerError(%q) = (%d, %q), want (%d, %q)", test.raw, status, body, test.wantStatus, test.wantBody)
			}
		})
	}
}
