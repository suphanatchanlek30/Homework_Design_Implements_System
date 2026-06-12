package config

import (
	"fmt"
	"os"
)

type Config struct {
	AppPort          string
	DBHost           string
	DBPort           string
	DBName           string
	DBUser           string
	DBPassword       string
	DBRootPassword   string
	MySQLHostPort    string
	SchemaPath       string
	SeedPath         string
	TZ               string
}

// Load reads application settings from environment variables with sensible local defaults.
// โหลดค่าตั้งต้นของระบบจาก environment variables พร้อม fallback สำหรับ local development
func Load() Config {
	return Config{
		AppPort:        getEnv("APP_PORT", "3000"),
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "3307"),
		DBName:         getEnv("DB_NAME", "promotion_engine"),
		DBUser:         getEnv("DB_USER", "promotion"),
		DBPassword:     getEnv("DB_PASSWORD", "promotion123"),
		DBRootPassword: getEnv("MYSQL_ROOT_PASSWORD", "rootpassword"),
		MySQLHostPort:  getEnv("MYSQL_HOST_PORT", "3307"),
		SchemaPath:     getEnv("SCHEMA_PATH", "database/schema.sql"),
		SeedPath:       getEnv("SEED_PATH", "database/seed.sql"),
		TZ:             getEnv("TZ", "Asia/Bangkok"),
	}
}

// AppAddress returns the listening address for the HTTP server.
// คืนค่า address ที่ HTTP server จะใช้ในการ listen
func (c Config) AppAddress() string {
	return fmt.Sprintf(":%s", c.AppPort)
}

// MySQLDSN builds the DSN string used by both GORM and raw MySQL connections.
// สร้าง DSN สำหรับใช้กับทั้ง GORM และการเชื่อมต่อ MySQL แบบตรง
func (c Config) MySQLDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&multiStatements=true&charset=utf8mb4,utf8&collation=utf8mb4_unicode_ci", c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}

// getEnv returns an environment variable or a fallback when the variable is empty.
// อ่านค่า environment variable และใช้ fallback เมื่อไม่มีการตั้งค่าไว้
func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
