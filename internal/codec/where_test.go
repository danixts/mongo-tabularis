package codec

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

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
