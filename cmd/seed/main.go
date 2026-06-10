package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/joho/godotenv"
	appconfig "github.com/suphanatchanlek30/homework_design_implements_system/internal/config"
	appdatabase "github.com/suphanatchanlek30/homework_design_implements_system/internal/database"
	appseed "github.com/suphanatchanlek30/homework_design_implements_system/internal/seed"
)

func main() {
	_ = godotenv.Load()

	var schemaPath string
	var seedPath string

	flag.StringVar(&schemaPath, "schema", "", "optional path to schema.sql for fresh database initialization")
	flag.StringVar(&seedPath, "seed", "", "path to seed.sql")
	flag.Parse()

	config := appconfig.Load()
	if seedPath == "" {
		seedPath = config.SeedPath
	}

	db, err := appdatabase.OpenMySQL(config.MySQLDSN())
	if err != nil {
		log.Fatalf("connect mysql: %v", err)
	}
	defer db.Close()

	runner := appseed.New(db)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if schemaPath != "" {
		if err := runner.RunSchemaAndSeed(ctx, schemaPath, seedPath); err != nil {
			log.Fatalf("seed database: %v", err)
		}
	} else {
		if err := runner.RunSeed(ctx, seedPath); err != nil {
			log.Fatalf("seed database: %v", err)
		}
	}

	fmt.Println("seed completed successfully")
}