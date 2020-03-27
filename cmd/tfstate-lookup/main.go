package main

import (
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
	)
	flag.StringVar(&stateFile, "state", "terraform.tfstate", "tfstate file path")
	flag.StringVar(&stateFile, "s", "terraform.tfstate", "tfstate file path")
	flag.Parse()
	if len(flag.Args()) == 0 {
		flag.Usage()
		return nil
	}

	f, err := os.Open(stateFile)
	if err != nil {
		return err
	}
	defer f.Close()

	s, err := tfstate.Read(f)
	if err != nil {
		return err
	}
	res, err := s.Lookup(flag.Arg(0))
	if err != nil {
		return err
	}
	fmt.Println(res.String())
	return nil
}
