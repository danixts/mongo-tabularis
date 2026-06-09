# mongo-tabularis

A MongoDB driver plugin for [Tabularis](https://github.com/debba/tabularis), written in Go.

It connects Tabularis to any MongoDB deployment — standalone, replica set, sharded cluster or Atlas (`mongodb+srv://`) — with optional TLS/SSL and authentication, comparable to the connectivity offered by tools like DataGrip. Communication happens over **JSON-RPC 2.0 on stdin/stdout**, the language-agnostic protocol Tabularis uses for plugins.

## Features

- Connect via host/port, full connection string, or `mongodb+srv://` DNS seed list.
- Replica sets, `authSource`, `authMechanism`, TLS/SSL and arbitrary URI options.
- Collection browsing and schema inference by sampling documents.
- Index inspection (name, columns, uniqueness, primary).
- Query execution with MongoDB shell syntax: `find`, `findOne`, `aggregate`, `count`, `countDocuments`, `estimatedDocumentCount`.
- Extended JSON filters (e.g. `{"_id": {"$oid": "..."}}`).
- Inline document CRUD from the Tabularis data grid.
- DDL-equivalent script generation (`createCollection`, `createIndex`, `$rename`).

## Connection

The plugin builds its connection in priority order:

1. A full connection string in `connection_string` / `uri` (or pasted into Tabularis' connection-string import) is used verbatim — this enables SRV, replica sets and every URI option.
2. A `mongodb://` / `mongodb+srv://` value in the host field.
3. Individual fields (`host`, `port`, `username`, `password`, `database`, `ssl_mode`) plus optional `srv`, `auth_source`, `auth_mechanism`, `replica_set` and free-form `options`. Credentials are percent-encoded automatically.

## Project layout

```
cmd/tabularis-mongodb-plugin/   entry point, wires the pool and the RPC server
internal/rpc/                   JSON-RPC server and the method registry (handler map)
internal/mongodb/               connection pool, URI builder and MongoDB operations
internal/shell/                 MongoDB shell-syntax query parser
internal/codec/                 BSON <-> JSON conversion and filter parsing
```

Method routing uses a handler registry map (open for extension, closed for modification) rather than a dispatch switch.

## Performance

- Object and array cells are returned as compact inline JSON previews (field
  order preserved), so nested documents are readable in the grid without
  opening each cell.
- Schema inference is cached per collection with a short TTL and pre-warmed in
  the background when collections are listed, so query autocompletion has field
  names available with no perceptible latency. The cache is invalidated on
  insert.
- Schema snapshot and batch-column inference run in parallel with bounded
  concurrency.
- Aggregations paginate via `$skip`/`$limit` pushed into the pipeline instead of
  materialising full result sets.
- The JSON-RPC server processes requests concurrently (bounded), so the
  parallel metadata calls Tabularis issues do not serialise. Responses are
  matched by id and may return out of order.

## Build

```bash
go build -trimpath -ldflags "-s -w" -o tabularis-mongodb-plugin ./cmd/tabularis-mongodb-plugin
```

## Install

```bash
./sync.sh
```

This builds the binary and copies it together with `manifest.json` into the Tabularis plugins directory:

| OS      | Plugins directory                                                     |
|---------|-----------------------------------------------------------------------|
| Linux   | `~/.local/share/tabularis/plugins/mongodb/`                           |
| macOS   | `~/Library/Application Support/com.debba.tabularis/plugins/mongodb/`  |
| Windows | `%APPDATA%\com.debba.tabularis\plugins\mongodb\`                      |

Restart Tabularis afterwards.

## Test

```bash
go test ./...
```

Manual JSON-RPC check:

```bash
echo '{"jsonrpc":"2.0","method":"test_connection","params":{"params":{"host":"localhost","port":27017,"database":"admin"}},"id":1}' \
  | ./tabularis-mongodb-plugin
```

## Tech stack

- Go 1.26
- [mongo-driver/v2](https://github.com/mongodb/mongo-go-driver) (official driver, supports MongoDB 4.0+)
- JSON-RPC 2.0 over stdio

## License

Apache License 2.0
