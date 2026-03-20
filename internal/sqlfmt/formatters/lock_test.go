package formatters

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/tkcrm/pgxgen/internal/sqlfmt/lexer"
)

func TestFormatLock(t *testing.T) {
	options := DefaultOptions()
	tests := []struct {
		name        string
		tokenSource []Formatter
		want        string
	}{
		{
			name: "normalcase",
			tokenSource: []Formatter{
				Token{Options: options, Token: lexer.Token{Type: lexer.LOCK, Value: "LOCK"}},
				Token{Options: options, Token: lexer.Token{Type: lexer.IDENT, Value: "table"}},
				Token{Options: options, Token: lexer.Token{Type: lexer.IN, Value: "IN"}},
				Token{Options: options, Token: lexer.Token{Type: lexer.IDENT, Value: "xxx"}},
			},
			want: "\nLOCK table\nIN xxx",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			el := &Lock{Options: options, Elements: tt.tokenSource}

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
