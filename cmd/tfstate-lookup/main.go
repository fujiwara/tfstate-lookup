package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fujiwara/tfstate-lookup/tfstate"
	"github.com/manifoldco/promptui"
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
		interactive      bool
		timeout          time.Duration
	)
	for _, name := range DefaultStateFiles {
		if _, err := os.Stat(name); err == nil {
			defaultStateFile = name
			break
		}
	}

	flag.StringVar(&stateLoc, "state", defaultStateFile, "tfstate file path or URL")
	flag.StringVar(&stateLoc, "s", defaultStateFile, "tfstate file path or URL")
	flag.BoolVar(&interactive, "i", false, "interactive mode")
	flag.DurationVar(&timeout, "timeout", 0, "timeout for reading tfstate")
	flag.Parse()

	var ctx = context.Background()
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	state, err := tfstate.ReadURL(ctx, stateLoc)
	if err != nil {
		return err
	}
	if len(flag.Args()) == 0 {
		names, err := state.List()
		if err != nil {
			return err
		}
		if interactive {
			selected, err := promptForSelection(names)
			if err != nil {
				return err
			}
			obj, err := state.Lookup(selected)
			if err != nil {
				return err
			}
			printObject(obj)
		} else {
			fmt.Println(strings.Join(names, "\n"))
		}
	} else {
		obj, err := state.Lookup(flag.Arg(0))
		if err != nil {
			return err
		}
		printObject(obj)
	}
	return nil
}

func printObject(obj *tfstate.Object) {
	b := obj.Bytes()
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

func promptForSelection(choices []string) (string, error) {
	prompt := promptui.Select{
		Label:             "Select an item",
		Items:             choices,
		StartInSearchMode: true,
		Size:              20,
		Searcher: func(input string, index int) bool {
			return strings.Contains(choices[index], input)
		},
	}
	_, result, err := prompt.Run()
	if err != nil {
		return "", err
	}
	return result, nil
}
