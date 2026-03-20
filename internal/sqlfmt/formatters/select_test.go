package formatters

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/tkcrm/pgxgen/internal/sqlfmt/lexer"
)

func TestFormatSelect(t *testing.T) {
	options := DefaultOptions()
	tests := []struct {
		name        string
		tokenSource []Formatter
		want        string
	}{
		{
			name: "normal case",
			tokenSource: []Formatter{
				Token{Options: options, Token: lexer.Token{Type: lexer.SELECT, Value: "SELECT"}},
				Token{Options: options, Token: lexer.Token{Type: lexer.IDENT, Value: "name"}},
				Token{Options: options, Token: lexer.Token{Type: lexer.COMMA, Value: ","}},
				Token{Options: options, Token: lexer.Token{Type: lexer.IDENT, Value: "age"}},
			},
			want: "\nSELECT\n  name,\n  age",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			el := &Select{Options: options, Elements: tt.tokenSource}

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

func TestIncrementIndent(t *testing.T) {
	options := DefaultOptions()
	s := &Select{
		Options:     options,
		Elements:    nil,
		IndentLevel: 0,
	}
	s.AddIndent(1)
	got := s.IndentLevel
	want := 1
	if got != want {
		t.Errorf("want %#v got %#v", want, got)
	}
}
