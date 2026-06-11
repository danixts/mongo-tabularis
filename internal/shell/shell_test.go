package shell

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		wantOK    bool
		wantQuery Query
	}{
		{
			name:      "find with filter",
			input:     `db.users.find({"age": {"$gt": 18}})`,
			wantOK:    true,
			wantQuery: Query{Collection: "users", Operation: "find", Args: []string{`{"age": {"$gt": 18}}`}},
		},
		{
			name:      "find with projection",
			input:     `db.users.find({"active": true}, {"name": 1})`,
			wantOK:    true,
			wantQuery: Query{Collection: "users", Operation: "find", Args: []string{`{"active": true}`, `{"name": 1}`}},
		},
		{
			name:      "aggregate trailing semicolon",
			input:     `db.orders.aggregate([{"$match": {"status": "shipped"}}]);`,
			wantOK:    true,
			wantQuery: Query{Collection: "orders", Operation: "aggregate", Args: []string{`[{"$match": {"status": "shipped"}}]`}},
		},
		{
			name:      "comma inside string is not a separator",
			input:     `db.people.find({"name": "Doe, John"})`,
			wantOK:    true,
			wantQuery: Query{Collection: "people", Operation: "find", Args: []string{`{"name": "Doe, John"}`}},
		},
		{
			name:   "not a shell query",
			input:  "SELECT * FROM users",
			wantOK: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := Parse(tc.input)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && !reflect.DeepEqual(got, tc.wantQuery) {
				t.Errorf("query = %#v, want %#v", got, tc.wantQuery)
			}
		})
	}
}

func TestParseSQL(t *testing.T) {
	cases := []struct {
		input  string
		want   SQLSelect
		wantOK bool
	}{
		{"SELECT * FROM users", SQLSelect{Collection: "users"}, true},
		{"select * from `orders`", SQLSelect{Collection: "orders"}, true},
		{`SELECT * FROM "events" WHERE total = 96`, SQLSelect{Collection: "events", Where: "total = 96"}, true},
		{`SELECT * FROM "events" WHERE { body.total : 96 }`, SQLSelect{Collection: "events", Where: "{ body.total : 96 }"}, true},
		{`SELECT * FROM "events" ORDER BY createdAt DESC`, SQLSelect{Collection: "events", OrderBy: "createdAt DESC"}, true},
		{`SELECT * FROM "events" WHERE total = 96 ORDER BY createdAt DESC`, SQLSelect{Collection: "events", Where: "total = 96", OrderBy: "createdAt DESC"}, true},
		{`SELECT * FROM "events" WHERE a = 1 ORDER BY b ASC LIMIT 50`, SQLSelect{Collection: "events", Where: "a = 1", OrderBy: "b ASC"}, true},
		{"db.users.find({})", SQLSelect{}, false},
	}
	for _, tc := range cases {
		got, ok := ParseSQL(tc.input)
		if ok != tc.wantOK || got != tc.want {
			t.Errorf("ParseSQL(%q) = %#v,%v want %#v,%v", tc.input, got, ok, tc.want, tc.wantOK)
		}
	}
}

func TestParseCollectionShorthand(t *testing.T) {
	got, ok := Parse("db.events()")
	if !ok || got.Collection != "events" || got.Operation != "find" {
		t.Errorf("Parse(db.events()) = %#v,%v", got, ok)
	}
}
