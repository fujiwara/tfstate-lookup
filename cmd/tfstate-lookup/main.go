package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/fujiwara/tfstate-lookup/tfstate"
	"github.com/hashicorp/logutils"
)

func init() {
	level := os.Getenv("LOG_LEVEL")
	if level == "" {
		level = "warn"
	}
	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"debug", "warn", "error"},
		MinLevel: logutils.LogLevel(level),
		Writer:   os.Stderr,
	}
	log.SetOutput(filter)
}

func main() {
	var stateFile string
	flag.StringVar(&stateFile, "state", "terraform.tfstate", "tfstate file path")
	flag.Parse()

	f, err := os.Open(stateFile)
	if err != nil {
		panic(err)
	}

	s, err := tfstate.Read(f)
	if err != nil {
		panic(err)
	}
	res, err := s.Lookup(os.Args[1])
	if err != nil {
		panic(err)
	}
	json.NewEncoder(os.Stdout).Encode(res)
}
