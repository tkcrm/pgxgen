package formatters

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/tkcrm/pgxgen/internal/sqlfmt/lexer"
)

func TestFormatUnion(t *testing.T) {
	options := DefaultOptions()
	tests := []struct {
		name        string
		tokenSource []Formatter
		want        string
	}{
		{
			name: "normal case1",
			tokenSource: []Formatter{
				Token{Options: options, Token: lexer.Token{Type: lexer.UNION, Value: "UNION"}},
				Token{Options: options, Token: lexer.Token{Type: lexer.ALL, Value: "ALL"}},
			},
			want: "\nUNION ALL",
		},
		{
			name: "normal case2",
			tokenSource: []Formatter{
				Token{Options: options, Token: lexer.Token{Type: lexer.INTERSECT, Value: "INTERSECT"}},
			},
			want: "\nINTERSECT",
		},
		{
			name: "normal case3",
			tokenSource: []Formatter{
				Token{Options: options, Token: lexer.Token{Type: lexer.EXCEPT, Value: "EXCEPT"}},
			},
			want: "\nEXCEPT",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			el := &TieGroup{Options: options, Elements: tt.tokenSource}

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
