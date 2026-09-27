package cql_test

import (
	"context"
	"database/sql"
	"log"
	"time"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/xo/cql"
)

// The examples need a server, so none of them checks its output.

func Example() {
	db, err := sql.Open("cql", "cql://cassandra:cassandra@127.0.0.1:9042/app?consistency=localQuorum")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	rows, err := db.QueryContext(ctx, "SELECT id, name, tags FROM users WHERE org = ?", "xo")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			id   gocql.UUID
			name sql.Null[string]
			tags []string
		)
		if err := rows.Scan(&id, &name, &tags); err != nil {
			log.Fatal(err)
		}
		log.Println(id, name.V, tags)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
}

func ExampleNewConnector() {
	cfg := gocql.NewCluster("10.0.0.1", "10.0.0.2")
	cfg.Keyspace = "app"
	cfg.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(gocql.RoundRobinHostPolicy())
	db := sql.OpenDB(cql.NewConnector(cfg))
	defer db.Close()
	if err := db.PingContext(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func ExampleWithOptions() {
	db, err := sql.Open("cql", "cql://127.0.0.1/app")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	// Every statement that runs with ctx reads at LOCAL_ONE.
	ctx := cql.WithOptions(context.Background(), cql.Consistency(gocql.LocalOne))
	// An argument overrides the context for one statement.
	_, err = db.ExecContext(ctx, "UPDATE users SET name = ? WHERE id = ?", "ken", 1,
		cql.Consistency(gocql.Quorum), cql.Timestamp(time.Now()))
	if err != nil {
		log.Fatal(err)
	}
}

func Example_batch() {
	db, err := sql.Open("cql", "cql://127.0.0.1/app")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	// A batch is one CQL statement. A logged batch is atomic, but it is not
	// isolated and it cannot roll back.
	_, err = db.ExecContext(context.Background(), `BEGIN BATCH
	INSERT INTO users (id, name) VALUES (?, ?);
	INSERT INTO users_by_name (name, id) VALUES (?, ?);
APPLY BATCH`, 1, "ken", "ken", 1)
	if err != nil {
		log.Fatal(err)
	}
}

func Example_lightweightTransaction() {
	db, err := sql.Open("cql", "cql://127.0.0.1/app")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	// The first column of the result is [applied]. Run the statement as a
	// query to read it. When it is false, the row also holds the values that
	// are there already, so the number of columns changes.
	rows, err := db.QueryContext(context.Background(),
		"INSERT INTO users (id, name) VALUES (?, ?) IF NOT EXISTS", 1, "ken")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		log.Fatal(err)
	}
	var applied bool
	dest := make([]any, len(cols))
	dest[0] = &applied
	for i := 1; i < len(dest); i++ {
		dest[i] = new(any)
	}
	for rows.Next() {
		if err := rows.Scan(dest...); err != nil {
			log.Fatal(err)
		}
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	log.Println("applied:", applied)
}
