package lexer

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

// Dialect represents the SQL dialect for tokenization.
type Dialect string

const (
	DialectPostgreSQL Dialect = "postgresql"
	DialectMySQL      Dialect = "mysql"
	DialectSQLite     Dialect = "sqlite"
)

// Tokenize sql string and returns slice of Token. Ignores Token of white-space, new-line and tab, as
// they have no semantic meaning.
func Tokenize(sql string, dialect Dialect) ([]Token, error) {
	// Prepare tokenizer
	t := &tokenizer{
		r:       bufio.NewReader(strings.NewReader(sql)),
		dialect: dialect,
	}

	// Execute tokenizer
	var tokens []Token
	for {

		// Get next token
		token, err := t.scan()
		if err != nil {
			return nil, fmt.Errorf("tokenizer error: %w", err)
		}

		// Abort loop at the end
		if token.Type == EOF {

			// Append EOF token to tokens, because parser will also run until EOF token
			tokens = append(tokens, token)

			// Return generated sequence of tokens
			return tokens, nil
		}

		// Skip empty formatting token
		if token.Type == WHITESPACE {
			continue
		}
		if token.Type == NEWLINE {
			continue
		}
		if token.Type == TAB {
			continue
		}

		// Append token to token slice
		tokens = append(tokens, token)
	}
}

// tokenizer holds a working buffer to process and defines functions to execute against it
type tokenizer struct {
	r       *bufio.Reader
	dialect Dialect
}

// scan reads the first character of the buffer and, depending on it, proceeds to read additional ones until a
// full token is detected and returns it
func (t *tokenizer) scan() (Token, error) {
	// Peek if next characters represent a valid comparator. If so, read the according amount of bytes
	// from the buffer and return comparator token
	if comparatorNext, _ := peekComparator(t.r); comparatorNext != "" {
		for i := len(comparatorNext); i > 0; i-- {
			_, _, _ = t.r.ReadRune()
		}
		return Token{Type: COMPARATOR, Value: comparatorNext}, nil
	}

	// Read first character from buffer
	ch, _, errCh := t.r.ReadRune()
	if errCh != nil {
		if errCh.Error() == "EOF" {
			return Token{Type: EOF, Value: "EOF"}, nil
		}
		return Token{}, errCh
	}

	// Prepare buffer for token and write read character to it
	var buf bytes.Buffer
	buf.WriteRune(ch)

	// Decide character, create and return token
	switch {
	case isWhitespace(ch):
		return Token{Type: WHITESPACE, Value: buf.String()}, nil

	case isNewline(ch):
		return Token{Type: NEWLINE, Value: buf.String()}, nil

	case isTab(ch):
		return Token{Type: TAB, Value: buf.String()}, nil

	case isPunctuation(ch):

		// Punctuation characters are only comprised out of a single character, except for DOBLECOLON tokens.
		// In case of a double colon, the next character needs to be read too.
		if isColon(ch) {

			// Read next character if there is one
			nextCh, _, errNext := t.r.ReadRune()
			if errNext != nil {
				if errNext.Error() == "EOF" { // Single colon was at the end of the string, which is okay
					return Token{Type: COLON, Value: buf.String()}, nil
				} else {
					return Token{}, errNext
				}
			}

			// Return colon or double colon token, depending on situation. Unread last character if necessary
			if isColon(nextCh) {
				return Token{Type: DOUBLECOLON, Value: fmt.Sprintf("%s%s", buf.String(), string(nextCh))}, nil
			} else {
				_ = t.r.UnreadRune() // Revert last read, because it belonged to the next token
				return Token{Type: COLON, Value: buf.String()}, nil
			}
		}

		// Lookup other token type in punctuation map
		if ttype, ok := punctuationMap[buf.String()]; ok {
			return Token{Type: ttype, Value: buf.String()}, nil
		}

		// Return with error in case of unexpected value
		return Token{}, fmt.Errorf("invalid punctuation value: %v", buf.String())

	case isSlash(ch) || isDash(ch):

		// Abort wrong comment indications
		if isSlash(ch) {
			if !t.peekSubsequent(isSlash) && !t.peekSubsequent(isAsterisk) {
				break
			}
		}
		if isDash(ch) && !t.peekSubsequent(isDash) {
			// Check for PostgreSQL JSON operators -> and ->>
			// Also handles already-formatted `- >` / `- >>` with whitespace
			if t.peekJsonArrow(&buf) {
				return Token{Type: COMPARATOR, Value: buf.String()}, nil
			}
			break
		}

		// Check if next character opens comment and read it
		comment, errComment := t.readComment(&buf, ch)
		if errComment != nil {
			return Token{}, errComment
		}

		// Return comment if one was read
		if comment != "" {
			return Token{Type: COMMENT, Value: comment}, nil
		}

	case isSingleQuote(ch):

		// Read subsequent characters until closing single quote
		for {
			chNext, _, errNext := t.r.ReadRune()
			if errNext != nil {
				if errNext.Error() == "EOF" {
					return Token{}, fmt.Errorf("unexpected EOF expected closing quote")
				} else {
					return Token{}, errNext
				}
			}

			// Append character to value
			buf.WriteRune(chNext)

			// Break loop once closing quote is found
			if isSingleQuote(chNext) {
				break
			}
		}

		// Return string token
		return Token{Type: STRING, Value: buf.String()}, nil

	case isBacktick(ch) && t.dialect == DialectMySQL:

		// Read subsequent characters until closing backtick (MySQL quoted identifier)
		for {
			chNext, _, errNext := t.r.ReadRune()
			if errNext != nil {
				if errNext.Error() == "EOF" {
					return Token{}, fmt.Errorf("unexpected EOF expected closing backtick")
				} else {
					return Token{}, errNext
				}
			}

			buf.WriteRune(chNext)

			if isBacktick(chNext) {
				break
			}
		}

		return Token{Type: IDENT, Value: buf.String()}, nil

	case isDollarSign(ch) && t.dialect == DialectPostgreSQL:

		// Check for dollar-quoted string: $tag$...$tag$ or $$...$$
		tag, err := t.readDollarTag(&buf)
		if err != nil {
			break // Not a dollar-quoted string, treat as regular token
		}
		if tag != "" {
			// Read until closing dollar tag
			body, errBody := t.readDollarBody(&buf, tag)
			if errBody != nil {
				return Token{}, errBody
			}
			_ = body
			return Token{Type: STRING, Value: buf.String()}, nil
		}
		// Single $ sign, fall through to regular token reading
	}

	// Read subsequent characters until value is complete
	comparator := ""
	var comparatorErr error
	for {

		// Stop if next character starts comparator sequence. But only if previous check didn't return
		// an invalid comparator sequence, otherwise an invalid comparator might turn into a valid one
		// after reading further bytes. For example, ~~~ might be understood as ~~. An input like 'a~~~1'
		// would be interpreted as 'a~ ~~1'.
		if comparatorErr == nil {
			comparator, comparatorErr = peekComparator(t.r)
			if comparator != "" {
				break // Nothing was read yet, no need to unread
			}
		}

		// Read next character
		chNext, _, errNext := t.r.ReadRune()
		if errNext != nil {
			if errNext.Error() == "EOF" {
				break
			} else {
				return Token{}, errNext
			}
		}

		// Stop if next character doesn't belong to the value anymore. Unread last unnecessary character.
		if isPunctuation(chNext) || isSingleQuote(chNext) || isWhitespace(chNext) || isNewline(chNext) || isTab(chNext) {
			_ = t.r.UnreadRune()
			break
		}

		// Stop before dash that starts a JSON operator (-> or ->>) or comment (--)
		if isDash(chNext) {
			_ = t.r.UnreadRune()
			break
		}

		// Append character to value
		buf.WriteRune(chNext)
	}

	// Prepare default lookup key and token value
	key := strings.ToUpper(buf.String())
	val := key

	// Sanitize key and value, if they include a target operator '.'.
	// If token value contains period, it's specifying a target, e.g. a table. Put that aside for the lookup.
	if strings.Contains(buf.String(), ".") {
		slice := strings.Split(buf.String(), ".")
		key = strings.ToUpper(slice[len(slice)-1])
		val = strings.Join(slice[:len(slice)-1], ".") + "." + key
	}

	// Check if value is function name, indicated by a lookup match and a subsequent parenthesis
	if ttype, ok := functionMap[key]; ok {
		if ttype == FUNCTION && t.peekSubsequent(isParenthesisStart) {
			return Token{Type: FUNCTION, Value: val}, nil
		}
	}

	// Check if value is keyword. Subsequent parenthesis would not indicate a function but a sub query.
	if ttype, ok := keywordMap[val]; ok { // Use val instead of key, because "." should not be splitted for keyword lookups!

		// Ambiguous edge case. Table name might (such as "user") might collide with Postgres'
		// parenthesis-less function "USER". To address ambiguity, Postgres clients must put "user" into
		// double quotes. However, in other databases this ambiguity does not exist, so clients would never
		// double quote in this situation. By default, these function keywords are enabled and handled as such.
		// Function keywords (e.g. CURRENT_DATE, USER) only exist in PostgreSQL.
		// In other dialects, treat them as regular identifiers.
		if ttype == FUNCTIONKEYWORD && t.dialect != DialectPostgreSQL {
			return Token{Type: IDENT, Value: buf.String()}, nil
		}

		// Return keyword token
		return Token{Type: ttype, Value: val}, nil // Return looked-up token type
	}

	// Return IDENT token type without any sanitization, since it's neither a keyword nor a function
	return Token{Type: IDENT, Value: buf.String()}, nil
}

// peekSubsequent looks into the subsequent characters searching for a certain follow-up character but
// reverts all read characters at the end.
func (t *tokenizer) peekSubsequent(isCharacter func(ch rune) bool) bool {
	// Unread character at the end
	defer func() {
		_ = t.r.UnreadRune()
	}()

	// Read character
	nextCh, _, errNext := t.r.ReadRune()
	if errNext != nil {
		return false
	}

	// Evaluate character or, if necessary, step into recursive call to check subsequent character
	if isCharacter(nextCh) {
		return true
	} else {
		return false
	}
}

// readComment looks into the subsequent characters to identify a comment start sequence and reads until its end
func (t *tokenizer) readComment(buf *bytes.Buffer, chPrev rune) (string, error) {
	// Check kind of comment
	singleLine := false
	if isSlash(chPrev) && t.peekSubsequent(isSlash) {
		singleLine = true // one-line comment //
	} else if isDash(chPrev) && t.peekSubsequent(isDash) {
		singleLine = true // one-line comment --
	} else if t.peekSubsequent(isAsterisk) {
		singleLine = false // multi-line comment /* ... */
	} else {
		return "", nil
	}

	// Read subsequent characters until closing single quote
	for {
		chNext, _, errNext := t.r.ReadRune()
		if errNext != nil {
			if singleLine && errNext.Error() == "EOF" {
				return buf.String(), nil
			} else {
				return buf.String(), errNext
			}
		}

		// Stop reading single-line comment at new line
		if singleLine && isNewline(chNext) {
			return buf.String(), nil
		}

		// Append character to value
		buf.WriteRune(chNext)

		// Stop reading multi-line comment at termination sequence
		if !singleLine && isAsterisk(chPrev) && isSlash(chNext) {
			return buf.String(), nil
		}

		// Remember last ch
		chPrev = chNext
	}
}

func isPunctuation(ch rune) bool {
	_, is := punctuationMap[string(ch)]
	return is
}

func isNewline(ch rune) bool {
	return ch == '\n' || ch == '\r' || ch == '\f'
}

func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '　'
}

func isTab(ch rune) bool {
	return ch == '\t'
}

func isColon(ch rune) bool {
	return ch == ':'
}

func isParenthesisStart(ch rune) bool {
	return ch == '('
}

func isSingleQuote(ch rune) bool {
	return ch == '\''
}

func isSlash(ch rune) bool {
	return ch == '/'
}

func isDash(ch rune) bool {
	return ch == '-'
}

func isAsterisk(ch rune) bool {
	return ch == '*'
}

func isGreaterThan(ch rune) bool {
	return ch == '>'
}

// peekJsonArrow checks if the upcoming characters form a JSON arrow operator (-> or ->>),
// optionally skipping whitespace between `-` and `>`. If found, writes the arrow chars
// to buf (which must already contain `-`) and returns true.
// Uses Peek to avoid consuming characters on failure.
func (t *tokenizer) peekJsonArrow(buf *bytes.Buffer) bool {
	// Peek ahead to find '>' past optional whitespace
	peekLen := 1
	for {
		b, err := t.r.Peek(peekLen)
		if err != nil {
			return false
		}
		ch := rune(b[peekLen-1])
		if isWhitespace(ch) || isTab(ch) {
			peekLen++
			continue
		}
		if ch == '>' {
			// Found '->' pattern. Consume all peeked bytes.
			for i := 0; i < peekLen; i++ {
				_, _, _ = t.r.ReadRune()
			}
			buf.WriteRune('>')
			// Check for ->> (double arrow)
			if t.peekSubsequent(isGreaterThan) {
				gt2, _, _ := t.r.ReadRune()
				buf.WriteRune(gt2)
			}
			return true
		}
		// Not '>' — don't consume anything
		return false
	}
}

func isBacktick(ch rune) bool {
	return ch == '`'
}

func isDollarSign(ch rune) bool {
	return ch == '$'
}

// readDollarTag reads a dollar-quote tag after the initial '$'. Returns the full tag (e.g. "$$" or "$tag$").
// If the sequence is not a valid dollar-quote opening, unreads consumed characters and returns an error.
func (t *tokenizer) readDollarTag(buf *bytes.Buffer) (string, error) {
	tag := "$"
	var consumed []rune
	for {
		ch, _, err := t.r.ReadRune()
		if err != nil {
			// EOF after $: write any consumed tag chars to buf so they aren't lost
			for _, r := range consumed {
				buf.WriteRune(r)
			}
			return "", fmt.Errorf("unexpected EOF in dollar tag")
		}
		consumed = append(consumed, ch)
		tag += string(ch)
		if ch == '$' {
			// Valid dollar-quote tag found, write consumed chars to buf
			for _, r := range consumed {
				buf.WriteRune(r)
			}
			return tag, nil
		}
		// Tag chars must be identifier-safe (letter, digit, underscore)
		if !isTagChar(ch) {
			// Not a valid tag — unread all consumed characters
			// We can only unread the last one via UnreadRune, so unread just the invalid char
			// and write the valid tag chars (if any) to buf
			_ = t.r.UnreadRune()
			// Write any valid tag chars that were consumed (between $ and the invalid char)
			for _, r := range consumed[:len(consumed)-1] {
				buf.WriteRune(r)
			}
			return "", fmt.Errorf("invalid dollar tag character")
		}
	}
}

// readDollarBody reads the body of a dollar-quoted string until the closing tag is found.
func (t *tokenizer) readDollarBody(buf *bytes.Buffer, tag string) (string, error) {
	var body bytes.Buffer
	for {
		ch, _, err := t.r.ReadRune()
		if err != nil {
			return "", fmt.Errorf("unexpected EOF in dollar-quoted string")
		}
		buf.WriteRune(ch)
		body.WriteRune(ch)

		// Check if body ends with the closing tag
		if ch == '$' && strings.HasSuffix(body.String(), tag) {
			return body.String(), nil
		}
	}
}

func isTagChar(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_'
}
