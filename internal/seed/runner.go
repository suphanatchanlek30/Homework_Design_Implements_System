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

// New creates a seed runner around one raw SQL database connection.
// สร้างตัวช่วย run seed โดยอิงการเชื่อมต่อฐานข้อมูลแบบ raw SQL
func New(db *sql.DB) *Runner {
	return &Runner{DB: db}
}

// RunSeed executes only the seed SQL file against the target database.
// รันเฉพาะไฟล์ seed SQL กับฐานข้อมูลเป้าหมาย
func (r *Runner) RunSeed(ctx context.Context, seedPath string) error {
	if err := r.runSQLFile(ctx, seedPath); err != nil {
		return fmt.Errorf("run seed file: %w", err)
	}

	return nil
}

// RunSchemaAndSeed executes the schema file first and then the seed file.
// รันไฟล์ schema ก่อน แล้วค่อยตามด้วยไฟล์ seed
func (r *Runner) RunSchemaAndSeed(ctx context.Context, schemaPath, seedPath string) error {
	if err := r.runSQLFile(ctx, schemaPath); err != nil {
		return fmt.Errorf("run schema file: %w", err)
	}

	if err := r.runSQLFile(ctx, seedPath); err != nil {
		return fmt.Errorf("run seed file: %w", err)
	}

	return nil
}

// runSQLFile reads one SQL file, splits it into statements, and executes them in order.
// อ่านไฟล์ SQL หนึ่งไฟล์ แยกเป็น statement แล้วยิงรันตามลำดับ
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

// splitSQLStatements removes comments and blank lines before splitting a script on semicolons.
// ตัดคอมเมนต์กับบรรทัดว่างออกก่อนแยก script เป็น statement ตาม semicolon
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
