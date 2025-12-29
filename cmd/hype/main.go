package main

import (
	"log"

	"github.com/hymatrix/hype/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		log.Fatalf("error: %v", err)
	}
}
