package formatters

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/tkcrm/pgxgen/internal/sqlfmt/lexer"
)

func TestFormatSubquery(t *testing.T) {
	options := DefaultOptions()
	tests := []struct {
		name         string
		isColumnArea bool
		src          []Formatter
		want         string
	}{
		{
			name:         "normalcase column area",
			isColumnArea: true, // In column area all values are indented by one
			src: []Formatter{
				Token{Options: options, Token: lexer.Token{Type: lexer.STARTPARENTHESIS, Value: "("}},
				&Select{
					Options: options,
					Elements: []Formatter{
						Token{Options: options, Token: lexer.Token{Type: lexer.SELECT, Value: "SELECT"}},
						Token{Options: options, Token: lexer.Token{Type: lexer.IDENT, Value: "xxxxxx"}},
					},
					IndentLevel: 0,
				},
				&From{
					Options: options,
					Elements: []Formatter{
						Token{Options: options, Token: lexer.Token{Type: lexer.FROM, Value: "FROM"}},
						Token{Options: options, Token: lexer.Token{Type: lexer.IDENT, Value: "xxxxxx"}},
					},
					IndentLevel: 0,
				},
				Token{Options: options, Token: lexer.Token{Type: lexer.ENDPARENTHESIS, Value: ")"}},
			},
			want: "\n(\n  SELECT\n    xxxxxx\n  FROM xxxxxx\n)",
		},
		{
			name:         "normalcase outside column area",
			isColumnArea: false, // No additional indent outside of column area
			src: []Formatter{
				Token{Options: options, Token: lexer.Token{Type: lexer.STARTPARENTHESIS, Value: "("}},
				&Select{
					Options: options,
					Elements: []Formatter{
						Token{Options: options, Token: lexer.Token{Type: lexer.SELECT, Value: "SELECT"}},
						Token{Options: options, Token: lexer.Token{Type: lexer.IDENT, Value: "xxxxxx"}},
					},
					IndentLevel: 0,
				},
				&From{
					Options: options,
					Elements: []Formatter{
						Token{Options: options, Token: lexer.Token{Type: lexer.FROM, Value: "FROM"}},
						Token{Options: options, Token: lexer.Token{Type: lexer.IDENT, Value: "xxxxxx"}},
					},
					IndentLevel: 0,
				},
				Token{Options: options, Token: lexer.Token{Type: lexer.ENDPARENTHESIS, Value: ")"}},
			},
			want: " (\n  SELECT\n    xxxxxx\n  FROM xxxxxx\n)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			el := &Subquery{Options: options, Elements: tt.src, IndentLevel: 0}
			el.IsColumnArea = tt.isColumnArea

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
