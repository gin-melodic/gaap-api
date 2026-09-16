package transaction

import "testing"

func TestEndDateFilterBoundary(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantOp  string
		wantVal string
	}{
		{
			name:    "plain date becomes exclusive next-midnight boundary",
			input:   "2026-09-04",
			wantOp:  " < ",
			wantVal: "2026-09-05 00:00:00",
		},
		{
			name:    "year boundary plain date rolls over correctly",
			input:   "2026-12-31",
			wantOp:  " < ",
			wantVal: "2027-01-01 00:00:00",
		},
		{
			name:    "leap day plain date stays within February",
			input:   "2028-02-29",
			wantOp:  " < ",
			wantVal: "2028-03-01 00:00:00",
		},
		{
			name:    "date with time of day keeps inclusive comparison",
			input:   "2026-09-04T18:30:00",
			wantOp:  "<=",
			wantVal: "2026-09-04T18:30:00",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op, val := endDateFilter(tt.input)
			if op != tt.wantOp || val != tt.wantVal {
				t.Fatalf("endDateFilter(%q) = (%q, %q), want (%q, %q)", tt.input, op, val, tt.wantOp, tt.wantVal)
			}
		})
	}
}
