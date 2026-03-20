package formatters

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/tkcrm/pgxgen/internal/sqlfmt/lexer"
)

func TestFormatGeneric(t *testing.T) {
	options := DefaultOptions()
	tests := []struct {
		name        string
		tokenSource []Formatter
		want        string
	}{
		{
			name: "normal case",
			tokenSource: []Formatter{
				Token{Options: options, Token: lexer.Token{Type: lexer.CREATE, Value: "CREATE"}},
				Token{Options: options, Token: lexer.Token{Type: lexer.TABLE, Value: "TABLE"}},
				Token{Options: options, Token: lexer.Token{Type: lexer.IDENT, Value: "something1"}},
			},
			want: "CREATE TABLE something1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			el := &Generic{Options: options, Elements: tt.tokenSource}

			_ = el.Format(buf, nil, 0)
			got := buf.String()
			if tt.want != got {
				t.Errorf("\n=======================\n=== WANT =============>\n%s\n=======================\n=== GOT ==============>\n%s\n=======================", tt.want, got)
			} else {
				fmt.Println(fmt.Sprintf("%s\n%s", got, "========================================================================"))
			}
		})
	}
}
