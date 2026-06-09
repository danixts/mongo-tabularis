package rpc

import (
	"context"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/danixts/mongo-tabularis/internal/mongodb"
)

type session struct {
	ctx      context.Context
	client   *mongo.Client
	database string
	params   Params
}

type handlerFunc func(session) (any, error)

type handler struct {
	needsConnection bool
	run             handlerFunc
}

func connected(run handlerFunc) handler {
	return handler{needsConnection: true, run: run}
}

func offline(run handlerFunc) handler {
	return handler{needsConnection: false, run: run}
}

func constant(value any) handler {
	return offline(func(session) (any, error) { return value, nil })
}

func unsupported(message string) handler {
	return offline(func(session) (any, error) { return nil, fmt.Errorf("%s", message) })
}

func newRegistry() map[string]handler {
	emptyList := []any{}
	return map[string]handler{
		"test_connection": connected(func(s session) (any, error) {
			return mongodb.TestConnection(s.ctx, s.client, s.database)
		}),
		"get_databases": connected(func(s session) (any, error) {
			return mongodb.ListDatabases(s.ctx, s.client, s.database), nil
		}),
		"get_tables": connected(func(s session) (any, error) {
			return mongodb.ListCollections(s.ctx, s.client, s.database)
		}),
		"get_columns": connected(func(s session) (any, error) {
			return mongodb.InferColumns(s.ctx, s.client, s.database, s.params.Table)
		}),
		"get_indexes": connected(func(s session) (any, error) {
			return mongodb.ListIndexes(s.ctx, s.client, s.database, s.params.Table)
		}),
		"execute_query": connected(func(s session) (any, error) {
			return mongodb.ExecuteQuery(s.ctx, s.client, s.database, s.params.Query, s.params.Limit, s.params.Page)
		}),
		"insert_record": connected(func(s session) (any, error) {
			return mongodb.InsertRecord(s.ctx, s.client, s.database, s.params.Table, s.params.Data)
		}),
		"update_record": connected(func(s session) (any, error) {
			return mongodb.UpdateRecord(s.ctx, s.client, s.database, s.params.Table, primaryKeyColumn(s.params), s.params.PKValue, s.params.ColumnName, s.params.NewValue)
		}),
		"delete_record": connected(func(s session) (any, error) {
			return mongodb.DeleteRecord(s.ctx, s.client, s.database, s.params.Table, primaryKeyColumn(s.params), s.params.PKValue)
		}),
		"get_schema_snapshot": connected(func(s session) (any, error) {
			return mongodb.SchemaSnapshot(s.ctx, s.client, s.database)
		}),
		"get_all_columns_batch": connected(func(s session) (any, error) {
			return mongodb.AllColumnsBatch(s.ctx, s.client, s.database)
		}),
		"drop_index": connected(func(s session) (any, error) {
			return nil, mongodb.DropIndex(s.ctx, s.client, s.database, s.params.Table, s.params.IndexName)
		}),

		"get_schemas":            constant(emptyList),
		"get_foreign_keys":       constant(emptyList),
		"get_views":              constant(emptyList),
		"get_view_definition":    constant(emptyList),
		"get_view_columns":       constant(emptyList),
		"get_routines":           constant(emptyList),
		"get_routine_parameters": constant(emptyList),

		"get_all_foreign_keys_batch": constant(map[string]any{}),

		"get_create_table_sql":       offline(createCollectionScript),
		"get_add_column_sql":         constant([]string{"// MongoDB is schemaless — fields are added automatically on insert"}),
		"get_alter_column_sql":       offline(renameFieldScript),
		"get_create_index_sql":       offline(createIndexScript),
		"get_create_foreign_key_sql": constant([]string{"// MongoDB does not support foreign key constraints"}),

		"create_view":            unsupported("Views are not supported in this driver"),
		"alter_view":             unsupported("Views are not supported in this driver"),
		"drop_view":              unsupported("Views are not supported in this driver"),
		"get_routine_definition": unsupported("MongoDB does not support stored routines"),
		"drop_foreign_key":       unsupported("MongoDB does not support foreign key constraints"),
	}
}

func primaryKeyColumn(params Params) string {
	if params.PKColumn == "" {
		return "_id"
	}
	return params.PKColumn
}

func createCollectionScript(s session) (any, error) {
	return []string{fmt.Sprintf("db.createCollection(%q)", s.params.TableName)}, nil
}

func renameFieldScript(s session) (any, error) {
	if s.params.OldColumn == nil || s.params.NewColumn == nil {
		return []string{"// No rename needed in MongoDB"}, nil
	}
	oldName := s.params.OldColumn.Name
	newName := s.params.NewColumn.Name
	if oldName == "" || oldName == newName {
		return []string{"// No rename needed in MongoDB"}, nil
	}
	return []string{fmt.Sprintf("db.%s.updateMany({}, {%q: {%q: %q}})", s.params.Table, "$rename", oldName, newName)}, nil
}

func createIndexScript(s session) (any, error) {
	keys := make([]string, 0, len(s.params.Columns))
	for _, column := range s.params.Columns {
		keys = append(keys, fmt.Sprintf("%q: 1", column))
	}
	uniqueOption := ""
	if s.params.IsUnique {
		uniqueOption = ", { unique: true }"
	}
	return []string{fmt.Sprintf("db.%s.createIndex({ %s }%s)", s.params.Table, strings.Join(keys, ", "), uniqueOption)}, nil
}
