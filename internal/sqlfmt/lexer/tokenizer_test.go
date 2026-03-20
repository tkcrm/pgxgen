package lexer

import (
	"bufio"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenize(t *testing.T) {
	testingSQLStatement := strings.Trim(`select name, age, sum, sum(case xxx) from users where name xxx and age = 'xxx' limit 100 except 100`, "`")
	want := []Token{
		{Type: SELECT, Value: "SELECT"},
		{Type: IDENT, Value: "name"},
		{Type: COMMA, Value: ","},
		{Type: IDENT, Value: "age"},
		{Type: COMMA, Value: ","},
		{Type: IDENT, Value: "sum"},
		{Type: COMMA, Value: ","},
		{Type: FUNCTION, Value: "SUM"},
		{Type: STARTPARENTHESIS, Value: "("},
		{Type: CASE, Value: "CASE"},
		{Type: IDENT, Value: "xxx"},
		{Type: ENDPARENTHESIS, Value: ")"},
		{Type: FROM, Value: "FROM"},
		{Type: IDENT, Value: "users"},
		{Type: WHERE, Value: "WHERE"},
		{Type: IDENT, Value: "name"},
		{Type: IDENT, Value: "xxx"},
		{Type: AND, Value: "AND"},
		{Type: IDENT, Value: "age"},
		{Type: COMPARATOR, Value: "="},
		{Type: STRING, Value: "'xxx'"},
		{Type: LIMIT, Value: "LIMIT"},
		{Type: IDENT, Value: "100"},
		{Type: EXCEPT, Value: "EXCEPT"},
		{Type: IDENT, Value: "100"},
		{Type: EOF, Value: "EOF"},
	}
	got, err := Tokenize(testingSQLStatement, DialectPostgreSQL)
	assert.Nil(t, err)
	assert.Equal(t, want, got)
}

func TestTokenizeDollarQuoting(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		dialect Dialect
		want    []Token
	}{
		{
			name:    "simple dollar-quoted string",
			sql:     "SELECT $$hello world$$",
			dialect: DialectPostgreSQL,
			want: []Token{
				{Type: SELECT, Value: "SELECT"},
				{Type: STRING, Value: "$$hello world$$"},
				{Type: EOF, Value: "EOF"},
			},
		},
		{
			name:    "tagged dollar-quoted string",
			sql:     "SELECT $tag$hello$tag$",
			dialect: DialectPostgreSQL,
			want: []Token{
				{Type: SELECT, Value: "SELECT"},
				{Type: STRING, Value: "$tag$hello$tag$"},
				{Type: EOF, Value: "EOF"},
			},
		},
		{
			name:    "dollar sign not a quote in non-postgres",
			sql:     "SELECT $1",
			dialect: DialectSQLite,
			want: []Token{
				{Type: SELECT, Value: "SELECT"},
				{Type: IDENT, Value: "$1"},
				{Type: EOF, Value: "EOF"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Tokenize(tt.sql, tt.dialect)
			assert.Nil(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTokenizeBacktickMySQL(t *testing.T) {
	sql := "SELECT `column_name` FROM `table_name`"
	want := []Token{
		{Type: SELECT, Value: "SELECT"},
		{Type: IDENT, Value: "`column_name`"},
		{Type: FROM, Value: "FROM"},
		{Type: IDENT, Value: "`table_name`"},
		{Type: EOF, Value: "EOF"},
	}
	got, err := Tokenize(sql, DialectMySQL)
	assert.Nil(t, err)
	assert.Equal(t, want, got)
}

func TestTokenizeFunctionKeywordsDialect(t *testing.T) {
	// In PostgreSQL, CURRENT_DATE is a FUNCTIONKEYWORD
	pgTokens, err := Tokenize("SELECT current_date", DialectPostgreSQL)
	assert.Nil(t, err)
	assert.Equal(t, FUNCTIONKEYWORD, pgTokens[1].Type)

	// In SQLite, CURRENT_DATE should be a regular IDENT
	sqliteTokens, err := Tokenize("SELECT current_date", DialectSQLite)
	assert.Nil(t, err)
	assert.Equal(t, IDENT, sqliteTokens[1].Type)
}

func Test_peekComparator(t *testing.T) {
	tests := []struct {
		testSequence   string
		wantComperator string
		wantError      assert.ErrorAssertionFunc
	}{
		{
			testSequence:   "~~42", // LIKE
			wantComperator: "~~",
			wantError:      assert.NoError,
		},
		{
			testSequence:   "~~*42", // ILIKE
			wantComperator: "~~*",
			wantError:      assert.NoError,
		},
		{
			testSequence:   "!~~42", // NOT LIKE
			wantComperator: "!~~",
			wantError:      assert.NoError,
		},
		{
			testSequence:   "!~~*42", // NOT ILIKE
			wantComperator: "!~~*",
			wantError:      assert.NoError,
		},
		{
			testSequence:   "=42", // EQUAL
			wantComperator: "=",
			wantError:      assert.NoError,
		},
		{
			testSequence:   ">42", // GREATER
			wantComperator: ">",
			wantError:      assert.NoError,
		},
		{
			testSequence:   "<42", // LOWER
			wantComperator: "<",
			wantError:      assert.NoError,
		},
		{
			testSequence:   ">=42", // GREATER OR EQUAL
			wantComperator: ">=",
			wantError:      assert.NoError,
		},
		{
			testSequence:   "<=42", // LOWER OR EQUAL
			wantComperator: "<=",
			wantError:      assert.NoError,
		},
		{
			testSequence:   "<>42", // NOT EQUAL
			wantComperator: "<>",
			wantError:      assert.NoError,
		},
		{
			testSequence:   "!=42", // NOT EQUAL
			wantComperator: "!=",
			wantError:      assert.NoError,
		},
		{
			testSequence:   "a!=42", // not STARTING with valid comparator but no error
			wantComperator: "",
			wantError:      assert.NoError,
		},
		{
			testSequence:   "!a!=42", // not STARTING with valid comparator
			wantComperator: "",
			wantError:      assert.Error,
		},
		{
			testSequence:   "!=!=42", // not STARTING with valid comparator
			wantComperator: "",
			wantError:      assert.Error,
		},
		{
			testSequence:   "~~~~~~~~~~*42", // not STARTING with valid comparator
			wantComperator: "",
			wantError:      assert.Error,
		},
		{
			testSequence:   "~~~~~~~~~~*~~42", // not STARTING with valid comparator
			wantComperator: "",
			wantError:      assert.Error,
		},
		{
			testSequence:   "!=a!=42", // starting with valid validator, although it's invalid later down the street
			wantComperator: "!=",
			wantError:      assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testSequence, func(t *testing.T) {
			// Fill test reader with input sequence
			r := bufio.NewReader(strings.NewReader(tt.testSequence))

			got, err := peekComparator(r)
			if !tt.wantError(t, err, fmt.Sprintf("peekComparator(%v)", r)) {
				return
			}
			assert.Equalf(t, tt.wantComperator, got, "peekComparator(%v)", r)

			// Check remaining characters in buffer
			remaining, _ := r.ReadString('\n')
			if len(remaining) != len(tt.testSequence) {
				t.Errorf("Invalid remaining buffer length. Want %d got %d", len(tt.testSequence), len(remaining))
				return
			}

			// Check if original string is reverted
			if remaining != tt.testSequence {
				t.Errorf("Invalid remaining buffer content. Want '%s' got '%s'", tt.testSequence, remaining)
				return
			}
		})
	}
}
