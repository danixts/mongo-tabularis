package codec

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

var (
	bareKeyPattern   = regexp.MustCompile(`([{,]\s*)([A-Za-z_$][\w$.]*)(\s*:)`)
	andSeparator     = regexp.MustCompile(`(?i)\s+AND\s+`)
	conditionPattern = regexp.MustCompile(`^\s*(.+?)\s*(>=|<=|!=|<>|=|>|<)\s*(.+?)\s*$`)
	likePattern      = regexp.MustCompile(`(?i)^\s*(.+?)\s+LIKE\s+(.+?)\s*$`)
)

var comparisonOperators = map[string]string{
	">":  "$gt",
	"<":  "$lt",
	">=": "$gte",
	"<=": "$lte",
	"!=": "$ne",
	"<>": "$ne",
}

func ParseWhere(where string) (bson.D, error) {
	where = strings.TrimSpace(where)
	if where == "" {
		return bson.D{}, nil
	}
	if strings.HasPrefix(where, "{") {
		return parseLooseFilter(where)
	}
	return sqlConditionsToFilter(where)
}

func ParseOrderBy(text string) bson.D {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	sort := bson.D{}
	for _, part := range strings.Split(text, ",") {
		tokens := strings.Fields(strings.TrimSpace(part))
		if len(tokens) == 0 {
			continue
		}
		column := cleanIdentifier(tokens[0])
		if column == "" {
			continue
		}
		direction := int32(1)
		if len(tokens) > 1 && strings.EqualFold(tokens[1], "DESC") {
			direction = -1
		}
		sort = append(sort, bson.E{Key: column, Value: direction})
	}
	return sort
}

func parseLooseFilter(text string) (bson.D, error) {
	if document, err := ParseFilter(text); err == nil {
		return document, nil
	}
	normalized := bareKeyPattern.ReplaceAllString(text, `$1"$2"$3`)
	return ParseFilter(normalized)
}

func sqlConditionsToFilter(where string) (bson.D, error) {
	filter := bson.D{}
	for _, clause := range andSeparator.Split(where, -1) {
		condition, err := parseCondition(clause)
		if err != nil {
			return nil, err
		}
		filter = append(filter, condition...)
	}
	return filter, nil
}

func parseCondition(clause string) (bson.D, error) {
	if match := likePattern.FindStringSubmatch(clause); match != nil {
		column := cleanIdentifier(match[1])
		pattern := likeToRegex(stripQuotes(strings.TrimSpace(match[2])))
		return bson.D{{Key: column, Value: bson.D{{Key: "$regex", Value: pattern}}}}, nil
	}

	match := conditionPattern.FindStringSubmatch(clause)
	if match == nil {
		return nil, fmt.Errorf("unsupported filter condition: %q", strings.TrimSpace(clause))
	}
	column := cleanIdentifier(match[1])
	value := parseLiteral(strings.TrimSpace(match[3]))

	if operator, ok := comparisonOperators[match[2]]; ok {
		return bson.D{{Key: column, Value: bson.D{{Key: operator, Value: value}}}}, nil
	}
	return bson.D{{Key: column, Value: value}}, nil
}

func parseLiteral(token string) any {
	if unquoted := stripQuotes(token); unquoted != token {
		return unquoted
	}
	switch strings.ToLower(token) {
	case "true":
		return true
	case "false":
		return false
	case "null":
		return nil
	}
	if integer, err := strconv.ParseInt(token, 10, 64); err == nil {
		return integer
	}
	if float, err := strconv.ParseFloat(token, 64); err == nil {
		return float
	}
	return token
}

func cleanIdentifier(raw string) string {
	return strings.Trim(strings.TrimSpace(raw), "`\"[]")
}

func stripQuotes(token string) string {
	if len(token) >= 2 {
		first, last := token[0], token[len(token)-1]
		if (first == '\'' && last == '\'') || (first == '"' && last == '"') {
			return token[1 : len(token)-1]
		}
	}
	return token
}

func likeToRegex(pattern string) string {
	escaped := regexp.QuoteMeta(pattern)
	escaped = strings.ReplaceAll(escaped, "%", ".*")
	escaped = strings.ReplaceAll(escaped, "_", ".")
	return "^" + escaped + "$"
}
