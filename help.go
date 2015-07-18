package main

import (
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/yuin/gopher-lua"
)

func printHelp(L *lua.LState) int {
	fmt.Printf("Usage: blade [OPTION] [<target>] [<args>]\n")
	fmt.Printf("\nOptions:\n")
	flag.PrintDefaults()
	fmt.Printf("\nTargets:\n")

	// transform pretty prints the default key = "" (empty string)
	transform := func(s string) string {
		if s == "" {
			return "<default>"
		}
		return s
	}

	// Sort and print targets
	keys := make([]string, len(subcommands))
	i := 0
	for target := range subcommands {
		keys[i] = target
		i++
	}
	sort.Strings(keys)

	for _, target := range keys {
		fmt.Printf("  %v: %v\n", transform(target), strings.Trim(subcommands[target].help, "\n"))
	}

	return 0
}
