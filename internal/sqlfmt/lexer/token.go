package lexer

import (
	"bufio"
	"fmt"
	"strings"
)

// Token is a token struct
type Token struct {
	Type  TokenType
	Value string
}

// TokenType is an alias representing a kind of token
type TokenType int

// Token types
const (
	EOF TokenType = 1 + iota // eof
	WHITESPACE
	NEWLINE
	TAB
	COMMA
	COLON
	DOUBLECOLON
	COMMENT
	FUNCTION
	FUNCTIONKEYWORD
	STARTPARENTHESIS
	ENDPARENTHESIS
	STARTBRACKET
	ENDBRACKET
	STARTBRACE
	ENDBRACE
	COMPARATOR
	SURROUNDING
	TYPE
	IDENT  // field or table name
	STRING // values surrounded with single quotes
	UNION
	SELECT
	DISTINCT
	AS
	JOIN
	LEFT
	RIGHT
	INNER
	OUTER
	ON
	FROM
	WHERE
	EXISTS
	HAVING
	AND
	OR
	IN
	BETWEEN
	ANY
	ALL
	ARRAY
	IS
	LIKE
	ILIKE
	IF
	NOT
	NULL
	CASE
	WHEN
	THEN
	ELSE
	END
	ORDER
	GROUP
	BY
	DESC
	ASC
	LIMIT
	OVER
	RETURNING
	INSERT
	INTO
	UPDATE
	SET
	ALTER
	ADD
	RENAME
	MODIFY
	COLUMN
	TABLE
	DATABASE
	TO
	DELETE
	DROP
	CREATE
	DO
	VALUES
	FOR
	DISTINCTROW
	FILTER
	WITHIN
	COLLATE
	INTERSECT
	EXCEPT
	OFFSET
	FETCH
	FIRST
	ROWS
	USING
	OVERLAPS
	NATURAL
	CROSS
	TIME
	ZONE
	NULLS
	LAST
	AT
	LOCK
	WITH
	PRIMARY
	KEY

	SHOW
	DISCARD
	BEGIN
	SAVEPOINT
	RELEASE
	ROLLBACK
	COMMIT
)

// Define end keywords for each clause segment
var (
	EndOfSelect      = []TokenType{FROM, UNION, WHERE, ENDPARENTHESIS, EOF}
	EndOfCase        = []TokenType{END, EOF}
	EndOfFrom        = []TokenType{WHERE, INNER, OUTER, LEFT, RIGHT, JOIN, NATURAL, CROSS, ORDER, GROUP, UNION, OFFSET, LIMIT, FETCH, EXCEPT, INTERSECT, ENDPARENTHESIS, EOF}
	EndOfJoin        = []TokenType{WHERE, ORDER, GROUP, LIMIT, OFFSET, FETCH, LEFT, RIGHT, INNER, OUTER, NATURAL, CROSS, UNION, EXCEPT, INTERSECT, ENDPARENTHESIS, EOF}
	EndOfWhere       = []TokenType{GROUP, ORDER, LIMIT, OFFSET, FETCH, UNION, EXCEPT, INTERSECT, RETURNING, ENDPARENTHESIS, EOF}
	EndOfAnd         = []TokenType{GROUP, ORDER, LIMIT, OFFSET, FETCH, UNION, EXCEPT, INTERSECT, AND, OR, ENDPARENTHESIS, EOF}
	EndOfOr          = []TokenType{GROUP, ORDER, LIMIT, OFFSET, FETCH, UNION, EXCEPT, INTERSECT, AND, OR, ENDPARENTHESIS, EOF}
	EndOfGroupBy     = []TokenType{ORDER, LIMIT, FETCH, OFFSET, UNION, EXCEPT, INTERSECT, HAVING, ENDPARENTHESIS, EOF}
	EndOfHaving      = []TokenType{LIMIT, OFFSET, FETCH, ORDER, UNION, EXCEPT, INTERSECT, ENDPARENTHESIS, EOF}
	EndOfOrderBy     = []TokenType{LIMIT, FETCH, OFFSET, UNION, EXCEPT, INTERSECT, ENDPARENTHESIS, EOF}
	EndOfLimitClause = []TokenType{UNION, EXCEPT, INTERSECT, ENDPARENTHESIS, EOF}
	EndOfParenthesis = []TokenType{ENDPARENTHESIS, EOF}
	EndOfTieClause   = []TokenType{SELECT, STARTPARENTHESIS, EOF}
	EndOfUpdate      = []TokenType{WHERE, SET, RETURNING, EOF}
	EndOfSet         = []TokenType{FROM, WHERE, RETURNING, EOF}
	EndOfReturning   = []TokenType{EOF}
	EndOfCreate      = []TokenType{ENDPARENTHESIS, EOF}
	EndOfAlter       = []TokenType{ENDPARENTHESIS, EOF}
	EndOfAdd         = []TokenType{ENDPARENTHESIS, EOF}
	EndOfDelete      = []TokenType{ENDPARENTHESIS, EOF}
	EndOfDrop        = []TokenType{ENDPARENTHESIS, EOF}
	EndOfInsert      = []TokenType{SET, VALUES, EOF}
	EndOfValues      = []TokenType{UPDATE, RETURNING, EOF}
	EndOfType        = []TokenType{ENDPARENTHESIS, EOF}
	EndOfLock        = []TokenType{EOF}
	EndOfWith        = []TokenType{ENDPARENTHESIS, EOF}
	EndOfFunction    = []TokenType{ENDPARENTHESIS, EOF}
	EndOfShow        = []TokenType{ENDPARENTHESIS, EOF}
	EndOfDiscard     = []TokenType{ENDPARENTHESIS, EOF}
	EndOfBegin       = []TokenType{ENDPARENTHESIS, EOF}
	EndOfSavepoint   = []TokenType{ENDPARENTHESIS, EOF}
	EndOfRollback    = []TokenType{ENDPARENTHESIS, EOF}
	EndOfCommit      = []TokenType{ENDPARENTHESIS, EOF}
	EndOfComment     []TokenType // Empty slice means anything is end token
)

// Define keywords indicating certain segment groups
var (
	TokenTypesOfGroupMaker  = []TokenType{SELECT, CASE, FROM, WHERE, ORDER, GROUP, LIMIT, AND, OR, HAVING, UNION, EXCEPT, INTERSECT, FUNCTION, STARTPARENTHESIS, TYPE, WITH}
	TokenTypesOfJoinMaker   = []TokenType{JOIN, INNER, OUTER, LEFT, RIGHT, NATURAL, CROSS}
	TokenTypesOfTieClause   = []TokenType{UNION, INTERSECT, EXCEPT}
	TokenTypesOfLimitClause = []TokenType{LIMIT, FETCH, OFFSET}
)

var keywordMap = map[string]TokenType{
	"SELECT":      SELECT,
	"FROM":        FROM,
	"WHERE":       WHERE,
	"CASE":        CASE,
	"ORDER":       ORDER,
	"BY":          BY,
	"AS":          AS,
	"JOIN":        JOIN,
	"LEFT":        LEFT,
	"RIGHT":       RIGHT,
	"INNER":       INNER,
	"OUTER":       OUTER,
	"ON":          ON,
	"WHEN":        WHEN,
	"END":         END,
	"GROUP":       GROUP,
	"DESC":        DESC,
	"ASC":         ASC,
	"LIMIT":       LIMIT,
	"OVER":        OVER,
	"AND":         AND,
	"OR":          OR,
	"IN":          IN,
	"ANY":         ANY,
	"ARRAY":       ARRAY,
	"IS":          IS,
	"IF":          IF,
	"NOT":         NOT,
	"NULL":        NULL,
	"DISTINCT":    DISTINCT,
	"LIKE":        LIKE,
	"ILIKE":       ILIKE,
	"BETWEEN":     BETWEEN,
	"UNION":       UNION,
	"ALL":         ALL,
	"HAVING":      HAVING,
	"EXISTS":      EXISTS,
	"UPDATE":      UPDATE,
	"SET":         SET,
	"RETURNING":   RETURNING,
	"CREATE":      CREATE,
	"ALTER":       ALTER,
	"ADD":         ADD,
	"RENAME":      RENAME,
	"MODIFY":      MODIFY,
	"COLUMN":      COLUMN,
	"TABLE":       TABLE,
	"DATABASE":    DATABASE,
	"TO":          TO,
	"DROP":        DROP,
	"DELETE":      DELETE,
	"INSERT":      INSERT,
	"INTO":        INTO,
	"DO":          DO,
	"VALUES":      VALUES,
	"FOR":         FOR,
	"THEN":        THEN,
	"ELSE":        ELSE,
	"DISTINCTROW": DISTINCTROW,
	"FILTER":      FILTER,
	"WITHIN":      WITHIN,
	"COLLATE":     COLLATE,
	"INTERSECT":   INTERSECT,
	"EXCEPT":      EXCEPT,
	"OFFSET":      OFFSET,
	"FETCH":       FETCH,
	"FIRST":       FIRST,
	"ROWS":        ROWS,
	"USING":       USING,
	"OVERLAPS":    OVERLAPS,
	"NATURAL":     NATURAL,
	"CROSS":       CROSS,
	"ZONE":        ZONE,
	"NULLS":       NULLS,
	"LAST":        LAST,
	"AT":          AT,
	"LOCK":        LOCK,
	"WITH":        WITH,
	"PRIMARY":     PRIMARY,
	"KEY":         KEY,

	/*
	 * Special queries
	 */
	"SHOW":      SHOW,
	"DISCARD":   DISCARD,
	"BEGIN":     BEGIN,
	"SAVEPOINT": SAVEPOINT,
	"RELEASE":   RELEASE,
	"ROLLBACK":  ROLLBACK,
	"COMMIT":    COMMIT,

	/*
	 * Data types
	 */
	"BIG":        TYPE,
	"BIGSERIAL":  TYPE,
	"BOOLEAN":    TYPE,
	"CHAR":       TYPE,
	"BIT":        TYPE,
	"TEXT":       TYPE,
	"INT":        TYPE,
	"INTEGER":    TYPE,
	"NUMERIC":    TYPE,
	"DECIMAL":    TYPE,
	"DEC":        TYPE,
	"FLOAT":      TYPE,
	"CUSTOMTYPE": TYPE,
	"VARCHAR":    TYPE,
	"VARBIT":     TYPE,
	"TIMESTAMP":  TYPE,
	"TIME":       TYPE,
	"SECOND":     TYPE,
	"INTERVAL":   TYPE,

	/*
	 * Additional Postgres functions without parenthesis.
	 * These
	 */
	"LOCALTIME":         FUNCTIONKEYWORD,
	"LOCALTIMESTAMP":    FUNCTIONKEYWORD,
	"CURRENT_DATE":      FUNCTIONKEYWORD,
	"CURRENT_TIME":      FUNCTIONKEYWORD,
	"CURRENT_TIMESTAMP": FUNCTIONKEYWORD,
	"CURRENT_USER":      FUNCTIONKEYWORD,
	"CURRENT_CATALOG":   FUNCTIONKEYWORD,
	"SESSION_USER":      FUNCTIONKEYWORD,
	"USER":              FUNCTIONKEYWORD,
}

var functionMap = map[string]TokenType{

	/*
	 * PostgreSQL functions
	 */
	"BIT_LENGTH":                         FUNCTION,
	"CHAR_LENGTH":                        FUNCTION,
	"LOWER":                              FUNCTION,
	"OCTET_LENGTH":                       FUNCTION,
	"OVERLAY":                            FUNCTION,
	"POSITION":                           FUNCTION,
	"SUBSTRING":                          FUNCTION,
	"TRIM":                               FUNCTION,
	"UPPER":                              FUNCTION,
	"ASCII":                              FUNCTION,
	"BTRIM":                              FUNCTION,
	"CHR":                                FUNCTION,
	"CONCAT":                             FUNCTION,
	"CONCAT_WS":                          FUNCTION,
	"CONVERT":                            FUNCTION,
	"CONVERT_FROM":                       FUNCTION,
	"CONVERT_TO":                         FUNCTION,
	"DECODE":                             FUNCTION,
	"ENCODE":                             FUNCTION,
	"FORMAT":                             FUNCTION,
	"INITCAP":                            FUNCTION,
	"LEFT":                               FUNCTION,
	"LENGTH":                             FUNCTION,
	"LPAD":                               FUNCTION,
	"LTRIM":                              FUNCTION,
	"MD5":                                FUNCTION,
	"PG_CLIENT_ENCODING":                 FUNCTION,
	"QUOTE_IDENT":                        FUNCTION,
	"QUOTE_LITERAL":                      FUNCTION,
	"QUOTE_NULLABLE":                     FUNCTION,
	"REGEXP_MATCHES":                     FUNCTION,
	"REGEXP_REPLACE":                     FUNCTION,
	"REGEXP_SPLIT_TO_ARRAY":              FUNCTION,
	"REGEXP_SPLIT_TO_TABLE":              FUNCTION,
	"REPEAT":                             FUNCTION,
	"REPLACE":                            FUNCTION,
	"REVERSE":                            FUNCTION,
	"RIGHT":                              FUNCTION,
	"RPAD":                               FUNCTION,
	"RTRIM":                              FUNCTION,
	"SPLIT_PART":                         FUNCTION,
	"STRPOS":                             FUNCTION,
	"SUBSTR":                             FUNCTION,
	"TO_ASCII":                           FUNCTION,
	"TO_HEX":                             FUNCTION,
	"TRANSLATE":                          FUNCTION,
	"GET_BIT":                            FUNCTION,
	"GET_BYTE":                           FUNCTION,
	"SET_BIT":                            FUNCTION,
	"SET_BYTE":                           FUNCTION,
	"TO_CHAR":                            FUNCTION,
	"TO_DATE":                            FUNCTION,
	"TO_NUMBER":                          FUNCTION,
	"TO_TIMESTAMP":                       FUNCTION,
	"AGE":                                FUNCTION,
	"CLOCK_TIMESTAMP":                    FUNCTION,
	"DATE_PART":                          FUNCTION,
	"DATE_TRUNC":                         FUNCTION,
	"EXTRACT":                            FUNCTION,
	"ISFINITE":                           FUNCTION,
	"JUSTIFY_DAYS":                       FUNCTION,
	"JUSTIFY_HOURS":                      FUNCTION,
	"JUSTIFY_INTERVAL":                   FUNCTION,
	"NOW":                                FUNCTION,
	"STATEMENT_TIMESTAMP":                FUNCTION,
	"TIMEOFDAY":                          FUNCTION,
	"TRANSACTION_TIMESTAMP":              FUNCTION,
	"ENUM_FIRST":                         FUNCTION,
	"ENUM_LAST":                          FUNCTION,
	"ENUM_RANGE":                         FUNCTION,
	"AREA":                               FUNCTION,
	"CENTER":                             FUNCTION,
	"DIAMETER":                           FUNCTION,
	"HEIGHT":                             FUNCTION,
	"ISCLOSED":                           FUNCTION,
	"ISOPEN":                             FUNCTION,
	"NPOINTS":                            FUNCTION,
	"PCLOSE":                             FUNCTION,
	"POPEN":                              FUNCTION,
	"RADIUS":                             FUNCTION,
	"WIDTH":                              FUNCTION,
	"BOX":                                FUNCTION,
	"CIRCLE":                             FUNCTION,
	"LSEG":                               FUNCTION,
	"PATH":                               FUNCTION,
	"POINT":                              FUNCTION,
	"POLYGON":                            FUNCTION,
	"ABBREV":                             FUNCTION,
	"BROADCAST":                          FUNCTION,
	"FAMILY":                             FUNCTION,
	"HOST":                               FUNCTION,
	"HOSTMASK":                           FUNCTION,
	"MASKLEN":                            FUNCTION,
	"NETMASK":                            FUNCTION,
	"NETWORK":                            FUNCTION,
	"SET_MASKLEN":                        FUNCTION,
	"TEXT":                               FUNCTION,
	"TRUNC":                              FUNCTION,
	"GET_CURRENT_TS_CONFIG":              FUNCTION,
	"NUMNODE":                            FUNCTION,
	"PLAINTO_TSQUERY":                    FUNCTION,
	"QUERYTREE":                          FUNCTION,
	"SETWEIGHT":                          FUNCTION,
	"STRIP":                              FUNCTION,
	"TO_TSQUERY":                         FUNCTION,
	"TO_TSVECTOR":                        FUNCTION,
	"TS_HEADLINE":                        FUNCTION,
	"TS_RANK":                            FUNCTION,
	"TS_RANK_CD":                         FUNCTION,
	"TS_REWRITE":                         FUNCTION,
	"TSVECTOR_UPDATE_TRIGGER":            FUNCTION,
	"TSVECTOR_UPDATE_TRIGGER_COLUMN":     FUNCTION,
	"TS_DEBUG":                           FUNCTION,
	"TS_LEXIZE":                          FUNCTION,
	"TS_PARSE":                           FUNCTION,
	"TS_TOKEN_TYPE":                      FUNCTION,
	"TS_STAT":                            FUNCTION,
	"XMLCOMMENT":                         FUNCTION,
	"XMLCONCAT":                          FUNCTION,
	"XMLELEMENT":                         FUNCTION,
	"XMLFOREST":                          FUNCTION,
	"XMLPI":                              FUNCTION,
	"XMLROOT":                            FUNCTION,
	"XMLAGG":                             FUNCTION,
	"XMLEXISTS":                          FUNCTION,
	"XML_IS_WELL_FORMED":                 FUNCTION,
	"XML_IS_WELL_FORMED_DOCUMENT":        FUNCTION,
	"XML_IS_WELL_FORMED_CONTENT":         FUNCTION,
	"XPATH":                              FUNCTION,
	"XPATH_EXISTS":                       FUNCTION,
	"TABLE_TO_XML":                       FUNCTION,
	"QUERY_TO_XML":                       FUNCTION,
	"CURSOR_TO_XML":                      FUNCTION,
	"TABLE_TO_XMLSCHEMA":                 FUNCTION,
	"QUERY_TO_XMLSCHEMA":                 FUNCTION,
	"CURSOR_TO_XMLSCHEMA":                FUNCTION,
	"TABLE_TO_XML_AND_XMLSCHEMA":         FUNCTION,
	"QUERY_TO_XML_AND_XMLSCHEMA":         FUNCTION,
	"SCHEMA_TO_XML":                      FUNCTION,
	"SCHEMA_TO_XMLSCHEMA":                FUNCTION,
	"SCHEMA_TO_XML_AND_XMLSCHEMA":        FUNCTION,
	"DATABASE_TO_XML":                    FUNCTION,
	"DATABASE_TO_XMLSCHEMA":              FUNCTION,
	"DATABASE_TO_XML_AND_XMLSCHEMA":      FUNCTION,
	"CURRVAL":                            FUNCTION,
	"LASTVAL":                            FUNCTION,
	"NEXTVAL":                            FUNCTION,
	"SETVAL":                             FUNCTION,
	"ARRAY_APPEND":                       FUNCTION,
	"ARRAY_CAT":                          FUNCTION,
	"ARRAY_NDIMS":                        FUNCTION,
	"ARRAY_DIMS":                         FUNCTION,
	"ARRAY_FILL":                         FUNCTION,
	"ARRAY_LENGTH":                       FUNCTION,
	"ARRAY_LOWER":                        FUNCTION,
	"ARRAY_PREPEND":                      FUNCTION,
	"ARRAY_TO_STRING":                    FUNCTION,
	"ARRAY_UPPER":                        FUNCTION,
	"STRING_TO_ARRAY":                    FUNCTION,
	"UNNEST":                             FUNCTION,
	"ARRAY_AGG":                          FUNCTION,
	"AVG":                                FUNCTION,
	"BIT_AND":                            FUNCTION,
	"BIT_OR":                             FUNCTION,
	"BOOL_AND":                           FUNCTION,
	"BOOL_OR":                            FUNCTION,
	"COUNT":                              FUNCTION,
	"EVERY":                              FUNCTION,
	"MAX":                                FUNCTION,
	"MIN":                                FUNCTION,
	"STRING_AGG":                         FUNCTION,
	"SUM":                                FUNCTION,
	"CORR":                               FUNCTION,
	"COVAR_POP":                          FUNCTION,
	"COVAR_SAMP":                         FUNCTION,
	"REGR_AVGX":                          FUNCTION,
	"REGR_AVGY":                          FUNCTION,
	"REGR_COUNT":                         FUNCTION,
	"REGR_INTERCEPT":                     FUNCTION,
	"REGR_R2":                            FUNCTION,
	"REGR_SLOPE":                         FUNCTION,
	"REGR_SXX":                           FUNCTION,
	"REGR_SXY":                           FUNCTION,
	"REGR_SYY":                           FUNCTION,
	"STDDEV":                             FUNCTION,
	"STDDEV_POP":                         FUNCTION,
	"STDDEV_SAMP":                        FUNCTION,
	"VARIANCE":                           FUNCTION,
	"VAR_POP":                            FUNCTION,
	"VAR_SAMP":                           FUNCTION,
	"ROW_NUMBER":                         FUNCTION,
	"RANK":                               FUNCTION,
	"DENSE_RANK":                         FUNCTION,
	"PERCENT_RANK":                       FUNCTION,
	"CUME_DIST":                          FUNCTION,
	"NTILE":                              FUNCTION,
	"LAG":                                FUNCTION,
	"LEAD":                               FUNCTION,
	"FIRST_VALUE":                        FUNCTION,
	"LAST_VALUE":                         FUNCTION,
	"NTH_VALUE":                          FUNCTION,
	"GENERATE_SERIES":                    FUNCTION,
	"GENERATE_SUBSCRIPTS":                FUNCTION,
	"CURRENT_DATABASE":                   FUNCTION,
	"CURRENT_QUERY":                      FUNCTION,
	"CURRENT_SCHEMA[":                    FUNCTION,
	"CURRENT_SCHEMAS":                    FUNCTION,
	"INET_CLIENT_ADDR":                   FUNCTION,
	"INET_CLIENT_PORT":                   FUNCTION,
	"INET_SERVER_ADDR":                   FUNCTION,
	"INET_SERVER_PORT":                   FUNCTION,
	"PG_BACKEND_PID":                     FUNCTION,
	"PG_CONF_LOAD_TIME":                  FUNCTION,
	"PG_IS_OTHER_TEMP_SCHEMA":            FUNCTION,
	"PG_LISTENING_CHANNELS":              FUNCTION,
	"PG_MY_TEMP_SCHEMA":                  FUNCTION,
	"PG_POSTMASTER_START_TIME":           FUNCTION,
	"VERSION":                            FUNCTION,
	"HAS_ANY_COLUMN_PRIVILEGE":           FUNCTION,
	"HAS_COLUMN_PRIVILEGE":               FUNCTION,
	"HAS_DATABASE_PRIVILEGE":             FUNCTION,
	"HAS_FOREIGN_DATA_WRAPPER_PRIVILEGE": FUNCTION,
	"HAS_FUNCTION_PRIVILEGE":             FUNCTION,
	"HAS_LANGUAGE_PRIVILEGE":             FUNCTION,
	"HAS_SCHEMA_PRIVILEGE":               FUNCTION,
	"HAS_SEQUENCE_PRIVILEGE":             FUNCTION,
	"HAS_SERVER_PRIVILEGE":               FUNCTION,
	"HAS_TABLE_PRIVILEGE":                FUNCTION,
	"HAS_TABLESPACE_PRIVILEGE":           FUNCTION,
	"PG_HAS_ROLE":                        FUNCTION,
	"PG_COLLATION_IS_VISIBLE":            FUNCTION,
	"PG_CONVERSION_IS_VISIBLE":           FUNCTION,
	"PG_FUNCTION_IS_VISIBLE":             FUNCTION,
	"PG_OPCLASS_IS_VISIBLE":              FUNCTION,
	"PG_OPERATOR_IS_VISIBLE":             FUNCTION,
	"PG_TABLE_IS_VISIBLE":                FUNCTION,
	"PG_TS_CONFIG_IS_VISIBLE":            FUNCTION,
	"PG_TS_DICT_IS_VISIBLE":              FUNCTION,
	"PG_TS_PARSER_IS_VISIBLE":            FUNCTION,
	"PG_TS_TEMPLATE_IS_VISIBLE":          FUNCTION,
	"PG_TYPE_IS_VISIBLE":                 FUNCTION,
	"FORMAT_TYPE":                        FUNCTION,
	"PG_DESCRIBE_OBJECT":                 FUNCTION,
	"PG_GET_CONSTRAINTDEF":               FUNCTION,
	"PG_GET_EXPR":                        FUNCTION,
	"PG_GET_FUNCTIONDEF":                 FUNCTION,
	"PG_GET_FUNCTION_ARGUMENTS":          FUNCTION,
	"PG_GET_FUNCTION_IDENTITY_ARGUMENTS": FUNCTION,
	"PG_GET_FUNCTION_RESULT":             FUNCTION,
	"PG_GET_INDEXDEF":                    FUNCTION,
	"PG_GET_KEYWORDS":                    FUNCTION,
	"PG_GET_RULEDEF":                     FUNCTION,
	"PG_GET_SERIAL_SEQUENCE":             FUNCTION,
	"PG_GET_TRIGGERDEF":                  FUNCTION,
	"PG_GET_USERBYID":                    FUNCTION,
	"PG_GET_VIEWDEF":                     FUNCTION,
	"PG_OPTIONS_TO_TABLE":                FUNCTION,
	"PG_TABLESPACE_DATABASES":            FUNCTION,
	"PG_TYPEOF":                          FUNCTION,
	"COL_DESCRIPTION":                    FUNCTION,
	"OBJ_DESCRIPTION":                    FUNCTION,
	"SHOBJ_DESCRIPTION":                  FUNCTION,
	"TXID_CURRENT":                       FUNCTION,
	"TXID_CURRENT_SNAPSHOT":              FUNCTION,
	"TXID_SNAPSHOT_XIP":                  FUNCTION,
	"TXID_SNAPSHOT_XMAX":                 FUNCTION,
	"TXID_SNAPSHOT_XMIN":                 FUNCTION,
	"TXID_VISIBLE_IN_SNAPSHOT":           FUNCTION,
	"CURRENT_SETTING":                    FUNCTION,
	"SET_CONFIG":                         FUNCTION,
	"PG_CANCEL_BACKEND":                  FUNCTION,
	"PG_RELOAD_CONF":                     FUNCTION,
	"PG_ROTATE_LOGFILE":                  FUNCTION,
	"PG_TERMINATE_BACKEND":               FUNCTION,
	"PG_CREATE_RESTORE_POINT":            FUNCTION,
	"PG_CURRENT_XLOG_INSERT_LOCATION":    FUNCTION,
	"PG_CURRENT_XLOG_LOCATION":           FUNCTION,
	"PG_START_BACKUP":                    FUNCTION,
	"PG_STOP_BACKUP":                     FUNCTION,
	"PG_SWITCH_XLOG":                     FUNCTION,
	"PG_XLOGFILE_NAME":                   FUNCTION,
	"PG_XLOGFILE_NAME_OFFSET":            FUNCTION,
	"PG_IS_IN_RECOVERY":                  FUNCTION,
	"PG_LAST_XLOG_RECEIVE_LOCATION":      FUNCTION,
	"PG_LAST_XLOG_REPLAY_LOCATION":       FUNCTION,
	"PG_LAST_XACT_REPLAY_TIMESTAMP":      FUNCTION,
	"PG_IS_XLOG_REPLAY_PAUSED":           FUNCTION,
	"PG_XLOG_REPLAY_PAUSE":               FUNCTION,
	"PG_XLOG_REPLAY_RESUME":              FUNCTION,
	"PG_COLUMN_SIZE":                     FUNCTION,
	"PG_DATABASE_SIZE":                   FUNCTION,
	"PG_INDEXES_SIZE":                    FUNCTION,
	"PG_RELATION_SIZE":                   FUNCTION,
	"PG_SIZE_PRETTY":                     FUNCTION,
	"PG_TABLE_SIZE":                      FUNCTION,
	"PG_TABLESPACE_SIZE":                 FUNCTION,
	"PG_TOTAL_RELATION_SIZE":             FUNCTION,
	"PG_RELATION_FILENODE":               FUNCTION,
	"PG_RELATION_FILEPATH":               FUNCTION,
	"PG_LS_DIR":                          FUNCTION,
	"PG_READ_FILE":                       FUNCTION,
	"PG_READ_BINARY_FILE":                FUNCTION,
	"PG_STAT_FILE":                       FUNCTION,
	"PG_ADVISORY_LOCK":                   FUNCTION,
	"PG_ADVISORY_LOCK_SHARED":            FUNCTION,
	"PG_ADVISORY_UNLOCK":                 FUNCTION,
	"PG_ADVISORY_UNLOCK_ALL":             FUNCTION,
	"PG_ADVISORY_UNLOCK_SHARED":          FUNCTION,
	"PG_ADVISORY_XACT_LOCK":              FUNCTION,
	"PG_ADVISORY_XACT_LOCK_SHARED":       FUNCTION,
	"PG_TRY_ADVISORY_LOCK":               FUNCTION,
	"PG_TRY_ADVISORY_LOCK_SHARED":        FUNCTION,
	"PG_TRY_ADVISORY_XACT_LOCK":          FUNCTION,
	"PG_TRY_ADVISORY_XACT_LOCK_SHARED":   FUNCTION,

	/*
	 * Additional SQL Functions
	 */
	"CHAR":              FUNCTION,
	"CHARINDEX":         FUNCTION,
	"DATALENGTH":        FUNCTION,
	"DIFFERENCE":        FUNCTION,
	"LEN":               FUNCTION,
	"NCHAR":             FUNCTION,
	"PATINDEX":          FUNCTION,
	"QUOTENAME":         FUNCTION,
	"REPLICATE":         FUNCTION,
	"SOUNDEX":           FUNCTION,
	"SPACE":             FUNCTION,
	"STR":               FUNCTION,
	"STUFF":             FUNCTION,
	"UNICODE":           FUNCTION,
	"ABS":               FUNCTION,
	"ACOS":              FUNCTION,
	"ASIN":              FUNCTION,
	"ATAN":              FUNCTION,
	"ATN2":              FUNCTION,
	"CEILING":           FUNCTION,
	"COS":               FUNCTION,
	"COT":               FUNCTION,
	"DEGREES":           FUNCTION,
	"EXP":               FUNCTION,
	"FLOOR":             FUNCTION,
	"LOG":               FUNCTION,
	"LOG10":             FUNCTION,
	"PI":                FUNCTION,
	"POWER":             FUNCTION,
	"RADIANS":           FUNCTION,
	"RAND":              FUNCTION,
	"ROUND":             FUNCTION,
	"SIGN":              FUNCTION,
	"SIN":               FUNCTION,
	"SQRT":              FUNCTION,
	"SQUARE":            FUNCTION,
	"TAN":               FUNCTION,
	"CURRENT_TIMESTAMP": FUNCTION,
	"DATEADD":           FUNCTION,
	"DATEDIFF":          FUNCTION,
	"DATEFROMPARTS":     FUNCTION,
	"DATENAME":          FUNCTION,
	"DATEPART":          FUNCTION,
	"DAY":               FUNCTION,
	"GETDATE":           FUNCTION,
	"GETUTCDATE":        FUNCTION,
	"ISDATE":            FUNCTION,
	"MONTH":             FUNCTION,
	"SYSDATETIME":       FUNCTION,
	"YEAR":              FUNCTION,
	"CAST":              FUNCTION,
	"COALESCE":          FUNCTION,
	"CURRENT_USER":      FUNCTION,
	"IIF":               FUNCTION,
	"ISNULL":            FUNCTION,
	"ISNUMERIC":         FUNCTION,
	"NULLIF":            FUNCTION,
	"SESSION_USER":      FUNCTION,
	"SESSIONPROPERTY":   FUNCTION,
	"SYSTEM_USER":       FUNCTION,
	"USER_NAME":         FUNCTION,

	/*
	 * Additional SQLite functions
	 */
	"CHANGES":                   FUNCTION,
	"GLOB":                      FUNCTION,
	"HEX":                       FUNCTION,
	"IFNULL":                    FUNCTION,
	"INSTR":                     FUNCTION,
	"LAST_INSERT_ROWID":         FUNCTION,
	"LIKELIHOOD":                FUNCTION,
	"LIKELY":                    FUNCTION,
	"LOAD_EXTENSION":            FUNCTION,
	"PRINTF":                    FUNCTION,
	"QUOTE":                     FUNCTION,
	"RANDOMBLOB":                FUNCTION,
	"SQLITE_COMPILEOPTION_GET":  FUNCTION,
	"SQLITE_COMPILEOPTION_USED": FUNCTION,
	"SQLITE_OFFSET":             FUNCTION,
	"SQLITE_SOURCE_ID":          FUNCTION,
	"SQLITE_VERSION":            FUNCTION,
	"TOTAL_CHANGES":             FUNCTION,
	"TYPEOF":                    FUNCTION,
	"UNHEX":                     FUNCTION,
	"UNLIKELY":                  FUNCTION,
	"ZEROBLOB":                  FUNCTION,
	"GROUP_CONCAT":              FUNCTION,
	"DATE":                      FUNCTION,
	"TIME":                      FUNCTION,
	"DATETIME":                  FUNCTION,
	"JULIANDAY":                 FUNCTION,
	"STRFTIME":                  FUNCTION,
	"RANDOM":                    FUNCTION,

	/*
	 * Additional Oracle functions
	 */
	"ATAN2":                        FUNCTION,
	"BITAND":                       FUNCTION,
	"COSH":                         FUNCTION,
	"LN":                           FUNCTION,
	"NANVL":                        FUNCTION,
	"REMAINDER":                    FUNCTION,
	"SINH":                         FUNCTION,
	"TANH":                         FUNCTION,
	"WIDTH_BUCKET":                 FUNCTION,
	"NLS_INITCAP":                  FUNCTION,
	"NLS_LOWER":                    FUNCTION,
	"NLSSORT":                      FUNCTION,
	"NLS_UPPER":                    FUNCTION,
	"REGEXP_SUBSTR":                FUNCTION,
	"TREAT":                        FUNCTION,
	"REGEXP_INSTR":                 FUNCTION,
	"ADD_MONTHS":                   FUNCTION,
	"DBTIMEZONE":                   FUNCTION,
	"FROM_TZ":                      FUNCTION,
	"LAST_DAY":                     FUNCTION,
	"MONTHS_BETWEEN":               FUNCTION,
	"NEW_TIME":                     FUNCTION,
	"NEXT_DAY":                     FUNCTION,
	"NUMTODSINTERVAL":              FUNCTION,
	"NUMTOYMINTERVAL":              FUNCTION,
	"SESSIONTIMEZONE":              FUNCTION,
	"SYS_EXTRACT_UTC":              FUNCTION,
	"SYSDATE":                      FUNCTION,
	"SYSTIMESTAMP":                 FUNCTION,
	"TO_TIMESTAMP_TZ":              FUNCTION,
	"TO_DSINTERVAL":                FUNCTION,
	"TO_YMINTERVAL":                FUNCTION,
	"TZ_OFFSET":                    FUNCTION,
	"ASCIISTR":                     FUNCTION,
	"BIN_TO_NUM":                   FUNCTION,
	"CHARTOROWID":                  FUNCTION,
	"COMPOSE":                      FUNCTION,
	"DECOMPOSE":                    FUNCTION,
	"HEXTORAW":                     FUNCTION,
	"RAWTOHEX":                     FUNCTION,
	"RAWTONHEX":                    FUNCTION,
	"ROWIDTOCHAR":                  FUNCTION,
	"ROWIDTONCHAR":                 FUNCTION,
	"SCN_TO_TIMESTAMP":             FUNCTION,
	"TIMESTAMP_TO_SCN":             FUNCTION,
	"TO_BINARY_DOUBLE":             FUNCTION,
	"TO_BINARY_FLOAT":              FUNCTION,
	"TO_CLOB":                      FUNCTION,
	"TO_LOB":                       FUNCTION,
	"TO_MULTI_BYTE":                FUNCTION,
	"TO_NCHAR":                     FUNCTION,
	"TO_NCLOB":                     FUNCTION,
	"TO_SINGLE_BYTE":               FUNCTION,
	"UNISTR":                       FUNCTION,
	"CARDINALITY":                  FUNCTION,
	"COLLECT":                      FUNCTION,
	"POWERMULTISET":                FUNCTION,
	"POWERMULTISET_BY_CARDINALITY": FUNCTION,
	"BFILENAME":                    FUNCTION,
	"CV":                           FUNCTION,
	"DEPTH":                        FUNCTION,
	"DUMP":                         FUNCTION,
	"EMPTY_BLOB":                   FUNCTION,
	"EMPTY_CLOB":                   FUNCTION,
	"EXISTSNODE":                   FUNCTION,
	"EXTRACTVALUE":                 FUNCTION,
	"LNNVL":                        FUNCTION,
	"NLS_CHARSET_DECL_LEN":         FUNCTION,
	"NLS_CHARSET_ID":               FUNCTION,
	"NLS_CHARSET_NAME":             FUNCTION,
	"NVL":                          FUNCTION,
	"NVL2":                         FUNCTION,
	"ORA_HASH":                     FUNCTION,
	"PRESENTNNV":                   FUNCTION,
	"PRESENTV":                     FUNCTION,
	"PREVIOUS":                     FUNCTION,
	"SYS_CONNECT_BY_PATH":          FUNCTION,
	"SYS_CONTEXT":                  FUNCTION,
	"SYS_DBURIGEN":                 FUNCTION,
	"SYS_GUID":                     FUNCTION,
	"SYS_TYPEID":                   FUNCTION,
	"SYS_XMLAGG":                   FUNCTION,
	"SYS_XMLGEN":                   FUNCTION,
	"UID":                          FUNCTION,
	"UPDATEXML":                    FUNCTION,
	"USERENV":                      FUNCTION,
	"VSIZE":                        FUNCTION,
	"XMLCOLATTVAL":                 FUNCTION,
	"XMLSEQUENCE":                  FUNCTION,
	"XMLTRANSFORM":                 FUNCTION,
	"CORR_S":                       FUNCTION,
	"CORR_K":                       FUNCTION,
	"GROUP_ID":                     FUNCTION,
	"GROUPING":                     FUNCTION,
	"GROUPING_ID":                  FUNCTION,
	"MEDIAN":                       FUNCTION,
	"PERCENTILE_CONT":              FUNCTION,
	"STATS_BINOMIAL_TEST":          FUNCTION,
	"STATS_CROSSTAB":               FUNCTION,
	"STATS_F_TEST":                 FUNCTION,
	"STATS_KS_TEST":                FUNCTION,
	"STATS_MODE":                   FUNCTION,
	"STATS_MW_TEST":                FUNCTION,
	"STATS_ONE_WAY_ANOVA":          FUNCTION,
	"STATS_T_TEST_ONE":             FUNCTION,
	"STATS_T_TEST_PAIRD":           FUNCTION,
	"STATS_T_TEST_INDEP":           FUNCTION,
	"STATS_T_TEST_INDEPU":          FUNCTION,
	"STATS_WSR_TEST":               FUNCTION,
	"PERCENTILE_DISC":              FUNCTION,
}

var comparatorMap = map[string]TokenType{
	"~~":   COMPARATOR, // LIKE
	"~~*":  COMPARATOR, // ILIKE
	"!~~":  COMPARATOR, // NOT LIKE
	"!~~*": COMPARATOR, // NOT ILIKE
	"=":    COMPARATOR,
	"!=":   COMPARATOR, // NOT EQUAL
	"<>":   COMPARATOR, // NOT EQUAL
	">":    COMPARATOR,
	"<":    COMPARATOR,
	">=":   COMPARATOR,
	"<=":   COMPARATOR,
}

// peekComparator peeks into the subsequent characters trying to identify valid comparator substrings, but
// tries not to match on broken substrings that are not really valid comparators.
func peekComparator(r *bufio.Reader) (string, error) {

	// Peek step by step into subsequent characters to search for a valid comparator
	steps := 1
	sequence := ""
	for {
		b, errPeek := r.Peek(steps)
		if errPeek != nil {
			break
		}

		// Convert to string and get last character to analyze
		s := string(b)
		ch := s[len(s)-1]

		// Check if character is plausible comparator
		if strings.Contains("~*!=<>", string(ch)) {
			sequence += string(ch)
		} else {
			break
		}

		// Increment bytes to read
		steps++
	}

	// Check if read sequence is valid comparator
	if sequence != "" {

		// Return comparator sequence
		if _, ok := comparatorMap[sequence]; ok {
			return sequence, nil
		}

		// Return error if invalid comparator sequence was detected
		return "", fmt.Errorf("invalid comparator sequence: %s", sequence)
	}

	// Return empty string if sequence was not a comparator
	return "", nil
}

var punctuationMap = map[string]TokenType{
	"(": STARTPARENTHESIS,
	")": ENDPARENTHESIS,
	"[": STARTBRACKET,
	"]": ENDBRACKET,
	"{": STARTBRACE,
	"}": ENDBRACKET,
	",": COMMA,
	":": COLON,
}
