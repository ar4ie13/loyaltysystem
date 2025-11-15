package main

import (
	"fmt"
	"log"

	"github.com/ar4ie13/loyaltysystem/internal/config"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := config.NewConfig()
	fmt.Println(cfg)
	return nil
}
