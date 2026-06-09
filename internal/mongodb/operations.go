package mongodb

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/danixts/mongo-tabularis/internal/codec"
	"github.com/danixts/mongo-tabularis/internal/shell"
)

const sampleSize = 100

type Column struct {
	Name            string `json:"name"`
	DataType        string `json:"data_type"`
	IsPK            bool   `json:"is_pk"`
	IsNullable      bool   `json:"is_nullable"`
	IsAutoIncrement bool   `json:"is_auto_increment"`
	DefaultValue    any    `json:"default_value"`
}

type Index struct {
	Name      string   `json:"name"`
	Columns   []string `json:"columns"`
	IsUnique  bool     `json:"is_unique"`
	IsPrimary bool     `json:"is_primary"`
}

type Collection struct {
	Name    string `json:"name"`
	Schema  any    `json:"schema"`
	Comment any    `json:"comment"`
}

type CollectionSnapshot struct {
	Name        string   `json:"name"`
	Schema      any      `json:"schema"`
	Comment     any      `json:"comment"`
	Columns     []Column `json:"columns"`
	ForeignKeys []any    `json:"foreign_keys"`
}

type Pagination struct {
	Page      uint32 `json:"page"`
	PageSize  uint32 `json:"page_size"`
	TotalRows any    `json:"total_rows"`
	HasMore   bool   `json:"has_more"`
}

type QueryResult struct {
	Columns      []string `json:"columns"`
	Rows         [][]any  `json:"rows"`
	AffectedRows int64    `json:"affected_rows"`
	Truncated    bool     `json:"truncated"`
	HasMore      bool     `json:"has_more"`
	Pagination   any      `json:"pagination"`
}

func TestConnection(ctx context.Context, client *mongo.Client, database string) (any, error) {
	if err := client.Database(database).RunCommand(ctx, bson.D{{Key: "ping", Value: 1}}).Err(); err != nil {
		return nil, err
	}
	return map[string]any{"success": true}, nil
}

func ListDatabases(ctx context.Context, client *mongo.Client, database string) any {
	names, err := client.ListDatabaseNames(ctx, bson.D{})
	if err != nil {
		return []string{database}
	}
	return names
}

func ListCollections(ctx context.Context, client *mongo.Client, database string) (any, error) {
	names, err := client.Database(database).ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	collections := make([]Collection, 0, len(names))
	for _, name := range names {
		collections = append(collections, Collection{Name: name})
	}
	return collections, nil
}

func InferColumns(ctx context.Context, client *mongo.Client, database, collection string) ([]Column, error) {
	cursor, err := client.Database(database).Collection(collection).Find(ctx, bson.D{}, options.Find().SetLimit(sampleSize))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	observedTypes := map[string][]string{}
	occurrences := map[string]int{}
	idType := "ObjectId"
	idObserved := false
	totalDocuments := 0

	for cursor.Next(ctx) {
		var document bson.D
		if err := cursor.Decode(&document); err != nil {
			return nil, err
		}
		totalDocuments++
		for _, field := range document {
			occurrences[field.Key]++
			typeName := codec.BSONTypeName(field.Value)
			if field.Key == "_id" {
				if !idObserved {
					idType = typeName
					idObserved = true
				}
				continue
			}
			observedTypes[field.Key] = append(observedTypes[field.Key], typeName)
		}
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}

	columns := make([]Column, 0, len(observedTypes)+1)
	columns = append(columns, Column{
		Name: "_id", DataType: idType, IsPK: true, IsAutoIncrement: true,
	})

	fieldNames := make([]string, 0, len(observedTypes))
	for name := range observedTypes {
		fieldNames = append(fieldNames, name)
	}
	sort.Strings(fieldNames)

	for _, name := range fieldNames {
		columns = append(columns, Column{
			Name:       name,
			DataType:   dominantType(observedTypes[name]),
			IsNullable: occurrences[name] < totalDocuments,
		})
	}
	return columns, nil
}

func ListIndexes(ctx context.Context, client *mongo.Client, database, collection string) (any, error) {
	cursor, err := client.Database(database).Collection(collection).Indexes().List(ctx)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	indexes := []Index{}
	for cursor.Next(ctx) {
		var raw bson.M
		if err := cursor.Decode(&raw); err != nil {
			return nil, err
		}
		name, _ := raw["name"].(string)
		unique, _ := raw["unique"].(bool)
		indexes = append(indexes, Index{
			Name:      name,
			Columns:   indexColumns(raw["key"]),
			IsUnique:  unique,
			IsPrimary: name == "_id_",
		})
	}
	return indexes, cursor.Err()
}

func ExecuteQuery(ctx context.Context, client *mongo.Client, database, query string, limit *uint32, page uint32) (any, error) {
	if parsed, ok := shell.Parse(query); ok {
		return dispatchShellQuery(ctx, client, database, parsed, limit, page)
	}
	if collection, ok := shell.ParseSQLFrom(query); ok {
		return executeFind(ctx, client, database, collection, bson.D{}, nil, limit, page)
	}
	return nil, fmt.Errorf("invalid query format. Use MongoDB shell syntax:\n  db.collection.find({})\n  db.collection.aggregate([...])")
}

type shellOperation func(ctx context.Context, client *mongo.Client, database string, query shell.Query, limit *uint32, page uint32) (any, error)

var shellOperations = map[string]shellOperation{
	"find":                   operationFind,
	"findOne":                operationFindOne,
	"aggregate":              operationAggregate,
	"count":                  operationCount,
	"countDocuments":         operationCount,
	"estimatedDocumentCount": operationEstimatedCount,
}

func supportedShellOperations() []string {
	names := make([]string, 0, len(shellOperations))
	for name := range shellOperations {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func dispatchShellQuery(ctx context.Context, client *mongo.Client, database string, query shell.Query, limit *uint32, page uint32) (any, error) {
	operation, ok := shellOperations[query.Operation]
	if !ok {
		return nil, fmt.Errorf("unsupported operation %q. Supported: %s", query.Operation, strings.Join(supportedShellOperations(), ", "))
	}
	return operation(ctx, client, database, query, limit, page)
}

func operationFind(ctx context.Context, client *mongo.Client, database string, query shell.Query, limit *uint32, page uint32) (any, error) {
	filter, err := codec.ParseFilter(query.Arg(0))
	if err != nil {
		return nil, err
	}
	var projection bson.D
	if raw := query.Arg(1); raw != "" {
		if projection, err = codec.ParseFilter(raw); err != nil {
			return nil, err
		}
	}
	return executeFind(ctx, client, database, query.Collection, filter, projection, limit, page)
}

func operationFindOne(ctx context.Context, client *mongo.Client, database string, query shell.Query, _ *uint32, _ uint32) (any, error) {
	filter, err := codec.ParseFilter(query.Arg(0))
	if err != nil {
		return nil, err
	}
	one := uint32(1)
	return executeFind(ctx, client, database, query.Collection, filter, nil, &one, 1)
}

func operationAggregate(ctx context.Context, client *mongo.Client, database string, query shell.Query, limit *uint32, page uint32) (any, error) {
	pipeline, err := codec.ParsePipeline(query.Arg(0))
	if err != nil {
		return nil, err
	}
	return executeAggregate(ctx, client, database, query.Collection, pipeline, limit, page)
}

func operationCount(ctx context.Context, client *mongo.Client, database string, query shell.Query, _ *uint32, _ uint32) (any, error) {
	filter, err := codec.ParseFilter(query.Arg(0))
	if err != nil {
		return nil, err
	}
	total, err := client.Database(database).Collection(query.Collection).CountDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}
	return countResult(total), nil
}

func operationEstimatedCount(ctx context.Context, client *mongo.Client, database string, query shell.Query, _ *uint32, _ uint32) (any, error) {
	total, err := client.Database(database).Collection(query.Collection).EstimatedDocumentCount(ctx)
	if err != nil {
		return nil, err
	}
	return countResult(total), nil
}

func executeFind(ctx context.Context, client *mongo.Client, database, collection string, filter, projection bson.D, limit *uint32, page uint32) (any, error) {
	if page == 0 {
		page = 1
	}
	findOptions := options.Find()
	if limit != nil {
		findOptions.SetSkip(int64((page - 1) * *limit))
		findOptions.SetLimit(int64(*limit) + 1)
	}
	if projection != nil {
		findOptions.SetProjection(projection)
	}

	cursor, err := client.Database(database).Collection(collection).Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	documents, err := decodeDocuments(ctx, cursor)
	if err != nil {
		return nil, err
	}

	hasMore := limit != nil && len(documents) > int(*limit)
	if hasMore {
		documents = documents[:*limit]
	}
	return buildResult(documents, limit, page, hasMore, nil), nil
}

func executeAggregate(ctx context.Context, client *mongo.Client, database, collection string, pipeline bson.A, limit *uint32, page uint32) (any, error) {
	if page == 0 {
		page = 1
	}
	if limit != nil {
		skip := int64((page - 1) * *limit)
		pipeline = append(append(bson.A{}, pipeline...),
			bson.D{{Key: "$skip", Value: skip}},
			bson.D{{Key: "$limit", Value: int64(*limit) + 1}},
		)
	}

	cursor, err := client.Database(database).Collection(collection).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	documents, err := decodeDocuments(ctx, cursor)
	if err != nil {
		return nil, err
	}

	hasMore := limit != nil && len(documents) > int(*limit)
	if hasMore {
		documents = documents[:*limit]
	}
	return buildResult(documents, limit, page, hasMore, nil), nil
}

func InsertRecord(ctx context.Context, client *mongo.Client, database, collection string, data map[string]any) (int64, error) {
	document := bson.D{}
	for key, value := range data {
		if key == "_id" {
			if value == nil {
				continue
			}
			document = append(document, bson.E{Key: key, Value: codec.ParseID(value)})
			continue
		}
		document = append(document, bson.E{Key: key, Value: codec.FromJSON(value)})
	}
	if _, err := client.Database(database).Collection(collection).InsertOne(ctx, document); err != nil {
		return 0, err
	}
	return 1, nil
}

func UpdateRecord(ctx context.Context, client *mongo.Client, database, collection, pkColumn string, pkValue any, columnName string, newValue any) (int64, error) {
	filter := bson.D{{Key: pkColumn, Value: primaryKeyValue(pkColumn, pkValue)}}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: columnName, Value: codec.FromJSON(newValue)}}}}
	result, err := client.Database(database).Collection(collection).UpdateOne(ctx, filter, update)
	if err != nil {
		return 0, err
	}
	return result.ModifiedCount, nil
}

func DeleteRecord(ctx context.Context, client *mongo.Client, database, collection, pkColumn string, pkValue any) (int64, error) {
	filter := bson.D{{Key: pkColumn, Value: primaryKeyValue(pkColumn, pkValue)}}
	result, err := client.Database(database).Collection(collection).DeleteOne(ctx, filter)
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}

func DropIndex(ctx context.Context, client *mongo.Client, database, collection, indexName string) error {
	return client.Database(database).Collection(collection).Indexes().DropOne(ctx, indexName)
}

func SchemaSnapshot(ctx context.Context, client *mongo.Client, database string) (any, error) {
	names, err := client.Database(database).ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	snapshots := make([]CollectionSnapshot, len(names))
	runBounded(len(names), schemaConcurrency, func(index int) {
		snapshots[index] = CollectionSnapshot{
			Name:        names[index],
			Columns:     inferColumnsOrEmpty(ctx, client, database, names[index]),
			ForeignKeys: []any{},
		}
	})
	return snapshots, nil
}

func AllColumnsBatch(ctx context.Context, client *mongo.Client, database string) (any, error) {
	names, err := client.Database(database).ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	columnsByIndex := make([][]Column, len(names))
	runBounded(len(names), schemaConcurrency, func(index int) {
		columnsByIndex[index] = inferColumnsOrEmpty(ctx, client, database, names[index])
	})
	result := make(map[string]any, len(names))
	for index, name := range names {
		result[name] = columnsByIndex[index]
	}
	return result, nil
}

func inferColumnsOrEmpty(ctx context.Context, client *mongo.Client, database, collection string) []Column {
	columns, err := InferColumns(ctx, client, database, collection)
	if err != nil {
		return []Column{}
	}
	return columns
}

func countResult(total int64) QueryResult {
	return QueryResult{Columns: []string{"count"}, Rows: [][]any{{total}}}
}

func buildResult(documents []bson.D, limit *uint32, page uint32, hasMore bool, total any) QueryResult {
	columns := collectColumns(documents)
	result := QueryResult{
		Columns:   columns,
		Rows:      documentsToRows(documents, columns),
		Truncated: hasMore,
		HasMore:   hasMore,
	}
	if limit != nil {
		result.Pagination = Pagination{Page: page, PageSize: *limit, TotalRows: total, HasMore: hasMore}
	}
	if result.Rows == nil {
		result.Rows = [][]any{}
	}
	return result
}

func decodeDocuments(ctx context.Context, cursor *mongo.Cursor) ([]bson.D, error) {
	var documents []bson.D
	for cursor.Next(ctx) {
		var document bson.D
		if err := cursor.Decode(&document); err != nil {
			return nil, err
		}
		documents = append(documents, document)
	}
	return documents, cursor.Err()
}

func collectColumns(documents []bson.D) []string {
	seen := map[string]bool{}
	var columns []string

	for _, document := range documents {
		for _, field := range document {
			if field.Key == "_id" {
				seen["_id"] = true
				columns = append(columns, "_id")
				break
			}
		}
		if seen["_id"] {
			break
		}
	}
	for _, document := range documents {
		for _, field := range document {
			if !seen[field.Key] {
				seen[field.Key] = true
				columns = append(columns, field.Key)
			}
		}
	}
	return columns
}

func documentsToRows(documents []bson.D, columns []string) [][]any {
	rows := make([][]any, 0, len(documents))
	for _, document := range documents {
		byKey := make(map[string]any, len(document))
		for _, field := range document {
			byKey[field.Key] = field.Value
		}
		row := make([]any, len(columns))
		for i, column := range columns {
			row[i] = codec.Cell(byKey[column])
		}
		rows = append(rows, row)
	}
	return rows
}

func dominantType(types []string) string {
	counts := map[string]int{}
	best, bestCount := "Mixed", 0
	for _, typeName := range types {
		counts[typeName]++
		if counts[typeName] > bestCount {
			best, bestCount = typeName, counts[typeName]
		}
	}
	return best
}

func indexColumns(key any) []string {
	var columns []string
	switch value := key.(type) {
	case bson.D:
		for _, field := range value {
			columns = append(columns, field.Key)
		}
	case bson.M:
		for name := range value {
			columns = append(columns, name)
		}
		sort.Strings(columns)
	}
	return columns
}

func primaryKeyValue(pkColumn string, pkValue any) any {
	if pkColumn == "_id" {
		return codec.ParseID(pkValue)
	}
	return codec.FromJSON(pkValue)
}
