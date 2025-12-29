package boot

import (
	"strings"
)

// splitSqlStatements splits SQL content into statements by semicolon,
// respecting single and double quoted strings to avoid incorrect splitting.
func splitSqlStatements(content string) []string {
	var stmts []string
	var buf strings.Builder
	var inQuote bool
	var quoteChar rune

	for _, r := range content {
		if inQuote {
			if r == quoteChar {
				inQuote = false
			}
			buf.WriteRune(r)
		} else {
			if r == '\'' || r == '"' {
				inQuote = true
				quoteChar = r
				buf.WriteRune(r)
			} else if r == ';' {
				stmts = append(stmts, buf.String())
				buf.Reset()
			} else {
				buf.WriteRune(r)
			}
		}
	}
	if buf.Len() > 0 {
		stmts = append(stmts, buf.String())
	}
	return stmts
}
