package parser

import (
	"fmt"

	"github.com/tkcrm/pgxgen/internal/sqlfmt/formatters"
	"github.com/tkcrm/pgxgen/internal/sqlfmt/lexer"
)

const joinStartRange = 3

// Parse parses a sequence of tokens returning a logically grouped slice of Formatters.
// Each Formatter is a logical segment of an SQL query. It may also be a group of such.
func Parse(tokens []lexer.Token, options *formatters.Options) ([]formatters.Formatter, error) {
	// Prepare parser for segment
	parser, errParser := NewParser(tokens, options)
	if errParser != nil {
		return nil, errParser
	}

	// Parse tokens
	result, errParse := parser.Parse()
	if errParse != nil {
		return nil, errParse
	}

	// Return process result
	return result, nil
}

// Parser initiates with a sequence of lexer tokens to be processed by the Parse() function.
// Furthermore, it holds all necessary state variables and a set of options to be assigned to each Formatter.
type Parser struct {
	options *formatters.Options // options used later to format element

	tokens   []lexer.Token
	endTypes []lexer.TokenType

	result []formatters.Formatter
}

// NewParser initializes a Parser with a sequence of lexer tokens representing an SQL query.
func NewParser(tokens []lexer.Token, options *formatters.Options) (*Parser, error) {
	// Use default options if none are passed
	if options == nil {
		options = formatters.DefaultOptions()
	}

	// Create initial Parser with according type, tokens and end types
	firstTokenType := tokens[0].Type
	switch firstTokenType {
	case lexer.COMMENT: // Just in case SQL query starts with comment. Otherwise, comment wouldn't be first token.
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfComment}, nil
	case lexer.SELECT:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfSelect}, nil
	case lexer.FROM:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfFrom}, nil
	case lexer.CASE:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfCase}, nil
	case lexer.JOIN, lexer.INNER, lexer.OUTER, lexer.LEFT, lexer.RIGHT, lexer.NATURAL, lexer.CROSS:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfJoin}, nil
	case lexer.WHERE:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfWhere}, nil
	case lexer.AND:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfAnd}, nil
	case lexer.OR:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfOr}, nil
	case lexer.GROUP:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfGroupBy}, nil
	case lexer.HAVING:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfHaving}, nil
	case lexer.ORDER:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfOrderBy}, nil
	case lexer.LIMIT, lexer.FETCH, lexer.OFFSET:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfLimitClause}, nil
	case lexer.STARTPARENTHESIS:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfParenthesis}, nil
	case lexer.UNION, lexer.INTERSECT, lexer.EXCEPT:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfTieClause}, nil
	case lexer.UPDATE:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfUpdate}, nil
	case lexer.SET:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfSet}, nil
	case lexer.RETURNING:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfReturning}, nil
	case lexer.CREATE:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfCreate}, nil
	case lexer.ALTER:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfAlter}, nil
	case lexer.DELETE:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfDelete}, nil
	case lexer.DROP:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfDrop}, nil
	case lexer.INSERT:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfInsert}, nil
	case lexer.VALUES:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfValues}, nil
	case lexer.FUNCTION:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfFunction}, nil
	case lexer.TYPE:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfType}, nil
	case lexer.LOCK:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfLock}, nil
	case lexer.WITH:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfWith}, nil
	case lexer.SHOW:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfShow}, nil
	case lexer.DISCARD:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfDiscard}, nil
	case lexer.BEGIN:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfBegin}, nil
	case lexer.SAVEPOINT:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfSavepoint}, nil
	case lexer.RELEASE:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfSavepoint}, nil
	case lexer.ROLLBACK:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfRollback}, nil
	case lexer.COMMIT:
		return &Parser{options: options, tokens: tokens, endTypes: lexer.EndOfCommit}, nil
	default:
		return nil, fmt.Errorf("invalid start token '%s'", tokens[0].Value)
	}
}

// Parse wraps parseSegment() and loops until EOF is reached. The initial sequence of tokens is usually
// comprised out of multiple segments (SELECT, FROM, WHERE,...). Parse() loops until EOF to make sure all
// segments are processed.
func (r *Parser) Parse() ([]formatters.Formatter, error) {
	// Prepare process variable
	var offset int

	// Verify that there is an EOF token at the end
	if r.tokens[len(r.tokens)-1].Type != lexer.EOF {
		return nil, fmt.Errorf("missing EOF token")
	}

	// Iterate and process segments until EOF is reached. Sequential segments are processed by this loop.
	// Nested segments are recursively processed by parseSegment().
	for r.tokens[offset].Type != lexer.EOF {

		// Prepare parser for segment
		segmentParser, errSegmentParser := NewParser(r.tokens[offset:], r.options)
		if errSegmentParser != nil {
			return nil, errSegmentParser
		}

		// Parse segment
		idxEndSegment, errSegment := segmentParser.parseSegment()
		if errSegment != nil {
			return nil, errSegment
		}

		// Retrieve segment result
		segmentFormatter := segmentParser.buildFormatter()

		// Append segment result to total result
		r.result = append(r.result, segmentFormatter)

		// Increment offset counter to proceed with next segment, if available
		switch r.tokens[offset].Type {
		case lexer.STARTPARENTHESIS, lexer.CASE, lexer.FUNCTION, lexer.TYPE:
			offset += idxEndSegment + 1 // Some types have end tags, e.g. "END" closing "CASE" or ")" closing "(". Next token starts after them.
		default:
			offset += idxEndSegment
		}
	}

	// Return process result
	return r.result, nil
}

// parseSegment iterates a token segment, creates according Formatters and appends them to the final result.
// It iterates until a suitable end token type (depending on the segment's initial token type) could be found.
// Whenever an intermediate subsequence (certain token type) is detected, a new Parser is initialized
// (recursively) processing that subsequence and appending it as a Formatter group. The same way subsequences
// can be nested arbitrarily, Parsers will be nested equally. A Parser may fork a Parser for a
// subsequence, and so on. The nested Parser results are aggregated by their parent Parser as they are yielded.
func (r *Parser) parseSegment() (int, error) {
	// Prepare process variables
	var (
		idx          int
		tokenCurrent lexer.Token
	)

	// Iterate over all token
	for {

		// Abort if no end token could be found and to prevent out-of-bound panics. Query might not be valid SQL.
		if idx >= len(r.tokens) {
			return 0, fmt.Errorf("could not find end token for '%s' token sequence", r.tokens[0].Value)
		}

		// Get reference of token to analyze
		tokenCurrent = r.tokens[idx]

		// Check for end token or new segment if current token is not first token
		if idx > 0 {

			// Check if token is end of segment
			if r.isEndToken(idx) {
				return idx, nil
			}

			// Check if token introduces query subsegment
			newSegment := r.isNewSegment(idx)
			if newSegment {

				// Create new parser for subsegment
				segmentParser, errSegmentParser := NewParser(r.tokens[idx:], r.options)
				if errSegmentParser != nil {
					return 0, fmt.Errorf("invalid segment: %s", errSegmentParser)
				}

				// Check if segment parser actually contains a suitable end token
				if !segmentParser.hasEndType() {
					return 0, fmt.Errorf("'%s' segment has no end keyword", tokenCurrent.Value)
				}

				// Parse subsegment
				idxEndSegment, errSegment := segmentParser.parseSegment()
				if errSegment != nil {
					return 0, errSegment
				}

				// Append subsegment result to parent segment result
				segmentFormatter := segmentParser.buildFormatter()
				if segmentFormatter == nil {
					return 0, fmt.Errorf("invalid segment result: %#v", segmentParser.result)
				}

				// Append subsegment result to parent segment result
				r.result = append(r.result, segmentFormatter)

				// Skip tokens that were processed as a subsegment parser
				switch tokenCurrent.Type {
				case lexer.STARTPARENTHESIS, lexer.CASE, lexer.FUNCTION, lexer.TYPE:
					idx += idxEndSegment + 1 // Some types have end tags, e.g. "END" closing "CASE" or ")" closing "(". Next token starts after them.
				default:
					idx += idxEndSegment
				}

				// Continue with next token
				continue
			}
		}

		// Append token to result
		r.result = append(r.result, formatters.Token{
			Options: r.options,
			Token: lexer.Token{
				Type:  tokenCurrent.Type,
				Value: tokenCurrent.Value,
			},
		})

		// Increase index to continue with next token
		idx++
	}
}

// hasEndType determines if the Parser's token sequence includes a suitable and expected end token type
func (r *Parser) hasEndType() bool {
	// Return true if there are no end types defined, meaning that anything is an end type
	if len(r.endTypes) == 0 {
		return true
	}

	// Check if end type is contained
	for _, token := range r.tokens {
		for _, ttype := range r.endTypes {
			if token.Type == ttype {
				return true
			}
		}
	}

	// Return false if end type is missing
	return false
}

// isEndToken determines if the token at index idx is an end token
func (r *Parser) isEndToken(idx int) bool {
	// Return true if there are no end types defined, meaning that anything is an end type
	if len(r.endTypes) == 0 {
		return true
	}

	// Get tokens to work with
	tokenFirst := r.tokens[0]
	tokenCurrent := r.tokens[idx]

	// Ignore usual token end type if within join clause
	for _, ttype := range lexer.TokenTypesOfJoinMaker {
		if tokenFirst.Type == ttype && idx < joinStartRange {
			return false
		}
	}

	// Check if token is end token
	for _, tokenEndType := range r.endTypes {
		if tokenCurrent.Type == tokenEndType || tokenCurrent.Type == lexer.EOF {
			return true
		}
	}
	return false
}

// isNewSegment checks whether the token at index idx indicates a new subsegment that needs to be handled
func (r *Parser) isNewSegment(idx int) bool {
	// Get tokens to work with
	tokenFirst := r.tokens[0]
	tokenCurrent := r.tokens[idx]
	tokenPrevious := r.tokens[idx-1] // isNewSegment() is only called if idx > 0
	tokenNext := r.tokens[idx+1]     // There will always be an EOF token at the end

	//
	// Negative indicators:
	//

	// Not a new segment, if just the parenthesis of a function call
	if tokenCurrent.Type == lexer.STARTPARENTHESIS && tokenPrevious.Type == lexer.FUNCTION {
		return false
	}

	// Not a new segment, if just the parenthesis of a type call
	if tokenCurrent.Type == lexer.STARTPARENTHESIS && tokenPrevious.Type == lexer.TYPE {
		return false
	}

	// Not a new segment, if type call, as indicated by subsequent parenthesis
	if tokenCurrent.Type == lexer.TYPE && tokenNext.Type != lexer.STARTPARENTHESIS {
		return false
	}

	// Not a new segment, if AND/OR within CASE
	if tokenFirst.Type == lexer.CASE && (tokenCurrent.Type == lexer.AND || tokenCurrent.Type == lexer.OR) {
		return false
	}

	// Not a new segment if WITH follows a TYPE token (e.g., TIMESTAMP WITH TIME ZONE)
	if tokenCurrent.Type == lexer.WITH && tokenPrevious.Type == lexer.TYPE {
		return false
	}

	//
	// Positive indicators:
	//

	// Check if token is indicating join clause.
	// However, ignore keyword if it appears in start of join group such as LEFT INNER JOIN.
	// In this case, "INNER" and "JOIN" are group keyword, but should not make subGroup
	for _, ttype := range lexer.TokenTypesOfJoinMaker {
		if tokenCurrent.Type == ttype && idx >= joinStartRange {
			return true
		}
	}

	// Check if token is of any type commonly indicating subsegment
	for _, v := range lexer.TokenTypesOfGroupMaker {
		if tokenCurrent.Type == v {
			// lexer.GROUP is only a group marker, if it is followed by lexer.BY
			if v == lexer.GROUP && tokenNext.Type != lexer.BY {
				return false
			} else {
				return true
			}
		}
	}

	// Return false as a fallback if no case for subsegment could be made
	return false
}

// buildFormatter creates a Formatter for the intermediate Parser subsegment, representing a
// segment of the SQL query, which can then be appended to the result sequence
func (r *Parser) buildFormatter() formatters.Formatter {
	// Get variables to work with
	elements := r.result
	firstElement, _ := elements[0].(formatters.Token)

	// Build suitable Formatter group and return it
	switch firstElement.Type {
	case lexer.COMMENT: // Just in case first element of SQL string is query.
		// Otherwise, comment is just a normal token within a series of elements of another formatter
		return &formatters.Token{
			Options: r.options, Token: lexer.Token{
				Type:  firstElement.Type,
				Value: firstElement.Value,
			},
		}
	case lexer.SELECT:
		return &formatters.Select{Options: r.options, Elements: elements}
	case lexer.FROM:
		return &formatters.From{Options: r.options, Elements: elements}
	case lexer.JOIN, lexer.INNER, lexer.OUTER, lexer.LEFT, lexer.RIGHT, lexer.NATURAL, lexer.CROSS:
		return &formatters.Join{Options: r.options, Elements: elements}
	case lexer.WHERE:
		return &formatters.Where{Options: r.options, Elements: elements}
	case lexer.AND:
		return &formatters.And{Options: r.options, Elements: elements}
	case lexer.OR:
		return &formatters.Or{Options: r.options, Elements: elements}
	case lexer.GROUP:
		return &formatters.GroupBy{Options: r.options, Elements: elements}
	case lexer.ORDER:
		return &formatters.OrderBy{Options: r.options, Elements: elements}
	case lexer.HAVING:
		return &formatters.Having{Options: r.options, Elements: elements}
	case lexer.LIMIT, lexer.OFFSET, lexer.FETCH:
		return &formatters.Limit{Options: r.options, Elements: elements}
	case lexer.UNION, lexer.INTERSECT, lexer.EXCEPT:
		return &formatters.TieGroup{Options: r.options, Elements: elements}
	case lexer.SET:
		return &formatters.Set{Options: r.options, Elements: elements}
	case lexer.RETURNING:
		return &formatters.Returning{Options: r.options, Elements: elements}
	case lexer.LOCK:
		return &formatters.Lock{Options: r.options, Elements: elements}
	case lexer.INSERT:
		return &formatters.Insert{Options: r.options, Elements: elements}
	case lexer.VALUES:
		return &formatters.Values{Options: r.options, Elements: elements}
	case lexer.WITH:
		return &formatters.With{Options: r.options, Elements: elements}
	case lexer.CASE:

		// End token of CASE group("END") has to be added to the group
		endToken := formatters.Token{Options: r.options, Token: lexer.Token{Type: lexer.END, Value: "END"}}
		elements = append(elements, endToken)
		return &formatters.Case{Options: r.options, Elements: elements}

	case lexer.STARTPARENTHESIS:

		// End token of sub query group (")") has to be added in the group
		endToken := formatters.Token{Options: r.options, Token: lexer.Token{Type: lexer.ENDPARENTHESIS, Value: ")"}}
		elements = append(elements, endToken)

		// Create subquery indenter if first keyword is SELECT or related keyword. Subqueries are not a
		// lot different to parenthesis groups, but this gives us additional information and format control
		switch elements[1].(type) {
		case *formatters.Select:
			return &formatters.Subquery{Options: r.options, Elements: elements}
		}

		// Return normal parenthesis group otherwise
		return &formatters.Parenthesis{Options: r.options, Elements: elements}

	case lexer.FUNCTION:

		// End token of function group (")") has to be added in the group
		endToken := formatters.Token{Options: r.options, Token: lexer.Token{Type: lexer.ENDPARENTHESIS, Value: ")"}}
		elements = append(elements, endToken)
		return &formatters.Function{Options: r.options, Elements: elements}

	case lexer.TYPE:

		// End token of TYPE group (")") has to be added in the group
		endToken := formatters.Token{Options: r.options, Token: lexer.Token{Type: lexer.ENDPARENTHESIS, Value: ")"}}
		elements = append(elements, endToken)
		return &formatters.Type{Options: r.options, Elements: elements}

	case lexer.CREATE, lexer.ALTER, lexer.UPDATE, lexer.DELETE, lexer.DROP,
		lexer.SHOW, lexer.DISCARD, lexer.BEGIN, lexer.SAVEPOINT, lexer.RELEASE, lexer.ROLLBACK, lexer.COMMIT:
		return &formatters.Generic{Options: r.options, Elements: elements}
	}

	// Return nil as no group could be built
	return nil
}
