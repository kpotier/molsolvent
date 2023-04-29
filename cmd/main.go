package main

import (
	"fmt"
	"log"
	"os"

	"github.com/kpotier/molsolvent/pkg/cfg"
)

func main() {
	log := log.New(os.Stdout, "", log.LstdFlags)

	if len(os.Args) != 2 {
		log.Fatal("one argument is needed: path of the configuration file")
	}

	c, err := cfg.New(os.Args[1])
	if err != nil {
		log.Fatal(fmt.Errorf("New: %w", err))
	}

	c.Start(log)
}
