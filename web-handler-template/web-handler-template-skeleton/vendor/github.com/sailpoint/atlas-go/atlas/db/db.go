// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/sailpoint/atlas-go/atlas/config"
	"github.com/sailpoint/atlas-go/atlas/log"
)

// Config is a type that holds the database connection configuration.
type Config struct {
	Host     string
	User     string
	Password string
	Database string
}

// NewConfig reads configuration values from the specified config source.
func NewConfig(cfg config.Source) Config {
	c := Config{}
	c.Host = config.GetString(cfg, "ATLAS_DB_HOST", "localhost")
	c.Database = config.GetString(cfg, "ATLAS_DB_NAME", "postgres")
	c.User = config.GetString(cfg, "ATLAS_DB_USER", "postgres")
	c.Password = config.GetString(cfg, "ATLAS_DB_PASSWORD", "2thecloud")

	return c
}

// Migrate performs a database migration on the specified PostgreSQL database.
func Migrate(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}

// Connect establishes a connection to a PostgreSQL database.
func Connect(config Config) (*sql.DB, error) {
	escapedUser := url.PathEscape(config.User)
	escapedPassword := url.PathEscape(config.Password)

	url := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", escapedUser, escapedPassword, config.Host, config.Database)

	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// RollbackUnlessCommitted will attempt to rollback the current transaction, unless it has already been committed
func RollbackUnlessCommitted(ctx context.Context, tx *sql.Tx) {
	if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
		log.Errorf(ctx, "unable to rollback transaction: %v", err)
	}
}
