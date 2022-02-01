package main

import (
	"log"
	"os"

	"github.com/tkcrm/pgxgen/internal/pgxgen"
)

func main() {
	args := os.Args[1:]
	if err := pgxgen.Start(args); err != nil {
		log.Fatalf("error: %v", err)
	}
}
