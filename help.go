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

	// Check if the default function has been changed
	if blade, ok := L.GetGlobal("blade").(*lua.LTable); ok {
		if defaultTarget, ok := blade.RawGetString("default").(*lua.LFunction); ok && defaultTarget != LPrintHelp {
			defaultHelp := ""
			if h, ok := helps[defaultTarget]; ok {
				defaultHelp = h
			}
			fmt.Printf("  <default>: %v\n", strings.Trim(defaultHelp, "\n"))
		}
	}

	keys := make([]string, len(subcommands))
	i := 0
	for target := range subcommands {
		keys[i] = target
		i++
	}
	sort.Strings(keys)
	for _, target := range keys {
		fmt.Printf("  %v: %v\n", target, strings.Trim(subcommands[target].help, "\n"))
	}

	return 0
}
