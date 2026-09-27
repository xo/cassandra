package cql

import (
	"errors"
	"reflect"
	"testing"
	"time"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
)

// dsnFields holds every setting that a DSN can express.
type dsnFields struct {
	Hosts                    []string
	Keyspace                 string
	Consistency              gocql.Consistency
	Timeout                  time.Duration
	ConnectTimeout           time.Duration
	WriteCoalesceWaitTime    time.Duration
	NumConns                 int
	IgnorePeerAddr           bool
	DisableInitialHostLookup bool
	Auth                     *gocql.PasswordAuthenticator
	SSL                      *gocql.SslOptions
}

func fieldsOf(cfg *gocql.ClusterConfig) dsnFields {
	f := dsnFields{
		Hosts:                    cfg.Hosts,
		Keyspace:                 cfg.Keyspace,
		Consistency:              cfg.Consistency,
		Timeout:                  cfg.Timeout,
		ConnectTimeout:           cfg.ConnectTimeout,
		WriteCoalesceWaitTime:    cfg.WriteCoalesceWaitTime,
		NumConns:                 cfg.NumConns,
		IgnorePeerAddr:           cfg.IgnorePeerAddr,
		DisableInitialHostLookup: cfg.DisableInitialHostLookup,
		SSL:                      cfg.SslOpts,
	}
	if auth, ok := cfg.Authenticator.(gocql.PasswordAuthenticator); ok {
		f.Auth = &auth
	}
	return f
}

// defaults returns the fields of a DSN that sets nothing but hosts.
func defaults(hosts ...string) dsnFields {
	f := fieldsOf(gocql.NewCluster())
	f.Hosts = hosts
	return f
}

func TestParseDSN(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		dsn  string
		want func(*dsnFields)
	}{
		{"empty", "", func(f *dsnFields) { f.Hosts = []string{"127.0.0.1"} }},
		{"one host", "h1", func(f *dsnFields) { f.Hosts = []string{"h1"} }},
		{"hosts with spaces", " h1:9042 , h2 ", func(f *dsnFields) { f.Hosts = []string{"h1:9042", "h2"} }},
		{"old form", "h1,h2?keyspace=ks&consistency=localQuorum&timeout=5s&connectTimeout=6s&numConns=3&username=u&password=p%26w",
			func(f *dsnFields) {
				f.Hosts = []string{"h1", "h2"}
				f.Keyspace, f.Consistency = "ks", gocql.LocalQuorum
				f.Timeout, f.ConnectTimeout, f.NumConns = 5*time.Second, 6*time.Second, 3
				f.Auth = &gocql.PasswordAuthenticator{Username: "u", Password: "p&w"}
			}},
		{"old form flags", "h1?ignorePeerAddr=true&disableInitialHostLookup=true&writeCoalesceWaitTime=1ms",
			func(f *dsnFields) {
				f.Hosts = []string{"h1"}
				f.IgnorePeerAddr, f.DisableInitialHostLookup, f.WriteCoalesceWaitTime = true, true, time.Millisecond
			}},
		{"old form tls", "h1?enableHostVerification=true&certPath=%2Fc&keyPath=%2Fk&caPath=%2Fa",
			func(f *dsnFields) {
				f.Hosts = []string{"h1"}
				f.SSL = &gocql.SslOptions{EnableHostVerification: true, CertPath: "/c", KeyPath: "/k", CaPath: "/a"}
			}},
		{"url", "cql://u:p%40ss@h1:9042,h2:9043/ks?consistency=one",
			func(f *dsnFields) {
				f.Hosts = []string{"h1:9042", "h2:9043"}
				f.Keyspace, f.Consistency = "ks", gocql.One
				f.Auth = &gocql.PasswordAuthenticator{Username: "u", Password: "p@ss"}
			}},
		{"cassandra scheme in upper case", "CASSANDRA://h1", func(f *dsnFields) { f.Hosts = []string{"h1"} }},
		{"url with no host", "cql:///ks", func(f *dsnFields) { f.Hosts, f.Keyspace = []string{"127.0.0.1"}, "ks" }},
		{"url with user only", "cql://u@h1", func(f *dsnFields) {
			f.Hosts = []string{"h1"}
			f.Auth = &gocql.PasswordAuthenticator{Username: "u"}
		}},
		{"url with hosts and no ports", "cql://h1,h2", func(f *dsnFields) { f.Hosts = []string{"h1", "h2"} }},
		// The two forms that dbrun in dbmeta prints (D20).
		{"dbrun dsn", "127.0.0.1:32768?username=cassandra&password=cassandra&timeout=30s&connectTimeout=30s",
			func(f *dsnFields) {
				f.Hosts = []string{"127.0.0.1:32768"}
				f.Timeout, f.ConnectTimeout = 30*time.Second, 30*time.Second
				f.Auth = &gocql.PasswordAuthenticator{Username: "cassandra", Password: "cassandra"}
			}},
		{"dbrun url", "cassandra://cassandra:cassandra@127.0.0.1:32768/",
			func(f *dsnFields) {
				f.Hosts = []string{"127.0.0.1:32768"}
				f.Auth = &gocql.PasswordAuthenticator{Username: "cassandra", Password: "cassandra"}
			}},
		{"url with ipv6 and host keys", "cql://[::1]:9042/ks?host=[::2]:9042&host=h3",
			func(f *dsnFields) { f.Hosts, f.Keyspace = []string{"[::1]:9042", "[::2]:9042", "h3"}, "ks" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := ParseDSN(test.dsn)
			if err != nil {
				t.Fatal(err)
			}
			want := defaults()
			test.want(&want)
			if got := fieldsOf(cfg); !reflect.DeepEqual(got, want) {
				t.Errorf("got  %+v\nwant %+v", got, want)
			}
		})
	}
}

func TestParseDSNErrors(t *testing.T) {
	t.Parallel()
	for _, dsn := range []string{
		"h1?nope=1",
		"h1?blah",
		"h1?consistency=serial",
		"h1?keyspace=",
		"h1?timeout=soon",
		"h1?timeout=-1s",
		"h1?numConns=0",
		"h1?ignorePeerAddr=maybe",
		"h1?keyspace=a&keyspace=b",
		"h1?a=%zz",
		"cql://h1,[::1]:9042",
		// net/url reads the text after the last colon of the host part as
		// the port, so a list whose last host has no port is refused when an
		// earlier host has one.
		"cql://h1:9042,h2",
		"cql://u@h1?username=v",
		"cql://h1/ks?keyspace=ks",
		"cql://h1/ks/more",
		"cql://h1#frag",
	} {
		if _, err := ParseDSN(dsn); !errors.Is(err, ErrInvalidDSN) {
			t.Errorf("%q: got %v, want %v", dsn, err, ErrInvalidDSN)
		}
	}
}

func TestFormatDSN(t *testing.T) {
	t.Parallel()
	cfg := gocql.NewCluster("h1:9042", "h2:9042")
	cfg.Keyspace = "ks"
	cfg.Consistency = gocql.LocalQuorum
	cfg.Authenticator = gocql.PasswordAuthenticator{Username: "u", Password: "p@ss"}
	got, err := FormatDSN(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if want := "cql://u:p%40ss@h1:9042,h2:9042/ks?consistency=localQuorum"; got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	// A list that net/url refuses goes in host keys.
	got, err = FormatDSN(gocql.NewCluster("h1:9042", "h2"))
	if want := "cql://?host=h1%3A9042&host=h2"; err != nil || got != want {
		t.Errorf("got  %s %v\nwant %s", got, err, want)
	}
	for _, cfg := range []*gocql.ClusterConfig{
		func() *gocql.ClusterConfig { c := gocql.NewCluster(); c.Consistency = gocql.Serial; return c }(),
		func() *gocql.ClusterConfig {
			c := gocql.NewCluster()
			c.Authenticator = otherAuth{}
			return c
		}(),
	} {
		if _, err := FormatDSN(cfg); !errors.Is(err, ErrInvalidDSN) {
			t.Errorf("got %v, want %v", err, ErrInvalidDSN)
		}
	}
}

// otherAuth is an authenticator that the DSN cannot express.
type otherAuth struct{}

func (otherAuth) Challenge([]byte) ([]byte, gocql.Authenticator, error) { return nil, nil, nil }
func (otherAuth) Success([]byte) error                                  { return nil }

func TestFormatDSNRoundTrip(t *testing.T) {
	t.Parallel()
	for _, dsn := range []string{
		"h1",
		"h1,h2?keyspace=ks&consistency=all&timeout=1s&numConns=4",
		"h1?username=u&password=p",
		"h1?password=p",
		"h1?caPath=",
		"[::1]:9042,h2?keyspace=ks",
		"h1:9042,h2",
		"h1,h2:9042",
		"h1?keyspace=Mixed-Case",
		"cql://u:p@h1:9042/ks?writeCoalesceWaitTime=0s&ignorePeerAddr=true",
	} {
		roundTrip(t, dsn)
	}
}

// roundTrip parses dsn, formats the result, and parses that again. Both
// parses must give the same settings.
func roundTrip(t *testing.T, dsn string) {
	t.Helper()
	cfg, err := ParseDSN(dsn)
	if err != nil {
		return
	}
	s, err := FormatDSN(cfg)
	if err != nil {
		t.Fatalf("%q: formatting: %v", dsn, err)
	}
	again, err := ParseDSN(s)
	if err != nil {
		t.Fatalf("%q: parsing %q: %v", dsn, s, err)
	}
	if a, b := fieldsOf(cfg), fieldsOf(again); !reflect.DeepEqual(a, b) {
		t.Errorf("%q gave %q\nfirst  %+v\nsecond %+v", dsn, s, a, b)
	}
}

func FuzzParseDSN(f *testing.F) {
	for _, dsn := range []string{
		"h1,h2?keyspace=ks&consistency=localQuorum",
		"cql://u:p@h1:9042,h2/ks?timeout=1s",
		"cql://[::1]:9042/ks?host=[::2]",
		"h1?username=u&caPath=%2Fa",
	} {
		f.Add(dsn)
	}
	f.Fuzz(roundTrip)
}
