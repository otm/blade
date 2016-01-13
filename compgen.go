package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/otm/blade/luasrc"
	"github.com/otm/gluaflag"
	"github.com/yuin/gopher-lua"
)

type compgenerator interface {
	compgen(L *lua.LState, compWords []string, compCWords int) string
}

type funcCompgen struct {
	f *lua.LFunction
}

func (sc *funcCompgen) compgen(L *lua.LState, compWords []string, compCWords int) string {
	tbl := L.NewTable()
	for _, v := range compWords {
		tbl.Append(lua.LString(v))
	}

	if err := L.CallByParam(lua.P{
		Fn:      sc.f,
		NRet:    1,
		Protect: true,
	}, tbl, lua.LNumber(compCWords)); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return ""
	}

	ret := L.Get(-1)
	L.Pop(1)

	emit("Got %v", ret.String())
	return ret.String()
}

type strCompgen struct {
	s string
}

func (sc *strCompgen) compgen(L *lua.LState, compWords []string, compCWords int) string {
	return sc.s
}

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
	printOptions(s)
}

// constants for printing compopts header
const (
	HEADER    = "### BEGIN COMPGEN INFO\n"
	FOOTER    = "### END COMPGEN INFO\n"
	PREFIX    = "# "
	SEPARATOR = ": "
)

func compHeader() string {
	generateOption := func(key, value string) string {
		return PREFIX + key + SEPARATOR + value + "\n"
	}

	header := ""
	if compopts.filedir {
		header = header + generateOption("mode", "filedir")
	}

	if header != "" {
		header = HEADER + header + FOOTER
	}

	return header
}

func printOptions(opts []string) {
	fmt.Printf("%v", compHeader())
	fmt.Printf("%v", strings.Join(opts, "\n"))
}

func compgen() {
	// create and shift of the program name
	args := flag.Args()
	args = shift(args)
	index := 1
	prev := ""

	// analyse flags, since it has to start with "-" len(args) must be greater then 0
	for len(args) > 0 {
		if string(args[0][0]) == "-" || prev == "-f" {
			prev, args = args[0], shift(args)
			if flg.compCWords == index {
				printFlags()
				return
			}
			index++
		} else {
			break
		}
	}

	// setup Lua environment
	L, _, _ := setupEnv(flg.src)

	// analyse runner targets
	if flg.compCWords == index {
		s := []string{"help"}
		for target := range subcommands {
			if target == "" {
				continue
			}
			s = append(s, target)
		}

		printOptions(s)
		return
	}

	// Dont segfault
	if len(args) == 0 {
		return
	}

	target := args[0]

	if target == "help" {
		var s []string
		for target := range subcommands {
			if target == "" {
				continue
			}
			s = append(s, target)
		}

		printOptions(s)
		return
	}

	// pass it on to the runner target
	if cmd, ok := subcommands[target]; ok && cmd.compgen != nil {
		// TODO: compgen should return a []string insted of a string
		fmt.Printf("%v", cmd.compgen.compgen(L, args, flg.compCWords-index))
	}

	// check if subtarget exists
	if _, ok := subcommands[target]; !ok {
		return
	}

	// pass it to the runner flag target
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
			printOptions(gf.Compgen(L, flg.compCWords-index, args))
		}
	}

}

// blade -generate-bash-conf | sudo tee /etc/bash_completion.d/blade
func generateBashConfig() {
	fmt.Print(luasrc.Compgen)
}
