package cassandra

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/xo/dbimp"
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
		{"one host", "cassandra://h1", func(f *dsnFields) { f.Hosts = []string{"h1"} }},
		{"every key", "cassandra://h1?host=h2&keyspace=ks&consistency=localQuorum&timeout=5s&connectTimeout=6s&numConns=3&username=u&password=p%26w",
			func(f *dsnFields) {
				f.Hosts = []string{"h1", "h2"}
				f.Keyspace, f.Consistency = "ks", gocql.LocalQuorum
				f.Timeout, f.ConnectTimeout, f.NumConns = 5*time.Second, 6*time.Second, 3
				f.Auth = &gocql.PasswordAuthenticator{Username: "u", Password: "p&w"}
			}},
		{"flags", "cassandra://h1?ignorePeerAddr=true&disableInitialHostLookup=true&writeCoalesceWaitTime=1ms",
			func(f *dsnFields) {
				f.Hosts = []string{"h1"}
				f.IgnorePeerAddr, f.DisableInitialHostLookup, f.WriteCoalesceWaitTime = true, true, time.Millisecond
			}},
		{"tls", "cassandra://h1?enableHostVerification=true&certPath=%2Fc&keyPath=%2Fk&caPath=%2Fa",
			func(f *dsnFields) {
				f.Hosts = []string{"h1"}
				f.SSL = &gocql.SslOptions{EnableHostVerification: true, CertPath: "/c", KeyPath: "/k", CaPath: "/a"}
			}},
		{"url", "cassandra://u:p%40ss@h1:9042/ks?consistency=one&host=h2:9043",
			func(f *dsnFields) {
				f.Hosts = []string{"h1:9042", "h2:9043"}
				f.Keyspace, f.Consistency = "ks", gocql.One
				f.Auth = &gocql.PasswordAuthenticator{Username: "u", Password: "p@ss"}
			}},
		{"scheme in upper case", "CASSANDRA://h1", func(f *dsnFields) { f.Hosts = []string{"h1"} }},
		{"url with no host", "cassandra:///ks", func(f *dsnFields) { f.Hosts, f.Keyspace = []string{"127.0.0.1"}, "ks" }},
		{"url with user only", "cassandra://u@h1", func(f *dsnFields) {
			f.Hosts = []string{"h1"}
			f.Auth = &gocql.PasswordAuthenticator{Username: "u"}
		}},
		{"url with hosts and no ports", "cassandra://h1?host=h2", func(f *dsnFields) { f.Hosts = []string{"h1", "h2"} }},
		{"hosts with a query", "cassandra://127.0.0.1:32768?username=cassandra&password=cassandra&timeout=30s&connectTimeout=30s",
			func(f *dsnFields) {
				f.Hosts = []string{"127.0.0.1:32768"}
				f.Timeout, f.ConnectTimeout = 30*time.Second, 30*time.Second
				f.Auth = &gocql.PasswordAuthenticator{Username: "cassandra", Password: "cassandra"}
			}},
		{"credentials in the user information", "cassandra://cassandra:cassandra@127.0.0.1:32768/",
			func(f *dsnFields) {
				f.Hosts = []string{"127.0.0.1:32768"}
				f.Auth = &gocql.PasswordAuthenticator{Username: "cassandra", Password: "cassandra"}
			}},
		{"url with ipv6 and host keys", "cassandra://[::1]:9042/ks?host=[::2]:9042&host=h3",
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
	tests := []struct {
		dsn  string
		want error
	}{
		{"", dbimp.ErrScheme},
		{"h1", dbimp.ErrScheme},
		{"cql://h1", dbimp.ErrScheme},
		{"http://h1", dbimp.ErrScheme},
		{"CASSANDRA:h1", dbimp.ErrInvalidValue},
		{"cassandra://h1?nope=1", dbimp.ErrUnknownKey},
		{"cassandra://h1?blah", dbimp.ErrUnknownKey},
		{"cassandra://h1?consistency=serial", dbimp.ErrInvalidValue},
		{"cassandra://h1?keyspace=", dbimp.ErrInvalidValue},
		{"cassandra://h1?timeout=soon", dbimp.ErrInvalidValue},
		{"cassandra://h1?timeout=-1s", dbimp.ErrInvalidValue},
		{"cassandra://h1?numConns=0", dbimp.ErrInvalidValue},
		{"cassandra://h1?ignorePeerAddr=maybe", dbimp.ErrInvalidValue},
		{"cassandra://h1?keyspace=a&keyspace=b", dbimp.ErrRepeatedKey},
		{"cassandra://h1?a=%zz", dbimp.ErrInvalidValue},
		// The host part holds one host (D34). net/url refuses a list that has
		// a port on the first host and none on the last, or an IPv6 address.
		{"cassandra://h1,h2", dbimp.ErrInvalidValue},
		{"cassandra://h1?host=h2,h3", dbimp.ErrInvalidValue},
		{"cassandra://h1,[::1]:9042", nil},
		{"cassandra://h1:9042,h2", nil},
		{"cassandra://u@h1?username=v", dbimp.ErrInvalidValue},
		{"cassandra://h1/ks?keyspace=ks", dbimp.ErrInvalidValue},
		{"cassandra://h1/ks/more", dbimp.ErrInvalidValue},
		{"cassandra://h1#frag", dbimp.ErrInvalidValue},
	}
	for _, tt := range tests {
		_, err := ParseDSN(tt.dsn)
		// A want of nil is an error of net/url, which wraps no sentinel.
		if err == nil || (tt.want != nil && !errors.Is(err, tt.want)) {
			t.Errorf("%q: got %v, want %v", tt.dsn, err, tt.want)
		}
	}
}

// TestParseDSNErrorsHideThePassword holds that an error never holds the DSN,
// because the DSN can hold a password.
func TestParseDSNErrorsHideThePassword(t *testing.T) {
	t.Parallel()
	for _, dsn := range []string{
		"cassandra://u:secret@h1?nope=1",
		"cassandra://u:secret@h1?timeout=soon",
		"cassandra://u:secret@h1:%zz",
		"http://u:secret@h1",
	} {
		_, err := ParseDSN(dsn)
		if err == nil || strings.Contains(err.Error(), "secret") {
			t.Errorf("%q: got %v", dsn, err)
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
	if want := "cassandra://u:p%40ss@h1:9042/ks?consistency=localQuorum&host=h2%3A9042"; got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	// The first host goes in the host part, and the others in host keys.
	got, err = FormatDSN(gocql.NewCluster("h1:9042", "h2"))
	if want := "cassandra://h1:9042?host=h2"; err != nil || got != want {
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
		if _, err := FormatDSN(cfg); !errors.Is(err, dbimp.ErrInvalidValue) {
			t.Errorf("got %v, want %v", err, dbimp.ErrInvalidValue)
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
		"cassandra://h1",
		"cassandra://h1?host=h2&keyspace=ks&consistency=all&timeout=1s&numConns=4",
		"cassandra://h1?username=u&password=p",
		"cassandra://h1?password=p",
		"cassandra://h1?caPath=",
		"cassandra://[::1]:9042?host=h2&keyspace=ks",
		"cassandra://h1:9042?host=h2",
		"cassandra://h1?host=h2:9042",
		"cassandra://h1?keyspace=Mixed-Case",
		"cassandra://u:p@h1:9042/ks?writeCoalesceWaitTime=0s&ignorePeerAddr=true",
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
		"cassandra://h1?host=h2&keyspace=ks&consistency=localQuorum",
		"cassandra://u:p@h1:9042/ks?timeout=1s&host=h2",
		"cassandra://[::1]:9042/ks?host=[::2]",
		"cassandra://h1?username=u&caPath=%2Fa",
	} {
		f.Add(dsn)
	}
	f.Fuzz(roundTrip)
}
