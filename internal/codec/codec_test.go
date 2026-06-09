package codec

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestToJSON(t *testing.T) {
	objectID, _ := bson.ObjectIDFromHex("507f1f77bcf86cd799439011")

	cases := []struct {
		name  string
		value any
		want  any
	}{
		{"object id to hex", objectID, "507f1f77bcf86cd799439011"},
		{"datetime to millis", bson.DateTime(1700000000000), int64(1700000000000)},
		{"int32 stays int32", int32(42), int32(42)},
		{"null becomes nil", bson.Null{}, nil},
		{"binary summary", bson.Binary{Data: []byte{1, 2, 3}}, "Binary(3 bytes)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ToJSON(tc.value); got != tc.want {
				t.Errorf("ToJSON(%v) = %v, want %v", tc.value, got, tc.want)
			}
		})
	}
}

func TestParseIDObjectID(t *testing.T) {
	parsed := ParseID("507f1f77bcf86cd799439011")
	if _, ok := parsed.(bson.ObjectID); !ok {
		t.Errorf("expected bson.ObjectID, got %T", parsed)
	}

	plain := ParseID("not-an-object-id")
	if _, ok := plain.(string); !ok {
		t.Errorf("expected string fallback, got %T", plain)
	}
}

func TestPreviewPreservesFieldOrder(t *testing.T) {
	value := bson.D{
		{Key: "discount", Value: int32(0)},
		{Key: "subtotal", Value: int32(80)},
		{Key: "tax", Value: int32(16)},
	}
	want := `{"discount": 0, "subtotal": 80, "tax": 16}`
	if got := Preview(value); got != want {
		t.Errorf("Preview = %q, want %q", got, want)
	}
}

func TestCellLeavesScalarsTyped(t *testing.T) {
	if got := Cell(int32(42)); got != int32(42) {
		t.Errorf("Cell(int32) = %v (%T), want typed 42", got, got)
	}
	if got := Cell(bson.A{int32(1), "x"}); got != `[1, "x"]` {
		t.Errorf("Cell(array) = %v, want array preview", got)
	}
}

func TestParseFilterExtendedJSON(t *testing.T) {
	filter, err := ParseFilter(`{"_id": {"$oid": "507f1f77bcf86cd799439011"}}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(filter) != 1 || filter[0].Key != "_id" {
		t.Fatalf("unexpected filter: %#v", filter)
	}
	if _, ok := filter[0].Value.(bson.ObjectID); !ok {
		t.Errorf("expected extended JSON $oid to decode to ObjectID, got %T", filter[0].Value)
	}
}
