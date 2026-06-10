package seed

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
)

type Runner struct {
	DB *sql.DB
}

func New(db *sql.DB) *Runner {
	return &Runner{DB: db}
}

func (r *Runner) RunSeed(ctx context.Context, seedPath string) error {
	if err := r.runSQLFile(ctx, seedPath); err != nil {
		return fmt.Errorf("run seed file: %w", err)
	}

	return nil
}

func (r *Runner) RunSchemaAndSeed(ctx context.Context, schemaPath, seedPath string) error {
	if err := r.runSQLFile(ctx, schemaPath); err != nil {
		return fmt.Errorf("run schema file: %w", err)
	}

	if err := r.runSQLFile(ctx, seedPath); err != nil {
		return fmt.Errorf("run seed file: %w", err)
	}

	return nil
}

func (r *Runner) runSQLFile(ctx context.Context, filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read %s: %w", filePath, err)
	}

	statements := splitSQLStatements(string(content))
	for _, statement := range statements {
		if _, err := r.DB.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("exec %s: %w", filePath, err)
		}
	}

	return nil
}

func splitSQLStatements(script string) []string {
	lines := strings.Split(script, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		filtered = append(filtered, line)
	}

	joined := strings.Join(filtered, "\n")
	parts := strings.Split(joined, ";")
	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		statement := strings.TrimSpace(part)
		if statement == "" {
			continue
		}
		statements = append(statements, statement)
	}

	return statements
}