package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

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
		os.Exit(1)
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
	fmt.Printf("%v", strings.Join(s, " "))
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

	// pass it to the runner flag target
	if fn := subcommands[target].flagFn; fn != nil {
		_ = require(L, "flag").(*lua.LTable)
		ud := gluaflag.New(L)

		if err := L.CallByParam(lua.P{
			Fn:      fn,
			NRet:    0,
			Protect: true,
		}, ud); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}

		if gf, ok := ud.Value.(*gluaflag.Gluaflag); ok {
			fmt.Print(gf.Compgen(L, flg.compCWords-index, args))
		}
	}

}

// blade -generate-bash-conf | sudo tee /etc/bash_completion.d/blade
func generateBashConfig() {
	conf := `_blade()
{
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    if [[ ${prev} == -f ]]; then
      if  [[ $(declare -f _filedir) ]]; then
        _filedir
      else
        COMPREPLY=( $(compgen -f -- ${cur}) )
      fi
      return 0
    fi

    flags=$(echo "${COMP_WORDS[@]}")
    flag=$(expr "${flags}" : '.*\(-f [^ ]* *\)')

    opts=$(blade $flag -compgen -comp-cwords $COMP_CWORD ${COMP_WORDS[@]})
    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    return 0
}
complete -F _blade blade
`
	fmt.Print(conf)
}
