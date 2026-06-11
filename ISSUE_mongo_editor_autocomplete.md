# Feature request: Mongo-aware autocomplete in the query editor for plugin drivers

## Summary

The query editor (Monaco) offers schema-based autocomplete for SQL drivers
(table/column suggestions in `SELECT ... FROM ...`). Plugin drivers whose query
language is **not** SQL — e.g. a MongoDB driver using shell syntax
`db.collection.find({ ... })` — get no field autocomplete in the editor, even
though the plugin already exposes the full schema via `get_tables`,
`get_columns`, `get_schema_snapshot` and `get_all_columns_batch`.

## Current behaviour

- The plugin returns collections and their inferred fields/types through the
  standard schema RPC methods (verified: `get_all_columns_batch` and
  `get_schema_snapshot` return field names + BSON types per collection).
- In the editor, typing inside `db.events.find({ ... })` produces no
  suggestions, because the host completion provider is SQL-oriented and does not
  recognise the `db.<collection>.<field>` context.

## Requested behaviour

A way for a plugin to drive editor completions for its own query language. A few
possible shapes:

1. A manifest capability such as `query_language: "mongodb"` (or a generic
   `editor_language`) that selects a non-SQL completion strategy, plus a
   completion source fed from the plugin's schema (collection names after
   `db.`, field names + types inside the document literal).
2. A plugin RPC like `get_completions({ query, cursor })` the host can call to
   request context-aware suggestions, letting the plugin decide what to offer.
3. Exposing the loaded schema (collections + fields + types) to a registered
   completion provider keyed by driver id, so plugins can supply a Monaco
   `CompletionItemProvider`.

## Why

NoSQL/document plugins are first-class via the JSON-RPC interface, but the
editor experience lags SQL drivers specifically because of completion. Today the
only workaround is a custom UI extension (e.g. a filter panel with its own
typeahead), which can't reach into the main editor.

## Context

- Plugin: a Go MongoDB driver (`mongodb`) speaking JSON-RPC 2.0 over stdio.
- Tabularis: 0.13.1.
- The plugin already implements the full schema-discovery method set and caches
  it, so the metadata needed for completions is readily available.
