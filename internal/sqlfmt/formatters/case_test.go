package formatters

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/tkcrm/pgxgen/internal/sqlfmt/lexer"
)

func TestFormatCase(t *testing.T) {
	options := DefaultOptions()
	tests := []struct {
		name        string
		tokenSource []Formatter
		want        string
	}{
		{
			name: "normal case",
			tokenSource: []Formatter{
				Token{Options: options, Token: lexer.Token{Type: lexer.CASE, Value: "CASE"}},
				Token{Options: options, Token: lexer.Token{Type: lexer.WHEN, Value: "WHEN"}},
				Token{Options: options, Token: lexer.Token{Type: lexer.IDENT, Value: "something"}},
				Token{Options: options, Token: lexer.Token{Type: lexer.IDENT, Value: "something"}},
				Token{Options: options, Token: lexer.Token{Type: lexer.END, Value: "END"}},
			},
			want: "\nCASE\n  WHEN something something\nEND",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			el := &Case{Options: options, Elements: tt.tokenSource}

			_ = el.Format(buf, nil, 0)
			got := buf.String()
			if tt.want != got {
				t.Errorf("\n=======================\n=== WANT =============>\n%s\n=======================\n=== GOT ==============>\n%s\n=======================", tt.want, got)
			} else {
				fmt.Printf("%s\n%s\n", got, "========================================================================")
			}
		})
	}
}
