package sqlfmt

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"github.com/tkcrm/pgxgen/internal/sqlfmt/formatters"
	"github.com/tkcrm/pgxgen/internal/sqlfmt/lexer"
	"github.com/tkcrm/pgxgen/internal/sqlfmt/parser"
)

// Format formats a single SQL statement using the given options.
func Format(sql string, options *formatters.Options) (string, error) {
	if options == nil {
		options = formatters.DefaultOptions()
	}

	// Tokenize SQL query string
	tokens, errTokenize := lexer.Tokenize(sql, options.Dialect)
	if errTokenize != nil {
		return "", fmt.Errorf("tokenization error: %w", errTokenize)
	}

	// Parse tokens and group them into a sequence of query segments
	tokensParsed, errParse := parser.Parse(tokens, options)
	if errParse != nil {
		return "", fmt.Errorf("parse error: %w", errParse)
	}

	// Format parsed tokens into prettified and uniformly formatted SQL string
	var buf bytes.Buffer
	for i, tokenParsed := range tokensParsed {
		if err := tokenParsed.Format(&buf, tokensParsed, i); err != nil {
			return "", err
		}
	}

	// Get formatted SQL string
	sqlFormatted := strings.Trim(buf.String(), "\n")

	// Add left spacing if desired
	if options.Padding != "" {
		sqlFormatted = addPadding(sqlFormatted, options.Padding)
	}

	// Safety check, compare if formatted query still has the same logic as input
	if !CompareSemantic(sql, sqlFormatted) {
		return "", fmt.Errorf("formatted result does not match input semantically")
	}

	return sqlFormatted, nil
}

// FormatFile formats a SQL file containing multiple statements separated by semicolons.
// It preserves comments between statements and returns the formatted file content.
func FormatFile(src []byte, options *formatters.Options) ([]byte, error) {
	if options == nil {
		options = formatters.DefaultOptions()
	}

	content := string(src)
	if strings.TrimSpace(content) == "" {
		return src, nil
	}

	// Split file into statements by semicolons, preserving comments
	statements := splitStatements(content)

	var results []string
	for _, stmt := range statements {
		trimmed := strings.TrimSpace(stmt)
		if trimmed == "" {
			continue
		}

		// Check if this is a comment-only block
		if isCommentOnly(trimmed) {
			results = append(results, trimmed)
			continue
		}

		formatted, err := Format(trimmed, options)
		if err != nil {
			return nil, fmt.Errorf("format statement error: %w\nstatement: %s", err, trimmed)
		}
		results = append(results, formatted)
	}

	if len(results) == 0 {
		return src, nil
	}

	// Join statements with semicolons and double newlines.
	// Comments are followed by a single newline (they attach to the next statement).
	var buf bytes.Buffer
	for i, r := range results {
		buf.WriteString(r)
		if !isCommentOnly(strings.TrimSpace(r)) {
			buf.WriteString(";")
		}
		if i < len(results)-1 {
			if isCommentOnly(strings.TrimSpace(r)) {
				buf.WriteString("\n")
			} else {
				buf.WriteString("\n\n")
			}
		}
	}
	buf.WriteString("\n")

	return buf.Bytes(), nil
}

// splitStatements splits SQL content by semicolons while respecting strings, comments,
// and dollar-quoted strings.
func splitStatements(content string) []string {
	var statements []string
	var current bytes.Buffer
	runes := []rune(content)
	i := 0

	for i < len(runes) {
		ch := runes[i]

		switch {
		// Single-quoted string
		case ch == '\'':
			current.WriteRune(ch)
			i++
			for i < len(runes) {
				current.WriteRune(runes[i])
				if runes[i] == '\'' {
					i++
					// Handle escaped quotes ''
					if i < len(runes) && runes[i] == '\'' {
						current.WriteRune(runes[i])
						i++
						continue
					}
					break
				}
				i++
			}

		// Dollar-quoted string (PostgreSQL)
		case ch == '$' && i+1 < len(runes):
			tag := readDollarTagFromRunes(runes, i)
			if tag != "" {
				current.WriteString(tag)
				i += len([]rune(tag))
				// Read until closing tag
				for i < len(runes) {
					current.WriteRune(runes[i])
					if runes[i] == '$' && strings.HasSuffix(current.String(), tag) {
						i++
						break
					}
					i++
				}
			} else {
				current.WriteRune(ch)
				i++
			}

		// Single-line comment
		case ch == '-' && i+1 < len(runes) && runes[i+1] == '-':
			for i < len(runes) && runes[i] != '\n' {
				current.WriteRune(runes[i])
				i++
			}

		// Multi-line comment
		case ch == '/' && i+1 < len(runes) && runes[i+1] == '*':
			current.WriteRune(ch)
			i++
			for i < len(runes) {
				current.WriteRune(runes[i])
				if runes[i] == '/' && i > 0 && runes[i-1] == '*' {
					i++
					break
				}
				i++
			}

		// Statement separator
		case ch == ';':
			stmt := current.String()
			if strings.TrimSpace(stmt) != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
			i++

		default:
			current.WriteRune(ch)
			i++
		}
	}

	// Add remaining content
	if strings.TrimSpace(current.String()) != "" {
		statements = append(statements, current.String())
	}

	return statements
}

// readDollarTagFromRunes reads a dollar-quote tag starting at position i.
// Returns the full tag (e.g. "$$" or "$tag$") or empty string if not a valid tag.
func readDollarTagFromRunes(runes []rune, i int) string {
	if runes[i] != '$' {
		return ""
	}
	j := i + 1
	// Check for $$ (empty tag)
	if j < len(runes) && runes[j] == '$' {
		return "$$"
	}
	// Read tag name
	for j < len(runes) {
		ch := runes[j]
		if ch == '$' {
			return string(runes[i : j+1])
		}
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_') {
			return ""
		}
		j++
	}
	return ""
}

// isCommentOnly checks if the string contains only comments (no SQL statements).
func isCommentOnly(s string) bool {
	inBlock := false
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Inside a block comment — check for closing
		if inBlock {
			if strings.Contains(trimmed, "*/") {
				inBlock = false
			}
			continue
		}

		// Single-line comments
		if strings.HasPrefix(trimmed, "--") || strings.HasPrefix(trimmed, "//") {
			continue
		}

		// Block comment start
		if strings.HasPrefix(trimmed, "/*") {
			if !strings.Contains(trimmed, "*/") {
				inBlock = true
				continue
			}
			// Block comment on single line — check if there's content after */
			afterClose := trimmed[strings.Index(trimmed, "*/")+2:]
			if strings.TrimSpace(afterClose) == "" {
				continue
			}
			return false
		}

		return false
	}
	return true
}

// CompareSemantic compares a formatted SQL string with the original input and checks whether they are
// logically still the same.
func CompareSemantic(sql string, formattedSql string) bool {
	before := removeSymbols(removeComments(sql))
	after := removeSymbols(removeComments(formattedSql))
	return strings.Compare(before, after) == 0
}

// addPadding adds desired left-side padding to each line of the string
func addPadding(s string, leftPadding string) string {
	var result []string
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		result = append(result, fmt.Sprintf("%s%s", leftPadding, scanner.Text()))
	}
	return strings.Join(result, "\n")
}

// removeComments removes one-line comments to make sure their formatting did not manipulate semantics.
func removeComments(str string) string {
	var strNew string
	var quoted bool
	var skip bool
	for i, c := range str {
		if !quoted && c == '\'' {
			quoted = true
			strNew += string(c)
			continue
		} else if quoted && c == '\'' {
			quoted = false
			strNew += string(c)
			continue
		} else if quoted {
			strNew += string(c)
			continue
		}

		if !skip {
			if c == '/' && len(str) > i+1 && str[i+1] == '/' {
				skip = true
				continue
			}
			if c == '-' && len(str) > i+1 && str[i+1] == '-' {
				skip = true
				continue
			}
		}

		if skip && c == '\n' {
			skip = false
		}

		if !skip {
			strNew += string(c)
		}
	}
	return strNew
}

// removeSymbols removes semantically unnecessary characters for comparison.
func removeSymbols(s string) string {
	var result []rune
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\f' || r == '\t' || r == ' ' || r == '　' {
			continue
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}
