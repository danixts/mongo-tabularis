package codec

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func Cell(value any) any {
	switch value.(type) {
	case bson.D, bson.M, bson.A, []any:
		return Preview(value)
	default:
		return ToJSON(value)
	}
}

func Preview(value any) string {
	var builder strings.Builder
	writePreview(&builder, value)
	return builder.String()
}

func writePreview(builder *strings.Builder, value any) {
	switch v := value.(type) {
	case bson.D:
		builder.WriteByte('{')
		for i, entry := range v {
			if i > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(strconv.Quote(entry.Key))
			builder.WriteString(": ")
			writePreview(builder, entry.Value)
		}
		builder.WriteByte('}')
	case bson.M:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		builder.WriteByte('{')
		for i, key := range keys {
			if i > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(strconv.Quote(key))
			builder.WriteString(": ")
			writePreview(builder, v[key])
		}
		builder.WriteByte('}')
	case bson.A:
		writeArray(builder, v)
	case []any:
		writeArray(builder, v)
	default:
		writeScalar(builder, value)
	}
}

func writeArray(builder *strings.Builder, items []any) {
	builder.WriteByte('[')
	for i, item := range items {
		if i > 0 {
			builder.WriteString(", ")
		}
		writePreview(builder, item)
	}
	builder.WriteByte(']')
}

func writeScalar(builder *strings.Builder, value any) {
	switch v := value.(type) {
	case nil, bson.Null, bson.Undefined, bson.MinKey, bson.MaxKey:
		builder.WriteString("null")
	case string:
		builder.WriteString(strconv.Quote(v))
	case bool:
		builder.WriteString(strconv.FormatBool(v))
	case int32:
		builder.WriteString(strconv.FormatInt(int64(v), 10))
	case int64:
		builder.WriteString(strconv.FormatInt(v, 10))
	case int:
		builder.WriteString(strconv.Itoa(v))
	case float64:
		builder.WriteString(strconv.FormatFloat(v, 'g', -1, 64))
	case float32:
		builder.WriteString(strconv.FormatFloat(float64(v), 'g', -1, 32))
	case bson.ObjectID:
		builder.WriteString(`ObjectId("`)
		builder.WriteString(v.Hex())
		builder.WriteString(`")`)
	case bson.DateTime:
		builder.WriteString(strconv.Quote(time.UnixMilli(int64(v)).UTC().Format(time.RFC3339)))
	case bson.Decimal128:
		builder.WriteString(v.String())
	default:
		builder.WriteString(strconv.Quote(fmt.Sprintf("%v", v)))
	}
}
