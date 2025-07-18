package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type Storage struct {
	DB *sql.DB
}

func (s *Storage) InitSchema(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS dependencies (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		system TEXT NOT NULL,
		name TEXT NOT NULL,
		version TEXT NOT NULL,
		relation TEXT,
		source_repo TEXT,
		openssf_score REAL,
		UNIQUE(system, name, version)
	);`
	_, err := s.DB.ExecContext(ctx, query)
	return err
}

const upsertDependencyQuery = `
  INSERT INTO dependencies (system, name, version, relation, source_repo, openssf_score)
  VALUES (?, ?, ?, ?, ?, ?)
  ON CONFLICT(system, name, version)
  DO UPDATE SET
    relation = excluded.relation,
    source_repo = excluded.source_repo,
    openssf_score = excluded.openssf_score;
`

func (s *Storage) UpsertDependencies(ctx context.Context, deps []Dependency) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, upsertDependencyQuery)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, dep := range deps {
		if _, err := stmt.ExecContext(ctx,
			dep.System,
			dep.Name,
			dep.Version,
			dep.Relation,
			dep.SourceRepo,
			dep.OpenSSFScore,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Storage) UpsertDependency(ctx context.Context, dep Dependency) error {
	_, err := s.DB.ExecContext(ctx, upsertDependencyQuery,
		dep.System,
		dep.Name,
		dep.Version,
		dep.Relation,
		dep.SourceRepo,
		dep.OpenSSFScore,
	)

	return err
}

func (s *Storage) GetDependency(ctx context.Context, system, name, version string) (Dependency, error) {
	var d Dependency
	err := s.DB.QueryRowContext(ctx,
		`SELECT system, name, version, relation, source_repo, openssf_score
	 FROM dependencies WHERE system=? AND name=? AND version=?`,
		system, name, version,
	).Scan(&d.System, &d.Name, &d.Version, &d.Relation, &d.SourceRepo, &d.OpenSSFScore)

	return d, err
}

func (s *Storage) ListDependenciesFiltered(ctx context.Context, name string, minScore *float64) ([]Dependency, error) {
	query := `
		SELECT system, name, version, relation, source_repo, openssf_score
		FROM dependencies
		WHERE 1=1
	`
	var args []any

	if name != "" {
		query += " AND name LIKE ?"
		args = append(args, "%"+name+"%")
	}

	if minScore != nil {
		query += " AND openssf_score >= ?"
		args = append(args, *minScore)
	}

	query += " ORDER BY system, name, version"

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Dependency
	for rows.Next() {
		var d Dependency
		if err := rows.Scan(&d.System, &d.Name, &d.Version, &d.Relation, &d.SourceRepo, &d.OpenSSFScore); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

func (s *Storage) DeleteDependency(ctx context.Context, system, name, version string) error {
	_, err := s.DB.ExecContext(ctx,
		`DELETE FROM dependencies WHERE system=? AND name=? AND version=?`,
		system, name, version)
	return err
}

func (s *Storage) GetDependenciesMap(ctx context.Context, deps []Dependency) (map[string]Dependency, error) {
	if len(deps) == 0 {
		return map[string]Dependency{}, nil
	}

	var (
		args       []any
		conditions []string
	)
	for _, dep := range deps {
		conditions = append(conditions, "(system = ? AND name = ? AND version = ?)")
		args = append(args, dep.System, dep.Name, dep.Version)
	}

	query := fmt.Sprintf(`
		SELECT system, name, version, source_repo, openssf_score, relation
		FROM dependencies
		WHERE %s;
	`, strings.Join(conditions, " OR "))

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]Dependency)
	for rows.Next() {
		var dep Dependency
		var score sql.NullFloat64
		if err := rows.Scan(&dep.System, &dep.Name, &dep.Version, &dep.SourceRepo, &score, &dep.Relation); err != nil {
			return nil, err
		}
		if score.Valid {
			dep.OpenSSFScore = &score.Float64
		}
		key := fmt.Sprintf("%s|%s|%s", dep.System, dep.Name, dep.Version)
		result[key] = dep
	}

	return result, nil
}
