package formatters

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/tkcrm/pgxgen/internal/sqlfmt/lexer"
)

func TestFormatJoin(t *testing.T) {
	options := DefaultOptions()
	tests := []struct {
		name        string
		tokenSource []Formatter
		want        string
	}{
		{
			name: "normalcase",
			tokenSource: []Formatter{
				Token{Options: options, Token: lexer.Token{Type: lexer.LEFT, Value: "LEFT"}},
				Token{Options: options, Token: lexer.Token{Type: lexer.OUTER, Value: "OUTER"}},
				Token{Options: options, Token: lexer.Token{Type: lexer.JOIN, Value: "JOIN"}},
				Token{Options: options, Token: lexer.Token{Type: lexer.IDENT, Value: "sometable"}},
				Token{Options: options, Token: lexer.Token{Type: lexer.ON, Value: "ON"}},
				Token{Options: options, Token: lexer.Token{Type: lexer.IDENT, Value: "status1"}},
				Token{Options: options, Token: lexer.Token{Type: lexer.IDENT, Value: "="}},
				Token{Options: options, Token: lexer.Token{Type: lexer.IDENT, Value: "status2"}},
			},

			want: "\nLEFT OUTER JOIN sometable ON status1 = status2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			el := &Join{Options: options, Elements: tt.tokenSource}

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
