package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/yuin/gopher-lua"
)

func shift(slice []string) []string {
	if len(slice) > 1 {
		return slice[1:]
	}

	return make([]string, 0)
}

func printFlags() {
	var s []string
	flag.VisitAll(func(fl *flag.Flag) {
		s = append(s, "-"+fl.Name)
	})
	fmt.Printf("%v", strings.Join(s, " "))
}

func compgen(L *lua.LState, subcommands commands) {
	// create and shift of the program name
	args := flag.Args()
	args = shift(args)
	index := 1

	// analyse flags, since it has to start with "-" len(args) must be greater then 0
	for len(args) > 0 {
		if string(args[0][0]) == "-" {
			args = shift(args)
			if flg.compCWords == index {
				printFlags()
				return
			}
			index++
		} else {
			break
		}
	}

	// analyse runner targets
	if flg.compCWords == index {
		var s []string
		for target := range subcommands {
			s = append(s, target)
		}
		fmt.Printf("%v", strings.Join(s, " "))
		return
	}

	// pass it on to the runner target
	target := args[0]
	if cmd, ok := subcommands[target]; ok && cmd.compgen != nil {
		fmt.Printf("%v", cmd.compgen.compgen(L, args, flg.compCWords-index))
	}

}
