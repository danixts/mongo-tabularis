package mongodb

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

const (
	dialTimeout = 8 * time.Second
	maxPoolSize = 50
)

type ConnParams struct {
	Driver   string `json:"driver"`
	Host     string `json:"host"`
	Port     *int   `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
	SSLMode  string `json:"ssl_mode"`

	ConnectionString string `json:"connection_string"`
	URI              string `json:"uri"`
	SRV              *bool  `json:"srv"`
	AuthSource       string `json:"auth_source"`
	AuthMechanism    string `json:"auth_mechanism"`
	ReplicaSet       string `json:"replica_set"`
	ExtraOptions     string `json:"options"`
}

func (p *ConnParams) UnmarshalJSON(data []byte) error {
	type raw ConnParams
	aux := struct {
		Port    any `json:"port"`
		SSLMode any `json:"ssl_mode"`
		SRV     any `json:"srv"`
		*raw
	}{raw: (*raw)(p)}

	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&aux); err != nil {
		return err
	}
	p.Port = coerceInt(aux.Port)
	p.SSLMode = coerceString(aux.SSLMode)
	p.SRV = coerceBool(aux.SRV)
	return nil
}

type Pool struct {
	mu      sync.Mutex
	clients map[string]*mongo.Client
	locks   map[string]*sync.Mutex
}

func NewPool() *Pool {
	return &Pool{
		clients: make(map[string]*mongo.Client),
		locks:   make(map[string]*sync.Mutex),
	}
}

func (pool *Pool) Acquire(ctx context.Context, params ConnParams) (*mongo.Client, string, error) {
	uri, database := buildURI(params)

	if client, ok := pool.lookup(uri); ok {
		return client, database, nil
	}

	lock := pool.lockFor(uri)
	lock.Lock()
	defer lock.Unlock()

	if client, ok := pool.lookup(uri); ok {
		return client, database, nil
	}

	client, err := connect(ctx, uri, params.SSLMode)
	if err != nil {
		return nil, database, err
	}

	pool.mu.Lock()
	pool.clients[uri] = client
	pool.mu.Unlock()
	return client, database, nil
}

func (pool *Pool) lookup(uri string) (*mongo.Client, bool) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	client, ok := pool.clients[uri]
	return client, ok
}

func (pool *Pool) lockFor(uri string) *sync.Mutex {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	lock, ok := pool.locks[uri]
	if !ok {
		lock = &sync.Mutex{}
		pool.locks[uri] = lock
	}
	return lock
}

func (pool *Pool) Close() {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	for _, client := range pool.clients {
		_ = client.Disconnect(context.Background())
	}
}

func connect(ctx context.Context, uri, sslMode string) (*mongo.Client, error) {
	clientOptions := options.Client().
		ApplyURI(uri).
		SetServerSelectionTimeout(dialTimeout).
		SetConnectTimeout(dialTimeout).
		SetMaxPoolSize(maxPoolSize)
	applyTLS(clientOptions, sslMode)

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create MongoDB client: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()
	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	return client, nil
}

func buildURI(params ConnParams) (uri string, database string) {
	database = strings.TrimSpace(params.Database)

	if direct := firstNonEmpty(params.URI, params.ConnectionString); direct != "" {
		return direct, fallbackDatabase(database, databaseFromURI(direct))
	}
	host := strings.TrimSpace(params.Host)
	if isMongoURI(host) {
		return host, fallbackDatabase(database, databaseFromURI(host))
	}

	if host == "" {
		host = "localhost"
	}
	if database == "" {
		database = "admin"
	}

	useSRV := params.SRV != nil && *params.SRV
	scheme := "mongodb"
	if useSRV {
		scheme = "mongodb+srv"
	}

	var builder strings.Builder
	builder.WriteString(scheme)
	builder.WriteString("://")
	if params.Username != "" {
		builder.WriteString(url.QueryEscape(params.Username))
		if params.Password != "" {
			builder.WriteByte(':')
			builder.WriteString(url.QueryEscape(params.Password))
		}
		builder.WriteByte('@')
	}
	builder.WriteString(formatHosts(host, params.Port, useSRV))
	builder.WriteByte('/')
	builder.WriteString(url.PathEscape(database))

	if query := buildQuery(params); query != "" {
		builder.WriteByte('?')
		builder.WriteString(query)
	}
	return builder.String(), database
}

func formatHosts(host string, port *int, useSRV bool) string {
	hosts := strings.Split(host, ",")
	for i, entry := range hosts {
		entry = strings.TrimSpace(entry)
		if !useSRV && port != nil && !strings.Contains(entry, ":") {
			entry = entry + ":" + strconv.Itoa(*port)
		}
		hosts[i] = entry
	}
	return strings.Join(hosts, ",")
}

func buildQuery(params ConnParams) string {
	values := url.Values{}
	if params.AuthSource != "" {
		values.Set("authSource", params.AuthSource)
	}
	if params.AuthMechanism != "" {
		values.Set("authMechanism", params.AuthMechanism)
	}
	if params.ReplicaSet != "" {
		values.Set("replicaSet", params.ReplicaSet)
	}
	if tlsEnabled(params.SSLMode) {
		values.Set("tls", "true")
		if tlsInsecure(params.SSLMode) {
			values.Set("tlsInsecure", "true")
		}
	}
	encoded := values.Encode()

	if extra := strings.TrimPrefix(strings.TrimSpace(params.ExtraOptions), "?"); extra != "" {
		if encoded != "" {
			encoded += "&"
		}
		encoded += extra
	}
	return encoded
}

func applyTLS(clientOptions *options.ClientOptions, sslMode string) {
	if tlsEnabled(sslMode) && tlsInsecure(sslMode) {
		clientOptions.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})
	}
}

var tlsDisabledModes = newStringSet("", "disable", "disabled", "off", "false", "none", "no")

var tlsInsecureModes = newStringSet("require", "required", "allow", "prefer", "preferred", "insecure", "skip-verify")

func tlsEnabled(mode string) bool {
	return !tlsDisabledModes.contains(mode)
}

func tlsInsecure(mode string) bool {
	return tlsInsecureModes.contains(mode)
}

func isMongoURI(value string) bool {
	return strings.HasPrefix(value, "mongodb://") || strings.HasPrefix(value, "mongodb+srv://")
}

func databaseFromURI(uri string) string {
	parsed, err := url.Parse(uri)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(parsed.Path, "/")
}

func fallbackDatabase(primary, secondary string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	if strings.TrimSpace(secondary) != "" {
		return secondary
	}
	return "admin"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func coerceInt(value any) *int {
	switch n := value.(type) {
	case float64:
		result := int(n)
		return &result
	case json.Number:
		if parsed, err := n.Int64(); err == nil {
			result := int(parsed)
			return &result
		}
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(n)); err == nil && n != "" {
			return &parsed
		}
	}
	return nil
}

func coerceString(value any) string {
	switch s := value.(type) {
	case string:
		return s
	case bool:
		return strconv.FormatBool(s)
	case float64:
		return strconv.FormatFloat(s, 'f', -1, 64)
	case json.Number:
		return s.String()
	default:
		return ""
	}
}

func coerceBool(value any) *bool {
	switch b := value.(type) {
	case bool:
		return &b
	case string:
		switch strings.ToLower(strings.TrimSpace(b)) {
		case "true", "1", "yes", "on":
			result := true
			return &result
		case "false", "0", "no", "off":
			result := false
			return &result
		}
	}
	return nil
}
