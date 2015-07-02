package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/yuin/gopher-lua"
)

func printHelp(L *lua.LState) int {
	fmt.Printf("Usage: blade [OPTION] [<target>] [<args>]\n")
	fmt.Printf("\nOptions:\n")
	flag.PrintDefaults()
	fmt.Printf("\nTargets:\n")
	for target, cmd := range subcommands {
		fmt.Printf("  %v: %v\n", target, strings.Trim(cmd.help, "\n"))
	}
	return 0
}
