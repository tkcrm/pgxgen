package formatters

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/tkcrm/pgxgen/internal/sqlfmt/lexer"
)

func TestFormatOrderBy(t *testing.T) {
	options := DefaultOptions()
	tests := []struct {
		name        string
		tokenSource []Formatter
		want        string
	}{
		{
			name: "normalcase",
			tokenSource: []Formatter{
				Token{Options: options, Token: lexer.Token{Type: lexer.ORDER, Value: "ORDER"}},
				Token{Options: options, Token: lexer.Token{Type: lexer.BY, Value: "BY"}},
				Token{Options: options, Token: lexer.Token{Type: lexer.IDENT, Value: "xxxxxx"}},
			},
			want: "\nORDER BY xxxxxx",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			el := &OrderBy{Options: options, Elements: tt.tokenSource}

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
