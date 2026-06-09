package mongodb

import "testing"

func intPtr(value int) *int    { return &value }
func boolPtr(value bool) *bool { return &value }

func TestBuildURI(t *testing.T) {
	cases := []struct {
		name         string
		params       ConnParams
		wantURI      string
		wantDatabase string
	}{
		{
			name:         "host and port",
			params:       ConnParams{Host: "localhost", Port: intPtr(27017)},
			wantURI:      "mongodb://localhost:27017/admin",
			wantDatabase: "admin",
		},
		{
			name:         "credentials are percent encoded with default auth source",
			params:       ConnParams{Host: "db.internal", Port: intPtr(27017), Database: "shop", Username: "ad min", Password: "p@ss:w/rd"},
			wantURI:      "mongodb://ad+min:p%40ss%3Aw%2Frd@db.internal:27017/shop?authSource=admin",
			wantDatabase: "shop",
		},
		{
			name:         "srv omits port",
			params:       ConnParams{Host: "cluster0.mongodb.net", Port: intPtr(27017), Database: "prod", Username: "user", Password: "secret", SRV: boolPtr(true)},
			wantURI:      "mongodb+srv://user:secret@cluster0.mongodb.net/prod?authSource=admin",
			wantDatabase: "prod",
		},
		{
			name:         "explicit auth source overrides default",
			params:       ConnParams{Host: "db.internal", Port: intPtr(27017), Database: "shop", Username: "user", Password: "secret", AuthSource: "shop"},
			wantURI:      "mongodb://user:secret@db.internal:27017/shop?authSource=shop",
			wantDatabase: "shop",
		},
		{
			name:         "no credentials means no auth source",
			params:       ConnParams{Host: "db.internal", Port: intPtr(27017), Database: "shop"},
			wantURI:      "mongodb://db.internal:27017/shop",
			wantDatabase: "shop",
		},
		{
			name:         "replica set seed list with options",
			params:       ConnParams{Host: "a:27017,b:27017,c:27017", Database: "app", ReplicaSet: "rs0", AuthSource: "admin"},
			wantURI:      "mongodb://a:27017,b:27017,c:27017/app?authSource=admin&replicaSet=rs0",
			wantDatabase: "app",
		},
		{
			name:         "tls enabled and insecure",
			params:       ConnParams{Host: "secure.host", Port: intPtr(27017), Database: "app", SSLMode: "require"},
			wantURI:      "mongodb://secure.host:27017/app?tls=true&tlsInsecure=true",
			wantDatabase: "app",
		},
		{
			name:         "full connection string passthrough",
			params:       ConnParams{ConnectionString: "mongodb+srv://u:p@cluster.example.net/inventory?retryWrites=true"},
			wantURI:      "mongodb+srv://u:p@cluster.example.net/inventory?retryWrites=true",
			wantDatabase: "inventory",
		},
		{
			name:         "uri provided in host field",
			params:       ConnParams{Host: "mongodb://localhost:27017/logs"},
			wantURI:      "mongodb://localhost:27017/logs",
			wantDatabase: "logs",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotURI, gotDatabase := buildURI(tc.params)
			if gotURI != tc.wantURI {
				t.Errorf("uri = %q, want %q", gotURI, tc.wantURI)
			}
			if gotDatabase != tc.wantDatabase {
				t.Errorf("database = %q, want %q", gotDatabase, tc.wantDatabase)
			}
		})
	}
}

func TestTLSModes(t *testing.T) {
	cases := []struct {
		mode        string
		wantEnabled bool
	}{
		{"", false},
		{"disable", false},
		{"OFF", false},
		{"require", true},
		{"verify-full", true},
	}
	for _, tc := range cases {
		if got := tlsEnabled(tc.mode); got != tc.wantEnabled {
			t.Errorf("tlsEnabled(%q) = %v, want %v", tc.mode, got, tc.wantEnabled)
		}
	}
}
