package cassandra

import (
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/xo/dbimp"
)

// defaultHost is the host of a DSN that names none.
const defaultHost = "127.0.0.1"

// scheme is the one scheme of a DSN, and the name that the driver registers.
const scheme = Name

// consistencies maps the consistency names of the DSN to gocql. The names are
// camel case, as the DSN has always written them.
var consistencies = map[string]gocql.Consistency{
	"any":         gocql.Any,
	"one":         gocql.One,
	"two":         gocql.Two,
	"three":       gocql.Three,
	"quorum":      gocql.Quorum,
	"all":         gocql.All,
	"localQuorum": gocql.LocalQuorum,
	"eachQuorum":  gocql.EachQuorum,
	"localOne":    gocql.LocalOne,
}

// The keys of the query of a DSN.
const (
	keyConsistency              = "consistency"
	keyKeyspace                 = "keyspace"
	keyTimeout                  = "timeout"
	keyConnectTimeout           = "connectTimeout"
	keyNumConns                 = "numConns"
	keyIgnorePeerAddr           = "ignorePeerAddr"
	keyDisableInitialHostLookup = "disableInitialHostLookup"
	keyWriteCoalesceWaitTime    = "writeCoalesceWaitTime"
	keyUsername                 = "username"
	keyPassword                 = "password"
	keyEnableHostVerification   = "enableHostVerification"
	keyCertPath                 = "certPath"
	keyKeyPath                  = "keyPath"
	keyCAPath                   = "caPath"
	keyHost                     = "host"
)

// knownKeys are the keys of the query that dbimp.NewQuery reads. The key host
// can repeat, so ParseDSN takes it out first.
var knownKeys = []string{
	keyConsistency, keyKeyspace, keyTimeout, keyConnectTimeout, keyNumConns,
	keyIgnorePeerAddr, keyDisableInitialHostLookup, keyWriteCoalesceWaitTime,
	keyUsername, keyPassword, keyEnableHostVerification, keyCertPath, keyKeyPath, keyCAPath,
}

// ParseDSN parses a DSN and returns the configuration of a gocql cluster. A
// DSN is a URL with the scheme cassandra, parsed with net/url, and no other
// scheme or form is read (dbimp D27):
//
//	cassandra://user:password@host1:9042/keyspace?consistency=localQuorum&host=host2
//
// The user information sets the username and the password, the host part
// holds one host, and the path names the keyspace. Each host key adds one more
// host, in order: cassandra://h1:9042/ks?host=h2:9042&host=[::1]:9042 (D34).
//
// The query takes these keys, and each has the default of gocql.NewCluster:
//
//	consistency               any, one, two, three, quorum, all, localQuorum, eachQuorum or localOne
//	keyspace                  the keyspace
//	timeout                   the timeout of a request, such as 10s
//	connectTimeout            the timeout of a new connection, such as 10s
//	numConns                  the number of connections to each host
//	ignorePeerAddr            true or false
//	disableInitialHostLookup  true or false
//	writeCoalesceWaitTime     a duration, such as 200µs
//	username, password        the credentials
//	enableHostVerification    true or false
//	certPath, keyPath, caPath the paths of the TLS files
//	host                      one more host
//
// A key that is not in the list is an error that wraps dbimp.ErrUnknownKey. A
// key that appears twice wraps dbimp.ErrRepeatedKey, except host. A scheme
// other than cassandra wraps dbimp.ErrScheme, and any other fault wraps
// dbimp.ErrInvalidValue. An error never holds the DSN, because the DSN can hold
// a password. A DSN with no host connects to 127.0.0.1.
func ParseDSN(dsn string) (*gocql.ClusterConfig, error) {
	u, err := dbimp.ParseURL(scheme, dsn)
	if err != nil {
		return nil, err
	}
	if u.Fragment != "" {
		return nil, fmt.Errorf("parsing the dsn: the fragment %q: %w", u.Fragment, dbimp.ErrInvalidValue)
	}
	if strings.Contains(u.Host, ",") {
		return nil, fmt.Errorf("parsing the dsn: the host part holds a list of hosts, and one host goes there: %w", dbimp.ErrInvalidValue)
	}
	values, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("parsing the query of the dsn: %w: %w", dbimp.ErrInvalidValue, err)
	}
	cfg := gocql.NewCluster()
	if u.Host != "" {
		cfg.Hosts = append(cfg.Hosts, u.Host)
	}
	for _, h := range values[keyHost] {
		if strings.Contains(h, ",") {
			return nil, fmt.Errorf("parsing key %q: a host holds a comma, and each host has a key of its own: %w", keyHost, dbimp.ErrInvalidValue)
		}
		if h != "" {
			cfg.Hosts = append(cfg.Hosts, h)
		}
	}
	delete(values, keyHost)
	u.RawQuery = values.Encode()
	q, err := dbimp.NewQuery(u, knownKeys...)
	if err != nil {
		return nil, err
	}
	if err := applyUser(q, u, cfg); err != nil {
		return nil, err
	}
	if err := applyKeyspace(q, u, cfg); err != nil {
		return nil, err
	}
	if err := applySettings(q, cfg); err != nil {
		return nil, err
	}
	if err := applyTLS(q, cfg); err != nil {
		return nil, err
	}
	if len(cfg.Hosts) == 0 {
		cfg.Hosts = []string{defaultHost}
	}
	return cfg, nil
}

// applyUser sets the credentials of the user information and of the query.
func applyUser(q dbimp.Query, u *url.URL, cfg *gocql.ClusterConfig) error {
	auth := gocql.PasswordAuthenticator{
		Username: q.String(keyUsername, ""),
		Password: q.String(keyPassword, ""),
	}
	if u.User != nil {
		if q.Has(keyUsername) || q.Has(keyPassword) {
			return fmt.Errorf("parsing the dsn: credentials in the user information and in the query: %w", dbimp.ErrInvalidValue)
		}
		auth.Username = u.User.Username()
		auth.Password, _ = u.User.Password()
	}
	if u.User != nil || q.Has(keyUsername) || q.Has(keyPassword) {
		cfg.Authenticator = auth
	}
	return nil
}

// applyKeyspace sets the keyspace of the path or of the query.
func applyKeyspace(q dbimp.Query, u *url.URL, cfg *gocql.ClusterConfig) error {
	keyspace := q.String(keyKeyspace, "")
	if p := strings.TrimPrefix(u.Path, "/"); p != "" {
		switch {
		case strings.Contains(p, "/"):
			return fmt.Errorf("parsing the dsn: the path %q holds more than a keyspace: %w", u.Path, dbimp.ErrInvalidValue)
		case q.Has(keyKeyspace):
			return fmt.Errorf("parsing the dsn: keyspace in the path and in the query: %w", dbimp.ErrInvalidValue)
		}
		keyspace = p
	}
	if q.Has(keyKeyspace) && keyspace == "" {
		return fmt.Errorf("parsing key %q: empty: %w", keyKeyspace, dbimp.ErrInvalidValue)
	}
	cfg.Keyspace = keyspace
	return nil
}

// applySettings sets the keys that take the defaults of cfg.
func applySettings(q dbimp.Query, cfg *gocql.ClusterConfig) error {
	var err error
	if name := q.String(keyConsistency, ""); name != "" {
		c, ok := consistencies[name]
		if !ok {
			return fmt.Errorf("parsing key %q: %q: %w", keyConsistency, name, dbimp.ErrInvalidValue)
		}
		cfg.Consistency = c
	}
	if cfg.Timeout, err = duration(q, keyTimeout, cfg.Timeout); err != nil {
		return err
	}
	if cfg.ConnectTimeout, err = duration(q, keyConnectTimeout, cfg.ConnectTimeout); err != nil {
		return err
	}
	if cfg.WriteCoalesceWaitTime, err = duration(q, keyWriteCoalesceWaitTime, cfg.WriteCoalesceWaitTime); err != nil {
		return err
	}
	if cfg.NumConns, err = q.Int(keyNumConns, cfg.NumConns); err != nil {
		return err
	}
	if cfg.NumConns < 1 {
		return fmt.Errorf("parsing key %q: %d is less than 1: %w", keyNumConns, cfg.NumConns, dbimp.ErrInvalidValue)
	}
	if cfg.IgnorePeerAddr, err = q.Bool(keyIgnorePeerAddr, cfg.IgnorePeerAddr); err != nil {
		return err
	}
	cfg.DisableInitialHostLookup, err = q.Bool(keyDisableInitialHostLookup, cfg.DisableInitialHostLookup)
	return err
}

// applyTLS sets the TLS options. gocql turns TLS on when SslOpts is set, so
// the driver sets it only when the query holds a TLS key.
func applyTLS(q dbimp.Query, cfg *gocql.ClusterConfig) error {
	if !q.Has(keyEnableHostVerification) && !q.Has(keyCertPath) && !q.Has(keyKeyPath) && !q.Has(keyCAPath) {
		return nil
	}
	verify, err := q.Bool(keyEnableHostVerification, false)
	if err != nil {
		return err
	}
	cfg.SslOpts = &gocql.SslOptions{
		EnableHostVerification: verify,
		CertPath:               q.String(keyCertPath, ""),
		KeyPath:                q.String(keyKeyPath, ""),
		CaPath:                 q.String(keyCAPath, ""),
	}
	return nil
}

// duration reads the key as a duration that is not negative.
func duration(q dbimp.Query, key string, def time.Duration) (time.Duration, error) {
	d, err := q.Duration(key, def)
	if err == nil && d < 0 {
		err = fmt.Errorf("parsing key %q: %s is negative: %w", key, d, dbimp.ErrInvalidValue)
	}
	return d, err
}

// FormatDSN writes cfg as a cassandra:// URL that ParseDSN reads back. It writes
// only the settings that differ from the defaults of gocql.NewCluster, and
// only the settings that the DSN can express. It returns an error for a value
// that the DSN cannot express, such as a consistency with no DSN name or an
// authenticator other than gocql.PasswordAuthenticator.
func FormatDSN(cfg *gocql.ClusterConfig) (string, error) {
	def := gocql.NewCluster()
	values := url.Values{}
	u := &url.URL{Scheme: scheme}
	if cfg.Consistency != def.Consistency {
		name, ok := consistencyName(cfg.Consistency)
		if !ok {
			return "", fmt.Errorf("formatting the dsn: consistency %s has no dsn name: %w", cfg.Consistency, dbimp.ErrInvalidValue)
		}
		values.Set(keyConsistency, name)
	}
	switch {
	case cfg.Keyspace == "":
	case isPlain(cfg.Keyspace):
		u.Path = "/" + cfg.Keyspace
	default:
		values.Set(keyKeyspace, cfg.Keyspace)
	}
	setDuration(values, "timeout", cfg.Timeout, def.Timeout)
	setDuration(values, "connectTimeout", cfg.ConnectTimeout, def.ConnectTimeout)
	setDuration(values, "writeCoalesceWaitTime", cfg.WriteCoalesceWaitTime, def.WriteCoalesceWaitTime)
	if cfg.NumConns != def.NumConns {
		values.Set("numConns", strconv.Itoa(cfg.NumConns))
	}
	if cfg.IgnorePeerAddr {
		values.Set("ignorePeerAddr", "true")
	}
	if cfg.DisableInitialHostLookup {
		values.Set("disableInitialHostLookup", "true")
	}
	switch auth := cfg.Authenticator.(type) {
	case nil:
	case gocql.PasswordAuthenticator:
		if auth.Password != "" {
			u.User = url.UserPassword(auth.Username, auth.Password)
		} else {
			u.User = url.User(auth.Username)
		}
	default:
		return "", fmt.Errorf("formatting the dsn: authenticator %T has no dsn form: %w", cfg.Authenticator, dbimp.ErrInvalidValue)
	}
	if ssl := cfg.SslOpts; ssl != nil {
		// gocql turns TLS on when SslOpts is set, even with no field set. The
		// key is always written, so that TLS stays on after a round trip.
		values.Set("enableHostVerification", strconv.FormatBool(ssl.EnableHostVerification))
		setString(values, "certPath", ssl.CertPath)
		setString(values, "keyPath", ssl.KeyPath)
		setString(values, "caPath", ssl.CaPath)
	}
	u.RawQuery = values.Encode()
	// The first host goes in the host part when net/url reads it back as it
	// is, and every other host goes in a host key, in order (D34).
	rest := cfg.Hosts
	if len(rest) > 0 && hostPartHolds(*u, rest[0]) {
		u.Host, rest = rest[0], rest[1:]
	}
	if len(rest) > 0 {
		values[keyHost] = slices.Clone(rest)
	}
	u.RawQuery = values.Encode()
	// url.URL writes no // when the host part and the path are both empty,
	// and a DSN with no // is not a URL.
	s := u.String()
	if !strings.HasPrefix(s, scheme+"://") {
		s = scheme + "://" + strings.TrimPrefix(s, scheme+":")
	}
	return s, nil
}

// consistencyName returns the DSN name of c.
func consistencyName(c gocql.Consistency) (string, bool) {
	for name, v := range consistencies {
		if v == c {
			return name, true
		}
	}
	return "", false
}

// hostPartHolds reports whether net/url reads u back with hosts as its host
// part.
func hostPartHolds(u url.URL, hosts string) bool {
	u.Host = hosts
	parsed, err := url.Parse(u.String())
	return err == nil && parsed.Host == hosts
}

// isPlain reports whether s holds only letters, digits and underscores, which
// is what an unquoted CQL name holds.
func isPlain(s string) bool {
	return strings.Trim(s, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_") == ""
}

func setDuration(values url.Values, key string, d, def time.Duration) {
	if d != def {
		values.Set(key, d.String())
	}
}

func setString(values url.Values, key, s string) {
	if s != "" {
		values.Set(key, s)
	}
}
