package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/fujiwara/tfstate-lookup/tfstate"
	"github.com/mattn/go-isatty"
)

var DefaultStateFiles = []string{
	"terraform.tfstate",
	".terraform/terraform.tfstate",
}

func main() {
	if err := _main(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func _main() error {
	var (
		stateLoc         string
		defaultStateFile = DefaultStateFiles[0]
	)
	for _, name := range DefaultStateFiles {
		if _, err := os.Stat(name); err == nil {
			defaultStateFile = name
			break
		}
	}

	flag.StringVar(&stateLoc, "state", defaultStateFile, "tfstate file path or URL")
	flag.StringVar(&stateLoc, "s", defaultStateFile, "tfstate file path or URL")
	flag.Parse()

	state, err := tfstate.ReadURL(stateLoc)
	if err != nil {
		return err
	}
	if len(flag.Args()) == 0 {
		names, err := state.List()
		if err != nil {
			return err
		}
		fmt.Println(strings.Join(names, "\n"))
	} else {
		res, err := state.Lookup(flag.Arg(0))
		if err != nil {
			return err
		}
		b := res.Bytes()
		w := os.Stdout
		if isatty.IsTerminal(w.Fd()) && (bytes.HasPrefix(b, []byte("[")) || bytes.HasPrefix(b, []byte("{"))) {
			var out bytes.Buffer
			json.Indent(&out, b, "", "  ")
			out.WriteRune('\n')
			out.WriteTo(w)
		} else {
			fmt.Fprintln(w, string(b))
		}
	}
	return nil
}
