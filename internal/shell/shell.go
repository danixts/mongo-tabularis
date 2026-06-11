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

type SQLSelect struct {
	Collection string
	Where      string
	OrderBy    string
}

func ParseSQL(input string) (SQLSelect, bool) {
	statement := strings.TrimRight(strings.TrimSpace(input), ";")
	upper := strings.ToUpper(statement)

	from := strings.Index(upper, " FROM ")
	if from < 0 {
		return SQLSelect{}, false
	}
	rest := statement[from+6:]
	restUpper := upper[from+6:]

	fields := strings.Fields(strings.TrimSpace(rest))
	if len(fields) == 0 || fields[0] == "" {
		return SQLSelect{}, false
	}
	result := SQLSelect{Collection: strings.Trim(fields[0], "`\"[]")}

	wherePos := strings.Index(restUpper, " WHERE ")
	orderPos := strings.Index(restUpper, " ORDER BY ")
	limitPos := strings.Index(restUpper, " LIMIT ")

	if orderPos >= 0 {
		end := len(rest)
		if limitPos > orderPos {
			end = limitPos
		}
		result.OrderBy = strings.TrimSpace(rest[orderPos+10 : end])
	}
	if wherePos >= 0 {
		end := len(rest)
		if orderPos > wherePos {
			end = orderPos
		} else if limitPos > wherePos {
			end = limitPos
		}
		result.Where = strings.TrimSpace(rest[wherePos+7 : end])
	}
	return result, true
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
