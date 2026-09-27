package cql

import (
	"errors"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
)

// defaultHost is the host of a DSN that names none.
const defaultHost = "127.0.0.1"

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

// ParseDSN parses a DSN in one of two forms, and returns the configuration of
// a gocql cluster.
//
// A DSN that starts with cql:// or cassandra:// is a URL:
//
//	cql://user:password@host1:9042,host2/keyspace?consistency=localQuorum
//
// The user information sets the username and the password, the host part
// holds the hosts, separated by commas, and the path names the keyspace.
// net/url refuses a list of hosts that holds an IPv6 address, so the query can
// also add one host for each host key: cql://[::1]:9042/ks?host=[::2]:9042.
//
// Any other DSN is a list of hosts, separated by commas, and a query:
//
//	host1:9042,host2?keyspace=keyspace&username=user&password=password
//
// Both forms take these keys in the query:
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
// A key that appears twice is an error, and so is a setting that appears in
// two places, such as a keyspace in the path and in a keyspace key. A DSN with
// no host connects to 127.0.0.1.
func ParseDSN(dsn string) (*gocql.ClusterConfig, error) {
	if isURL(dsn) {
		return parseURL(dsn)
	}
	hosts, query, _ := strings.Cut(dsn, "?")
	cfg := gocql.NewCluster()
	for h := range strings.SplitSeq(hosts, ",") {
		if h = strings.TrimSpace(h); h != "" {
			cfg.Hosts = append(cfg.Hosts, h)
		}
	}
	values, err := url.ParseQuery(query)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidDSN, err)
	}
	if err := applyQuery(cfg, values); err != nil {
		return nil, err
	}
	if len(cfg.Hosts) == 0 {
		cfg.Hosts = []string{defaultHost}
	}
	return cfg, nil
}

// isURL reports whether dsn starts with cql:// or cassandra://, in any case.
func isURL(dsn string) bool {
	lower := strings.ToLower(dsn)
	return strings.HasPrefix(lower, "cql://") || strings.HasPrefix(lower, "cassandra://")
}

// parseURL parses the URL form of a DSN.
func parseURL(dsn string) (*gocql.ClusterConfig, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidDSN, err)
	}
	if u.Fragment != "" {
		return nil, fmt.Errorf("%w: fragment %q", ErrInvalidDSN, u.Fragment)
	}
	values := u.Query()
	if u.User != nil {
		if values.Has("username") || values.Has("password") {
			return nil, fmt.Errorf("%w: credentials in the user information and in the query", ErrInvalidDSN)
		}
		values.Set("username", u.User.Username())
		if pass, ok := u.User.Password(); ok {
			values.Set("password", pass)
		}
	}
	if keyspace := strings.TrimPrefix(u.Path, "/"); keyspace != "" {
		if strings.Contains(keyspace, "/") {
			return nil, fmt.Errorf("%w: path %q holds more than a keyspace", ErrInvalidDSN, u.Path)
		}
		if values.Has("keyspace") {
			return nil, fmt.Errorf("%w: keyspace in the path and in the query", ErrInvalidDSN)
		}
		values.Set("keyspace", keyspace)
	}
	cfg := gocql.NewCluster()
	// Hostname and Port do not split a list, so the host part is split here.
	for h := range strings.SplitSeq(u.Host, ",") {
		if h != "" {
			cfg.Hosts = append(cfg.Hosts, h)
		}
	}
	if err := applyQuery(cfg, values); err != nil {
		return nil, err
	}
	if len(cfg.Hosts) == 0 {
		cfg.Hosts = []string{defaultHost}
	}
	return cfg, nil
}

// applyQuery sets the key of each value on cfg.
func applyQuery(cfg *gocql.ClusterConfig, values url.Values) error {
	var auth *gocql.PasswordAuthenticator
	var ssl *gocql.SslOptions
	for _, key := range slices.Sorted(maps.Keys(values)) {
		vals := values[key]
		if key == "host" {
			for _, h := range vals {
				if h != "" {
					cfg.Hosts = append(cfg.Hosts, h)
				}
			}
			continue
		}
		if len(vals) != 1 {
			return fmt.Errorf("%w: key %q appears %d times", ErrInvalidDSN, key, len(vals))
		}
		value := vals[0]
		var err error
		switch key {
		case "consistency":
			c, ok := consistencies[value]
			if !ok {
				err = fmt.Errorf("unknown consistency %q", value)
			}
			cfg.Consistency = c
		case "keyspace":
			if value == "" {
				err = errors.New("empty keyspace")
			}
			cfg.Keyspace = value
		case "timeout":
			cfg.Timeout, err = parseDuration(value)
		case "connectTimeout":
			cfg.ConnectTimeout, err = parseDuration(value)
		case "writeCoalesceWaitTime":
			cfg.WriteCoalesceWaitTime, err = parseDuration(value)
		case "numConns":
			cfg.NumConns, err = strconv.Atoi(value)
			if err == nil && cfg.NumConns < 1 {
				err = fmt.Errorf("numConns %d is less than 1", cfg.NumConns)
			}
		case "ignorePeerAddr":
			cfg.IgnorePeerAddr, err = strconv.ParseBool(value)
		case "disableInitialHostLookup":
			cfg.DisableInitialHostLookup, err = strconv.ParseBool(value)
		case "username", "password":
			if auth == nil {
				auth = &gocql.PasswordAuthenticator{}
			}
			if key == "username" {
				auth.Username = value
			} else {
				auth.Password = value
			}
		case "enableHostVerification", "certPath", "keyPath", "caPath":
			if ssl == nil {
				ssl = &gocql.SslOptions{}
			}
			switch key {
			case "enableHostVerification":
				ssl.EnableHostVerification, err = strconv.ParseBool(value)
			case "certPath":
				ssl.CertPath = value
			case "keyPath":
				ssl.KeyPath = value
			case "caPath":
				ssl.CaPath = value
			}
		default:
			err = errors.New("unknown key")
		}
		if err != nil {
			return fmt.Errorf("%w: key %q: %w", ErrInvalidDSN, key, err)
		}
	}
	if auth != nil {
		cfg.Authenticator = *auth
	}
	if ssl != nil {
		cfg.SslOpts = ssl
	}
	return nil
}

// parseDuration parses a duration that must not be negative.
func parseDuration(s string) (time.Duration, error) {
	d, err := time.ParseDuration(s)
	if err == nil && d < 0 {
		err = fmt.Errorf("duration %s is negative", s)
	}
	return d, err
}

// FormatDSN writes cfg as a cql:// URL that ParseDSN reads back. It writes
// only the settings that differ from the defaults of gocql.NewCluster, and
// only the settings that the DSN can express. It returns an error for a value
// that the DSN cannot express, such as a consistency with no DSN name or an
// authenticator other than gocql.PasswordAuthenticator.
func FormatDSN(cfg *gocql.ClusterConfig) (string, error) {
	def := gocql.NewCluster()
	values := url.Values{}
	u := &url.URL{Scheme: "cql"}
	if cfg.Consistency != def.Consistency {
		name, ok := consistencyName(cfg.Consistency)
		if !ok {
			return "", fmt.Errorf("%w: consistency %s has no dsn name", ErrInvalidDSN, cfg.Consistency)
		}
		values.Set("consistency", name)
	}
	switch {
	case cfg.Keyspace == "":
	case isPlain(cfg.Keyspace):
		u.Path = "/" + cfg.Keyspace
	default:
		values.Set("keyspace", cfg.Keyspace)
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
		return "", fmt.Errorf("%w: authenticator %T has no dsn form", ErrInvalidDSN, cfg.Authenticator)
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
	// net/url refuses a list of hosts that holds an IPv6 address, and a list
	// where an earlier host has a port and the last one has none. So the
	// hosts go in the host part only when every host is plain and net/url
	// reads the list back as it is. Otherwise every host goes in a host key.
	if joined := strings.Join(cfg.Hosts, ","); allPlainHosts(cfg.Hosts) && hostPartHolds(*u, joined) {
		u.Host = joined
		return u.String(), nil
	}
	values["host"] = slices.Clone(cfg.Hosts)
	u.RawQuery = values.Encode()
	// url.URL writes no // when the host part and the path are both empty,
	// and a DSN with no // is not a URL.
	s := u.String()
	if !strings.HasPrefix(s, "cql://") {
		s = "cql://" + strings.TrimPrefix(s, "cql:")
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

// allPlainHosts reports whether every host is plain. See isPlainHost.
func allPlainHosts(hosts []string) bool {
	for _, h := range hosts {
		if !isPlainHost(h) {
			return false
		}
	}
	return true
}

// isPlainHost reports whether host is a name or an IPv4 address with an
// optional port, which the host part of a URL can hold as it is.
func isPlainHost(host string) bool {
	name, port, ok := strings.Cut(host, ":")
	if ok && (port == "" || strings.Trim(port, "0123456789") != "") {
		return false
	}
	return name != "" && strings.Trim(name, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789.-_") == ""
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
