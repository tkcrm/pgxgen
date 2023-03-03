package sqlformatter

import (
	"fmt"
	"regexp"
	"strings"
)

var sep = "~::~"

type T struct {
	str         string
	shiftArr    []string
	tab         string
	arr         []string
	parensLevel int
	deep        int
}

type Replacement struct {
	pattern     *regexp.Regexp
	replacement string
}

func format(data []byte) string {
	sql := string(data)
	numSpaces := 2
	tab := strings.Repeat(" ", numSpaces)
	splitByQuotes := strings.Split(transformString(sql, []func(s string) string{
		func(s string) string { return regexp.MustCompile(`\\s+`).ReplaceAllString(s, " ") },
		func(s string) string { return strings.Replace(s, "'", strings.Join([]string{sep, "'"}, ""), -1) },
	}), sep)

	fmt.Println(splitByQuotes)

	input := T{
		str:         "",
		shiftArr:    createShiftArr(tab),
		tab:         tab,
		arr:         genArray(splitByQuotes, tab),
		parensLevel: 0,
		deep:        0,
	}
	output := input
	for i := 0; i < len(input.arr); i++ {
		output = genOutput(output, i)
	}

	return transformString(output.str, []func(s string) string{
		func(s string) string { return regexp.MustCompile("\\s+\\n").ReplaceAllString(s, "\n") },
		func(s string) string { return regexp.MustCompile("\\n+").ReplaceAllString(s, "\n") },
		func(s string) string { return strings.TrimSpace(s) },
	})
}

func genOutput(acc T, i int) T {
	originalEl := acc.arr[i]
	parensLevel := subqueryLevel(originalEl, acc.parensLevel)
	arr := make([]string, len(acc.arr))
	copy(arr, acc.arr)
	if regexp.MustCompile("SELECT|SET").MatchString(originalEl) {
		arr[i] = regexp.MustCompile(`,\\s+`).ReplaceAllString(originalEl, strings.Join([]string{",\n", acc.tab, acc.tab}, ""))
	}
	str, deep := updateStr(arr[i], parensLevel, acc, i)

	return T{
		str:         str,
		shiftArr:    acc.shiftArr,
		tab:         acc.tab,
		arr:         arr,
		parensLevel: parensLevel,
		deep:        deep,
	}
}

func updateStr(el string, parensLevel int, acc T, i int) (string, int) {
	if regexp.MustCompile("\\(\\s*SELECT").MatchString(el) {
		return strings.Join([]string{acc.str, acc.shiftArr[acc.deep+1], el}, ""), acc.deep + 1
	} else {
		var str string
		var deep int
		if strings.Contains(el, "'") {
			str = strings.Join([]string{acc.str, el}, "")
		} else {
			str = strings.Join([]string{acc.str, acc.shiftArr[acc.deep], el}, "")
		}

		if parensLevel < 1 && acc.deep != 0 {
			deep = acc.deep - 1
		} else {
			deep = acc.deep
		}

		return str, deep
	}
}

func transformString(str string, fns []func(s string) string) string {
	for _, fn := range fns {
		str = fn(str)
	}
	return str
}

func createShiftArr(tab string) []string {
	var a []string
	for i := 0; i < 100; i++ {
		a = append(a, strings.Join([]string{"\n", strings.Repeat(tab, i)}, ""))
	}
	return a
}

func genArray(splitByQuotes []string, tab string) []string {
	var a []string
	for i, _ := range splitByQuotes {
		a = append(a, splitIfEven(i, splitByQuotes[i], tab)...)
	}
	return a
}

func subqueryLevel(str string, level int) int {
	return level - (len(strings.Replace(str, "(", "", -1)) - len(strings.Replace(str, ")", "", -1)))
}

func allReplacements(tab string) []Replacement {
	return []Replacement{
		{regexp.MustCompile("(?i) AND "), sep + tab + "AND "},
		{regexp.MustCompile("(?i) BETWEEN "), sep + tab + "BETWEEN "},
		{regexp.MustCompile("(?i) CASE "), sep + tab + "CASE "},
		{regexp.MustCompile("(?i) ELSE "), sep + tab + "ELSE "},
		{regexp.MustCompile("(?i) END "), sep + tab + "END "},
		{regexp.MustCompile("(?i) FROM "), sep + "FROM "},
		{regexp.MustCompile("(?i) GROUP\\s+BY "), sep + "GROUP BY "},
		{regexp.MustCompile("(?i) HAVING "), sep + "HAVING "},
		{regexp.MustCompile("(?i) IN "), " IN "},
		{regexp.MustCompile("(?i) JOIN "), sep + "JOIN "},
		{regexp.MustCompile("(?i) CROSS(~::~)+JOIN "), sep + "CROSS JOIN "},
		{regexp.MustCompile("(?i) INNER(~::~)+JOIN "), sep + "INNER JOIN "},
		{regexp.MustCompile("(?i) LEFT(~::~)+JOIN "), sep + "LEFT JOIN "},
		{regexp.MustCompile("(?i) RIGHT(~::~)+JOIN "), sep + "RIGHT JOIN "},
		{regexp.MustCompile("(?i) ON "), sep + tab + "ON "},
		{regexp.MustCompile("(?i) OR "), sep + tab + "OR "},
		{regexp.MustCompile("(?i) ORDER\\s+BY "), sep + "ORDER BY "},
		{regexp.MustCompile("(?i) OVER "), sep + tab + "OVER "},
		{regexp.MustCompile("(?i)\\(\\s*SELECT "), sep + "(SELECT "},
		{regexp.MustCompile("(?i)\\)\\s*SELECT "), ")" + sep + "SELECT "},
		{regexp.MustCompile("(?i) THEN "), " THEN" + sep + tab},
		{regexp.MustCompile("(?i) UNION "), sep + "UNION" + sep},
		{regexp.MustCompile("(?i) USING "), sep + "USING "},
		{regexp.MustCompile("(?i) WHEN "), sep + tab + "WHEN "},
		{regexp.MustCompile("(?i) WHERE "), sep + "WHERE "},
		{regexp.MustCompile("(?i) WITH "), sep + "WITH "},
		{regexp.MustCompile("(?i) SET "), sep + "SET "},
		{regexp.MustCompile("(?i) ALL "), " ALL "},
		{regexp.MustCompile("(?i) AS "), " AS "},
		{regexp.MustCompile("(?i) ASC "), " ASC "},
		{regexp.MustCompile("(?i) DESC "), " DESC "},
		{regexp.MustCompile("(?i) DISTINCT "), " DISTINCT "},
		{regexp.MustCompile("(?i) EXISTS "), " EXISTS "},
		{regexp.MustCompile("(?i) NOT "), " NOT "},
		{regexp.MustCompile("(?i) NULL "), " NULL "},
		{regexp.MustCompile("(?i) LIKE "), " LIKE "},
		{regexp.MustCompile("(?i)\\s*SELECT "), "SELECT "},
		{regexp.MustCompile("(?i)\\s*UPDATE "), "UPDATE "},
		{regexp.MustCompile("(?i)\\s*DELETE "), "DELETE "},
		{regexp.MustCompile(strings.Join([]string{"(?i)(", sep, ")+"}, "")), sep},
	}
}

func splitSql(str string, tab string) []string {
	acc := str
	for _, r := range allReplacements(tab) {
		acc = r.pattern.ReplaceAllString(acc, r.replacement)
	}
	return strings.Split(acc, sep)
}

func splitIfEven(i int, str string, tab string) []string {
	if i%2 == 0 {
		return splitSql(str, tab)
	} else {
		return []string{str}
	}
}
