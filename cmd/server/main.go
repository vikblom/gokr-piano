package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	// Register dbmate driver.
	_ "github.com/amacneil/dbmate/pkg/driver/postgres"
	"github.com/jackc/pgx/v5"

	piano "github.com/vikblom/gokr-piano"
)

func hello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "HELLO\n")
}

func runMain() error {
	u, err := url.Parse(os.Getenv("DATABASE_URL"))
	if err != nil {
		return fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	err = piano.MigrateUp(u)
	if err != nil {
		log.Fatal(err)
	}

	// urlExample := "postgres://username:password@localhost:5432/database_name"
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer conn.Close(context.Background())

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return http.ListenAndServe(":"+port, http.HandlerFunc(hello))
}

func main() {
	err := runMain()
	if err != nil {
		log.Fatal(err)
	}
}
