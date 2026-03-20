package formatters

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/tkcrm/pgxgen/internal/sqlfmt/lexer"
)

func TestFormatParenthesis(t *testing.T) {
	options := DefaultOptions()
	tests := []struct {
		name string
		src  []Formatter
		want string
	}{
		{
			name: "normalcase",
			src: []Formatter{
				Token{Options: options, Token: lexer.Token{Type: lexer.STARTPARENTHESIS, Value: "("}},
				Token{Options: options, Token: lexer.Token{Type: lexer.IDENT, Value: "val1"}},
				Token{Options: options, Token: lexer.Token{Type: lexer.COMPARATOR, Value: ">"}},
				Token{Options: options, Token: lexer.Token{Type: lexer.IDENT, Value: "0"}},
				&And{
					Options: options,
					Elements: []Formatter{
						Token{Options: options, Token: lexer.Token{Type: lexer.AND, Value: "AND"}},
						Token{Options: options, Token: lexer.Token{Type: lexer.IDENT, Value: "val2"}},
						Token{Options: options, Token: lexer.Token{Type: lexer.COMPARATOR, Value: ">"}},
						Token{Options: options, Token: lexer.Token{Type: lexer.IDENT, Value: "0"}},
					},
					IndentLevel: 0,
				},
				Token{Options: options, Token: lexer.Token{Type: lexer.ENDPARENTHESIS, Value: ")"}},
			},
			want: " (\n  val1 > 0\n  AND val2 > 0\n)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			el := &Subquery{Options: options, Elements: tt.src, IndentLevel: 0}

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
