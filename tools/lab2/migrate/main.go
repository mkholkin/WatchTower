package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func main() {
	if os.Getenv("LAB2_TEST_ENV") != "1" {
		log.Fatal("only runs in the isolated Lab 2 environment")
	}
	db, err := sql.Open("pgx", os.Getenv("LAB2_DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err = goose.SetDialect("postgres"); err != nil {
		log.Fatal(err)
	}
	if err = goose.Up(db, "migrations"); err != nil {
		log.Fatal(err)
	}
}
