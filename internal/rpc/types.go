package rpc

import (
	"bytes"
	"encoding/json"

	"github.com/danixts/mongo-tabularis/internal/mongodb"
)

const (
	codeMethodNotFound  = -32601
	codeInternalError   = -32603
	codeConnectionError = -32000
)

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      json.RawMessage `json:"id"`
}

type ColumnRef struct {
	Name string `json:"name"`
}

type Params struct {
	Connection mongodb.ConnParams `json:"params"`
	Schema     *string            `json:"schema"`
	Table      string             `json:"table"`
	TableName  string             `json:"table_name"`
	Query      string             `json:"query"`
	Limit      *uint32            `json:"limit"`
	PageSize   *uint32            `json:"page_size"`
	Page       uint32             `json:"page"`
	Settings   map[string]any     `json:"settings"`
	Data       map[string]any     `json:"data"`
	PKColumn   string             `json:"pk_col"`
	PKValue    any                `json:"pk_val"`
	ColumnName string             `json:"col_name"`
	NewValue   any                `json:"new_val"`
	Columns    []string           `json:"columns"`
	IsUnique   bool               `json:"is_unique"`
	IndexName  string             `json:"index_name"`
	OldColumn  *ColumnRef         `json:"old_column"`
	NewColumn  *ColumnRef         `json:"new_column"`
}

func decodeParams(raw json.RawMessage) (Params, error) {
	var params Params
	if len(bytes.TrimSpace(raw)) == 0 {
		return params, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&params); err != nil {
		return params, err
	}
	return params, nil
}

func identifier(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage("null")
	}
	return raw
}
