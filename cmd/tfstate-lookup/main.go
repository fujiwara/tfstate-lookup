package main

import (
	"flag"
	"fmt"
	"os"

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
	if len(flag.Args()) == 0 {
		flag.Usage()
		return nil
	}

	s, err := tfstate.ReadFile(stateFile)
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
