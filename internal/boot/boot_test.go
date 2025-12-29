package boot

import (
	"reflect"
	"testing"
)

func Test_splitSqlStatements(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "Simple split",
			content: "SELECT 1; SELECT 2;",
			want:    []string{"SELECT 1", " SELECT 2"},
		},
		{
			name:    "Split with newline",
			content: "SELECT 1;\nSELECT 2;",
			want:    []string{"SELECT 1", "\nSELECT 2"},
		},
		{
			name:    "Semicolon in single quotes",
			content: "INSERT INTO t VALUES ('a;b');",
			want:    []string{"INSERT INTO t VALUES ('a;b')"},
		},
		{
			name:    "Semicolon in double quotes",
			content: `INSERT INTO t VALUES ("a;b");`,
			want:    []string{`INSERT INTO t VALUES ("a;b")`},
		},
		{
			name:    "Escaped quotes",
			content: "INSERT INTO t VALUES ('It''s ok');",
			want:    []string{"INSERT INTO t VALUES ('It''s ok')"},
		},
		{
			name:    "Mixed quotes",
			content: `SELECT "col" FROM t WHERE val = 'val;ue';`,
			want:    []string{`SELECT "col" FROM t WHERE val = 'val;ue'`},
		},
		{
			name:    "Multiple statements with quotes",
			content: "SELECT 'a;b'; SELECT 2;",
			want:    []string{"SELECT 'a;b'", " SELECT 2"},
		},
		{
			name:    "Trailing semicolon",
			content: "SELECT 1;",
			want:    []string{"SELECT 1"},
		},
		{
			name:    "No semicolon",
			content: "SELECT 1",
			want:    []string{"SELECT 1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := splitSqlStatements(tt.content); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitSqlStatements() = %v, want %v", got, tt.want)
			}
		})
	}
}
