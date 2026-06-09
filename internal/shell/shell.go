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
	open := strings.IndexByte(rest, '(')
	if open < 0 {
		return Query{}, false
	}
	head := rest[:open]
	collection, operation := head, "find"
	if dot := strings.IndexByte(head, '.'); dot >= 0 {
		collection = head[:dot]
		operation = head[dot+1:]
	}

	return Query{
		Collection: strings.TrimSpace(collection),
		Operation:  strings.TrimSpace(operation),
		Args:       splitTopLevelArgs(balancedArgs(rest[open+1:])),
	}, true
}

func ParseSQL(input string) (collection, where string, ok bool) {
	upper := strings.ToUpper(strings.TrimSpace(input))
	from := strings.Index(upper, " FROM ")
	if from < 0 {
		return "", "", false
	}
	remainder := strings.TrimSpace(input[from+6:])
	fields := strings.Fields(remainder)
	if len(fields) == 0 || fields[0] == "" {
		return "", "", false
	}
	collection = strings.Trim(fields[0], "`\"[]")

	if at := strings.Index(strings.ToUpper(remainder), " WHERE "); at >= 0 {
		where = strings.TrimSpace(remainder[at+7:])
	}
	return collection, where, true
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
