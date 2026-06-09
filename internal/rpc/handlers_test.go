package rpc

import "testing"

func TestEffectiveLimitPrefersPageSize(t *testing.T) {
	limit := uint32(10)
	pageSize := uint32(25)

	cases := []struct {
		name   string
		params Params
		want   *uint32
	}{
		{"page_size wins over limit", Params{Limit: &limit, PageSize: &pageSize}, &pageSize},
		{"falls back to limit", Params{Limit: &limit}, &limit},
		{"none set", Params{}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := effectiveLimit(tc.params)
			if got != tc.want {
				t.Errorf("effectiveLimit = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPrimaryKeyColumnDefaults(t *testing.T) {
	if got := primaryKeyColumn(Params{}); got != "_id" {
		t.Errorf("primaryKeyColumn default = %q, want _id", got)
	}
	if got := primaryKeyColumn(Params{PKColumn: "uuid"}); got != "uuid" {
		t.Errorf("primaryKeyColumn = %q, want uuid", got)
	}
}
