package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/fujiwara/tfstate-lookup/tfstate"
)

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

	s, err := tfstate.ReadFile(stateFile)
	if err != nil {
		return err
	}
	if len(flag.Args()) == 0 {
		names, err := s.List()
		if err != nil {
			return err
		}
		fmt.Println(strings.Join(names, "\n"))
	} else {
		res, err := s.Lookup(flag.Arg(0))
		if err != nil {
			return err
		}
		fmt.Println(res.String())
	}
	return nil
}
