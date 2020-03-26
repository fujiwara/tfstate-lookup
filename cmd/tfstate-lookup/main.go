package main

import (
	"encoding/json"
	"os"

	"github.com/fujiwara/tfstate-lookup/tfstate"
)

func main() {
	s, err := tfstate.Read(os.Stdin)
	if err != nil {
		panic(err)
	}
	res, err := s.Lookup(os.Args[1])
	if err != nil {
		panic(err)
	}
	json.NewEncoder(os.Stdout).Encode(res)
}
