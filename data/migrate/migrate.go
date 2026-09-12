// Package migrate applies ordered, forward-only PostgreSQL migrations.
package migrate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

var filenamePattern = regexp.MustCompile(`^([0-9]+)_[a-z0-9][a-z0-9_-]*\.up\.sql$`)

// Migration is one immutable, forward-only schema change.
type Migration struct {
	Version             int64
	Name, SQL, Checksum string
}

// Load reads and validates migrations from a filesystem directory.
func Load(source fs.FS, directory string) ([]Migration, error) {
	entries, err := fs.ReadDir(source, directory)
	if err != nil {
		return nil, fmt.Errorf("read migrations: %w", err)
	}
	migrations := make([]Migration, 0, len(entries))
	seen := make(map[int64]struct{})
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := filenamePattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}
		version, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil || version <= 0 {
			return nil, fmt.Errorf("invalid migration version in %q", entry.Name())
		}
		if _, exists := seen[version]; exists {
			return nil, fmt.Errorf("duplicate migration version %d", version)
		}
		contents, err := fs.ReadFile(source, path.Join(directory, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read migration %q: %w", entry.Name(), err)
		}
		sum := sha256.Sum256(contents)
		migrations = append(migrations, Migration{Version: version, Name: entry.Name(), SQL: string(contents), Checksum: hex.EncodeToString(sum[:])})
		seen[version] = struct{}{}
	}
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].Version < migrations[j].Version })
	return migrations, nil
}

// Apply runs pending migrations transactionally under a PostgreSQL advisory lock.
func Apply(ctx context.Context, pool *pgxpool.Pool, migrations []Migration) error {
	if pool == nil {
		return errors.New("PostgreSQL pool is required")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migrations: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(834623001)`); err != nil {
		return fmt.Errorf("lock migrations: %w", err)
	}
	if _, err := tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS swf_schema_migrations (version bigint PRIMARY KEY, name text NOT NULL, checksum text NOT NULL, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}
	rows, err := tx.Query(ctx, `SELECT version, checksum FROM swf_schema_migrations`)
	if err != nil {
		return fmt.Errorf("read migration state: %w", err)
	}
	applied := make(map[int64]string)
	for rows.Next() {
		var version int64
		var checksum string
		if err := rows.Scan(&version, &checksum); err != nil {
			rows.Close()
			return fmt.Errorf("scan migration state: %w", err)
		}
		applied[version] = checksum
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("read migration state: %w", err)
	}
	rows.Close()
	for _, migration := range migrations {
		if checksum, exists := applied[migration.Version]; exists {
			if checksum != migration.Checksum {
				return fmt.Errorf("migration %d checksum changed", migration.Version)
			}
			continue
		}
		if _, err := tx.Exec(ctx, migration.SQL); err != nil {
			return fmt.Errorf("apply migration %d: %w", migration.Version, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO swf_schema_migrations (version, name, checksum) VALUES ($1, $2, $3)`, migration.Version, migration.Name, migration.Checksum); err != nil {
			return fmt.Errorf("record migration %d: %w", migration.Version, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migrations: %w", err)
	}
	return nil
}
