package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

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

	var workspace string
	env, _ := ioutil.ReadFile(filepath.Join(filepath.Dir(stateFile), "environment"))
	workspace = string(env)

	f, err := os.Open(stateFile)
	if err != nil {
		return err
	}
	defer f.Close()

	s, err := tfstate.ReadWithWorkspace(f, workspace)
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
