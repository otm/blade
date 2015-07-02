package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/yuin/gopher-lua"
)

var (
	debug = false
)

// Action executes a command
func Action(L *lua.LState) int {
	s := L.ToString(1)
	fmt.Println("Action: ", s, " has been executed")

	L.Push(lua.LTrue)
	return 1
}

type compgenerator interface {
	compgen(L *lua.LState, compWords []string, compCWords int) string
}

type command struct {
	cmd     *lua.LFunction
	help    string
	compgen compgenerator
}

type commands map[string]command

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
		panic(err)
	}

	ret := L.Get(-1) // returned value
	L.Pop(1)

	return ret.String()
}

type strCompgen struct {
	s string
}

func (sc *strCompgen) compgen(L *lua.LState, compWords []string, compCWords int) string {
	return sc.s
}

var helps = make(map[*lua.LFunction]string)

// Help registers help commands in lua
func Help(L *lua.LState) int {
	targetFunc := L.CheckFunction(1)
	if v := L.ToString(2); v != "" {
		helps[targetFunc] = v
	} else {
		L.TypeError(2, lua.LTString)
	}

	return 0
}

var compgens = make(map[*lua.LFunction]compgenerator)

// Compgen registers autocompletion help for sub commands
func Compgen(L *lua.LState) int {
	targetFunc := L.CheckFunction(1)
	if v := L.ToFunction(2); v != nil {
		var c compgenerator = &funcCompgen{f: v}
		compgens[targetFunc] = c
	} else if v := L.ToString(2); v != "" {
		var c compgenerator = &strCompgen{s: v}
		compgens[targetFunc] = c
	} else {
		L.TypeError(2, lua.LTString)
	}

	return 0
}

// Sh runs a shell command
func Sh(L *lua.LState) int {

	cmd := exec.Command("bash", "-c", L.ToString(1))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				fmt.Printf("Error running command: Exit Status: %d\n", status.ExitStatus())
				L.Push(lua.LFalse)
				L.Push(lua.LNumber(status.ExitStatus()))
				return 2
			}
		}
	}
	L.Push(lua.LTrue)
	L.Push(lua.LNumber(0))
	return 2
}

type flags struct {
	debug      bool
	verbose    bool
	quiet      bool
	compgen    bool
	compCWords int
}

var flg *flags

// TODO (nils): A better flag parser is needed to handle git like cases
// examples:
// blade install -v
// blade test -system
// blade test -unit
func init() {
	flg = &flags{}
	//flag.Usage = usage
	flag.BoolVar(&flg.debug, "debug", false, "Enable debug output")
	flag.BoolVar(&flg.verbose, "verbose", false, "Enable verbose output")
	flag.BoolVar(&flg.quiet, "quiet", false, "Quiet, do not output")
	flag.BoolVar(&flg.compgen, "compgen", false, "Used for bash compleation")
	flag.IntVar(&flg.compCWords, "comp-cwords", 0, "Used for bash compleation")
}

/*
func usage() {
	fmt.Println("Some usage info")
}
*/

var subcommands = make(commands)

func main() {
	flag.Parse()

	L, tbl, cmd := setupEnv()

	if flg.compgen {
		compgen(L, subcommands)
		return
	}

	if flag.Arg(0) == "help" {
		printHelp(nil)
		return
	}

	setup(L, tbl)
	defer teardown(L, tbl)

	if flag.NArg() == 0 {
		defaultTarget(L, tbl)
		return
	}

	if flag.NArg() > 0 {
		customTarget(L, cmd, flag.Arg(0), flag.Args()[1:])
		return
	}
}

func emit(msgfmt string, args ...interface{}) {
	if flg.debug == false {
		return
	}
	log.Printf(msgfmt, args...)
}
