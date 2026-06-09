package rpc

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/danixts/mongo-tabularis/internal/mongodb"
)

const (
	readBufferSize    = 1 << 20
	requestTimeout    = 30 * time.Second
	connectionTimeout = 10 * time.Second
)

type Server struct {
	pool     *mongodb.Pool
	handlers map[string]handler
	reader   *bufio.Reader
	writer   *responseWriter
	logger   io.Writer
	rootCtx  context.Context
}

func NewServer(pool *mongodb.Pool, in io.Reader, out io.Writer, logger io.Writer) *Server {
	return &Server{
		pool:     pool,
		handlers: newRegistry(),
		reader:   bufio.NewReaderSize(in, readBufferSize),
		writer:   newResponseWriter(out),
		logger:   logger,
		rootCtx:  context.Background(),
	}
}

func (s *Server) Run() error {
	for {
		line, err := s.reader.ReadBytes('\n')
		if trimmed := bytes.TrimSpace(line); len(trimmed) > 0 {
			s.handle(trimmed)
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

func (s *Server) handle(line []byte) {
	var request Request
	if err := json.Unmarshal(line, &request); err != nil {
		s.log("failed to parse request: " + err.Error())
		return
	}

	target, ok := s.handlers[request.Method]
	if !ok {
		s.writer.failure(request.ID, codeMethodNotFound, "method "+request.Method+" not implemented")
		return
	}

	params, err := decodeParams(request.Params)
	if err != nil {
		s.writer.failure(request.ID, codeInternalError, "invalid params: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(s.rootCtx, requestTimeout)
	defer cancel()

	active := session{ctx: ctx, params: params}

	if target.needsConnection {
		connectCtx, connectCancel := context.WithTimeout(ctx, connectionTimeout)
		defer connectCancel()

		client, database, connErr := s.pool.Acquire(connectCtx, params.Connection)
		if connErr != nil {
			s.writer.failure(request.ID, codeConnectionError, connErr.Error())
			return
		}
		active.client = client
		active.database = database
	}

	result, runErr := target.run(active)
	if runErr != nil {
		s.writer.failure(request.ID, codeInternalError, runErr.Error())
		return
	}
	s.writer.success(request.ID, result)
}

func (s *Server) log(message string) {
	if s.logger != nil {
		io.WriteString(s.logger, message+"\n")
	}
}

type responseWriter struct {
	mu      sync.Mutex
	encoder *json.Encoder
}

func newResponseWriter(out io.Writer) *responseWriter {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	return &responseWriter{encoder: encoder}
}

func (w *responseWriter) success(id json.RawMessage, result any) {
	w.write(map[string]any{"jsonrpc": "2.0", "result": result, "id": identifier(id)})
}

func (w *responseWriter) failure(id json.RawMessage, code int, message string) {
	w.write(map[string]any{
		"jsonrpc": "2.0",
		"error":   map[string]any{"code": code, "message": message},
		"id":      identifier(id),
	})
}

func (w *responseWriter) write(payload map[string]any) {
	w.mu.Lock()
	defer w.mu.Unlock()
	_ = w.encoder.Encode(payload)
}
