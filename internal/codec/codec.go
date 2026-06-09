package codec

import (
	"encoding/json"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func BSONTypeName(value any) string {
	switch value.(type) {
	case float64, float32:
		return "Double"
	case string:
		return "String"
	case bson.A, []any:
		return "Array"
	case bson.D, bson.M:
		return "Object"
	case bool:
		return "Boolean"
	case nil, bson.Null:
		return "Null"
	case bson.Regex:
		return "RegExp"
	case bson.JavaScript, bson.CodeWithScope:
		return "JavaScript"
	case int32:
		return "Int32"
	case int64, int:
		return "Int64"
	case bson.Timestamp:
		return "Timestamp"
	case bson.Binary:
		return "Binary"
	case bson.ObjectID:
		return "ObjectId"
	case bson.DateTime:
		return "Date"
	case bson.Decimal128:
		return "Decimal128"
	case bson.Symbol:
		return "String"
	default:
		return "Unknown"
	}
}

func ToJSON(value any) any {
	switch v := value.(type) {
	case nil, bson.Null, bson.Undefined, bson.MinKey, bson.MaxKey:
		return nil
	case bson.D:
		object := make(map[string]any, len(v))
		for _, entry := range v {
			object[entry.Key] = ToJSON(entry.Value)
		}
		return object
	case bson.M:
		object := make(map[string]any, len(v))
		for key, entry := range v {
			object[key] = ToJSON(entry)
		}
		return object
	case bson.A:
		array := make([]any, len(v))
		for i, entry := range v {
			array[i] = ToJSON(entry)
		}
		return array
	case []any:
		return ToJSON(bson.A(v))
	case bson.ObjectID:
		return v.Hex()
	case bson.DateTime:
		return int64(v)
	case bson.Timestamp:
		return v.T
	case bson.Decimal128:
		return v.String()
	case bson.Binary:
		return fmt.Sprintf("Binary(%d bytes)", len(v.Data))
	case bson.Regex:
		return fmt.Sprintf("/%s/%s", v.Pattern, v.Options)
	case bson.JavaScript:
		return string(v)
	case bson.CodeWithScope:
		return string(v.Code)
	case bson.Symbol:
		return string(v)
	case json.Number:
		if integer, err := v.Int64(); err == nil {
			return integer
		}
		float, _ := v.Float64()
		return float
	default:
		return v
	}
}

func FromJSON(value any) any {
	switch v := value.(type) {
	case nil:
		return bson.Null{}
	case bool, string:
		return v
	case json.Number:
		if integer, err := v.Int64(); err == nil {
			return integer
		}
		float, _ := v.Float64()
		return float
	case float64:
		if v == float64(int64(v)) {
			return int64(v)
		}
		return v
	case []any:
		array := make(bson.A, len(v))
		for i, entry := range v {
			array[i] = FromJSON(entry)
		}
		return array
	case map[string]any:
		document := bson.D{}
		for key, entry := range v {
			document = append(document, bson.E{Key: key, Value: FromJSON(entry)})
		}
		return document
	default:
		return v
	}
}

func ParseID(value any) any {
	if text, ok := value.(string); ok {
		if objectID, err := bson.ObjectIDFromHex(text); err == nil {
			return objectID
		}
		return text
	}
	return FromJSON(value)
}

func ParseFilter(text string) (bson.D, error) {
	text = strings.TrimSpace(text)
	if text == "" || text == "{}" {
		return bson.D{}, nil
	}
	var document bson.D
	if err := bson.UnmarshalExtJSON([]byte(text), false, &document); err != nil {
		return nil, fmt.Errorf("invalid filter JSON: %w", err)
	}
	return document, nil
}

func ParsePipeline(text string) (bson.A, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return bson.A{}, nil
	}
	var pipeline bson.A
	if err := bson.UnmarshalExtJSON([]byte(text), false, &pipeline); err != nil {
		return nil, fmt.Errorf("invalid pipeline JSON: %w", err)
	}
	return pipeline, nil
}
