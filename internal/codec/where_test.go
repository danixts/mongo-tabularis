package codec

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestParseOrderBy(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"createdAt DESC", `{"createdAt":-1}`},
		{"name", `{"name":1}`},
		{"a DESC, b ASC", `{"a":-1,"b":1}`},
		{"", `null`},
	}
	for _, tc := range cases {
		got := ParseOrderBy(tc.input)
		if got == nil {
			if tc.want != "null" {
				t.Errorf("ParseOrderBy(%q) = nil, want %s", tc.input, tc.want)
			}
			continue
		}
		encoded, _ := bson.MarshalExtJSON(got, false, false)
		if string(encoded) != tc.want {
			t.Errorf("ParseOrderBy(%q) = %s, want %s", tc.input, encoded, tc.want)
		}
	}
}

func TestParseWhere(t *testing.T) {
	cases := []struct {
		name  string
		where string
		want  string
	}{
		{"sql equality number", "total = 96", `{"total":96}`},
		{"sql greater than", "durationMs > 0", `{"durationMs":{"$gt":0}}`},
		{"sql quoted string", "status = 'shipped'", `{"status":"shipped"}`},
		{"sql and", "a = 1 AND b > 2", `{"a":1,"b":{"$gt":2}}`},
		{"compass unquoted", "{ total : 96 }", `{"total":96}`},
		{"compass nested path", "{ body.total : 96 }", `{"body.total":96}`},
		{"strict json", `{ "age": { "$gt": 18 } }`, `{"age":{"$gt":18}}`},
		{"empty", "", `{}`},
	}
	for _, tc := range cases {
		got, err := ParseWhere(tc.where)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", tc.name, err)
			continue
		}
		encoded, err := bson.MarshalExtJSON(got, false, false)
		if err != nil {
			t.Errorf("%s: marshal error: %v", tc.name, err)
			continue
		}
		if string(encoded) != tc.want {
			t.Errorf("%s: ParseWhere(%q) = %s, want %s", tc.name, tc.where, encoded, tc.want)
		}
	}
}
