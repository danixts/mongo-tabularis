package shell

import "strings"

type Query struct {
	Collection string
	Operation  string
	Args       []string
}

func Parse(input string) (Query, bool) {
	statement := strings.TrimRight(strings.TrimSpace(input), ";")

	rest, ok := strings.CutPrefix(statement, "db.")
	if !ok {
		return Query{}, false
	}
	dot := strings.IndexByte(rest, '.')
	if dot < 0 {
		return Query{}, false
	}
	collection := strings.TrimSpace(rest[:dot])
	afterCollection := rest[dot+1:]

	open := strings.IndexByte(afterCollection, '(')
	if open < 0 {
		return Query{}, false
	}
	operation := strings.TrimSpace(afterCollection[:open])

	return Query{
		Collection: collection,
		Operation:  operation,
		Args:       splitTopLevelArgs(balancedArgs(afterCollection[open+1:])),
	}, true
}

func ParseSQLFrom(input string) (string, bool) {
	upper := strings.ToUpper(strings.TrimSpace(input))
	from := strings.Index(upper, " FROM ")
	if from < 0 {
		return "", false
	}
	fields := strings.Fields(strings.TrimSpace(input[from+6:]))
	if len(fields) == 0 || fields[0] == "" {
		return "", false
	}
	return strings.Trim(fields[0], "`\"[]"), true
}

func (q Query) Arg(index int) string {
	if index < len(q.Args) {
		return q.Args[index]
	}
	return ""
}

func balancedArgs(input string) string {
	depth := 0
	inString := false
	escaped := false
	for i, char := range input {
		switch {
		case escaped:
			escaped = false
		case char == '\\' && inString:
			escaped = true
		case char == '"':
			inString = !inString
		case inString:
		case char == '{' || char == '[' || char == '(':
			depth++
		case char == '}' || char == ']':
			if depth > 0 {
				depth--
			}
		case char == ')':
			if depth == 0 {
				return input[:i]
			}
			depth--
		}
	}
	return strings.TrimSpace(input)
}

func splitTopLevelArgs(input string) []string {
	depth := 0
	start := 0
	inString := false
	escaped := false
	var args []string

	for i, char := range input {
		switch {
		case escaped:
			escaped = false
		case char == '\\' && inString:
			escaped = true
		case char == '"':
			inString = !inString
		case inString:
		case char == '{' || char == '[' || char == '(':
			depth++
		case char == '}' || char == ']' || char == ')':
			if depth > 0 {
				depth--
			}
		case char == ',' && depth == 0:
			args = append(args, strings.TrimSpace(input[start:i]))
			start = i + 1
		}
	}
	if last := strings.TrimSpace(input[start:]); last != "" {
		args = append(args, last)
	}
	return args
}
