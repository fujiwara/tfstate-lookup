package main

import (
	"encoding/json"
	"flag"
	"fmt"
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
	if err := _main(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func _main() error {
	var (
		stateFile string
		raw       bool
	)
	flag.StringVar(&stateFile, "state", "terraform.tfstate", "tfstate file path")
	flag.StringVar(&stateFile, "s", "terraform.tfstate", "tfstate file path")
	flag.BoolVar(&raw, "raw", false, "raw output")
	flag.BoolVar(&raw, "r", false, "raw output")
	flag.Parse()
	if len(flag.Args()) == 0 {
		flag.Usage()
		return nil
	}

	f, err := os.Open(stateFile)
	if err != nil {
		return err
	}

	s, err := tfstate.Read(f)
	if err != nil {
		return err
	}
	res, err := s.Lookup(flag.Arg(0))
	if err != nil {
		return err
	}
	if raw {
		switch res.(type) {
		case string, float64:
			fmt.Fprintln(os.Stdout, res)
		default:
			json.NewEncoder(os.Stdout).Encode(res)
		}
	} else {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(res)
	}
	return nil
}
