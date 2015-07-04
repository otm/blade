package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

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
		os.Exit(1)
	}

	ret := L.Get(-1)
	L.Pop(1)

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
	fmt.Printf("%v", strings.Join(s, " "))
}

func compgen(L *lua.LState, subcommands targets) {
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
			if target == "" {
				continue
			}
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

// blade -generate-bash-conf | sudo tee /etc/bash_completion.d/blade
func generateBashConfig() {
	conf := `_blade()
{
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    opts=$(blade -compgen -comp-cwords $COMP_CWORD ${COMP_WORDS[@]})
    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    return 0
}
complete -F _blade blade
`
	fmt.Print(conf)
}
