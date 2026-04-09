package testutil

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ory/dockertest/v3"
)

var lookupHost = net.LookupHost

type PostgresEnv struct {
	pool        *dockertest.Pool
	resource    *dockertest.Resource
	DB          *pgxpool.Pool
	DatabaseURL string
}

func StartPostgres(ctx context.Context) (*PostgresEnv, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, err
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "16-alpine",
		Env: []string{
			"POSTGRES_USER=gateway",
			"POSTGRES_PASSWORD=gateway",
			"POSTGRES_DB=gateway",
		},
	})
	if err != nil {
		return nil, err
	}

	databaseURL := fmt.Sprintf(
		"postgres://gateway:gateway@%s:%s/gateway?sslmode=disable",
		dockerHost(),
		resource.GetPort("5432/tcp"),
	)

	var db *pgxpool.Pool
	if err := pool.Retry(func() error {
		db, err = pgxpool.New(ctx, databaseURL)
		if err != nil {
			return err
		}

		return db.Ping(ctx)
	}); err != nil {
		if db != nil {
			db.Close()
		}
		_ = pool.Purge(resource)
		return nil, err
	}

	return &PostgresEnv{
		pool:        pool,
		resource:    resource,
		DB:          db,
		DatabaseURL: databaseURL,
	}, nil
}

func (e *PostgresEnv) Close() error {
	if e.DB != nil {
		e.DB.Close()
	}
	if e.pool != nil && e.resource != nil {
		return e.pool.Purge(e.resource)
	}
	return nil
}

func dockerHost() string {
	if host := os.Getenv("DOCKER_HOST_NAME"); host != "" {
		return host
	}

	if _, err := lookupHost("host.docker.internal"); err == nil {
		return "host.docker.internal"
	}

	return "127.0.0.1"
}
