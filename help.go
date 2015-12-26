package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/otm/gluaflag"
	"github.com/yuin/gopher-lua"
)

func printHelp(L *lua.LState) int {
	if flag.NArg() > 1 {
		return printSubcommandHelp(flag.Arg(1), L)
	}

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
		fmt.Printf("  %v: %v\n", transform(target), strings.SplitN(strings.TrimSpace(subcommands[target].help), "\n", 2)[0])
	}

	return 0
}

func printSubcommandHelp(target string, L *lua.LState) int {
	if _, ok := subcommands[target]; !ok {
		fmt.Printf("Unknown subcommand: %v", target)
		os.Exit(1)
	}

	shortUsage := fmt.Sprintf("%v [<args>]", target)
	flagUsage := ""
	argUsage := ""

	// Check if we have a flag function, and run it
	if fn := subcommands[target].flagFn; fn != nil {
		_ = require(L, "flag").(*lua.LTable)
		ud := gluaflag.New(L, target)

		if err := L.CallByParam(lua.P{
			Fn:      fn,
			NRet:    0,
			Protect: true,
		}, ud); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}

		if gf, ok := ud.Value.(*gluaflag.FlagSet); ok {
			shortUsage = gf.ShortUsage()
			flagUsage = gf.FlagDefaults()
			argUsage = gf.ArgDefaults()
		}
	}

	fmt.Printf("Usage: blade %v\n", shortUsage)

	if subcommands[target].help != "" {
		fmt.Printf("\n%v\n", strings.TrimSpace(subcommands[target].help))
	}

	if flagUsage != "" {
		fmt.Printf("\nFlags:\n%v", flagUsage)
	}

	if argUsage != "" {
		fmt.Printf("\nArguments:\n%v", argUsage)
	}

	return 0
}
