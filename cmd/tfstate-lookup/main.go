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
	"github.com/simeji/jid"
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
		runJid           bool
		dump             bool
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
	flag.BoolVar(&runJid, "j", false, "run jid after selecting an item")
	flag.BoolVar(&dump, "dump", false, "dump all resources")
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
	var key string
	if len(flag.Args()) > 0 {
		key = flag.Arg(0)
	} else {
		if dump {
			return dumpObjects(state)
		}
		// list
		names, err := state.List()
		if err != nil {
			return err
		}
		if !interactive {
			fmt.Println(strings.Join(names, "\n"))
			return nil
		}
		key, err = promptForSelection(names)
		if err != nil {
			return err
		}
	}

	obj, err := state.Lookup(key)
	if err != nil {
		return err
	}
	if runJid {
		return jidObject(obj)
	} else {
		return printObject(obj)
	}
}

func dumpObjects(state *tfstate.TFState) error {
	res, err := state.Dump()
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(res)
}

func printObject(obj *tfstate.Object) error {
	b := obj.Bytes()
	w := os.Stdout
	if isatty.IsTerminal(w.Fd()) && (bytes.HasPrefix(b, []byte("[")) || bytes.HasPrefix(b, []byte("{"))) {
		var out bytes.Buffer
		json.Indent(&out, b, "", "  ")
		out.WriteRune('\n')
		_, err := out.WriteTo(w)
		return err
	} else {
		_, err := fmt.Fprintln(w, string(b))
		return err
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

func jidObject(obj *tfstate.Object) error {
	ea := &jid.EngineAttribute{
		DefaultQuery: ".",
		Monochrome:   false,
		PrettyResult: true,
	}
	r := bytes.NewReader(obj.Bytes())
	e, err := jid.NewEngine(r, ea)
	if err != nil {
		return err
	}
	result := e.Run()
	if err := result.GetError(); err != nil {
		return err
	}
	fmt.Println(result.GetContent())
	return nil
}
