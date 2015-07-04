package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/yuin/gopher-lua"
)

var (
	debug = false
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

type target struct {
	cmd     *lua.LFunction
	help    string
	compgen compgenerator
}

type targets map[string]target

var helps = make(map[*lua.LFunction]string)
var compgens = make(map[*lua.LFunction]compgenerator)

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

type shOpts struct {
	noEcho  bool
	noAbort bool
}

func shNoEcho(opts *shOpts) {
	opts.noEcho = true
}

func shNoAbort(opts *shOpts) {
	opts.noAbort = true
}

// SetShell sets/returns the current shell
func SetShell(L *lua.LState) int {
	shell = L.CheckString(1)
	L.Push(lua.LString(shell))
	return 1
}

var shell = "bash"

// Sh runs a shell command
func Sh(L *lua.LState, options ...func(opts *shOpts)) int {
	b := new(bytes.Buffer)
	opts := &shOpts{}
	for _, option := range options {
		option(opts)
	}

	cmd := exec.Command(shell, "-c", L.ToString(1))
	cmd.Stdout = io.MultiWriter(b, os.Stdout)
	cmd.Stderr = os.Stderr
	if !opts.noEcho {
		fmt.Printf("%v\n", L.ToString(1))
	}
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				if opts.noAbort {
					L.Push(lua.LNumber(status.ExitStatus()))
					L.Push(lua.LString(b.String()))
					return 2
				}

				L.Error(lua.LString(fmt.Sprintf("blade: Target: [%v] Error: %v", currentTarget, status.ExitStatus())), 0)
				os.Exit(1)
			}
		}
	}
	L.Push(lua.LNumber(0))
	L.Push(lua.LString(b.String()))
	return 2
}

func printStatus(L *lua.LState) int {
	w, _, err := terminal.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		emit("Unable to get terminal size: %v", err)
		return 0
	}
	reset := "\033[0m"
	red := "\033[31m"
	green := "\033[32m"
	blue := "\033[34m"

	status := fmt.Sprintf("[%vudef%v]", blue, reset)
	ok := fmt.Sprintf("[ %vok%v ]", green, reset)
	fail := fmt.Sprintf("[%vfail%v]", red, reset)
	switch L.Get(2).Type() {
	case lua.LTBool:
		status = fail
		if L.ToBool(2) {
			status = ok
		}
	case lua.LTNumber:
		status = fail
		if L.ToInt(2) == 0 {
			status = ok
		}
	}

	message := L.ToString(1)
	fmt.Printf("%v%v%v\n", message, strings.Repeat(" ", w-len(message)-len(status)+len(reset)+len(red)-1), status)

	return 0
}

type flags struct {
	debug       bool
	genBashConf bool
	compgen     bool
	compCWords  int
}

var flg *flags

func init() {
	flg = &flags{}
	flag.BoolVar(&flg.debug, "debug", false, "Enable debug output")
	flag.BoolVar(&flg.compgen, "compgen", false, "Used for bash compleation")
	flag.BoolVar(&flg.genBashConf, "generate-bash-conf", false, "Generate bash completion configuration")
	flag.IntVar(&flg.compCWords, "comp-cwords", 0, "Used for bash compleation")
}

var subcommands = make(targets)
var done chan struct{}
var cleanup = make(chan func(), 10)
var currentTarget string

func setupInterupt() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		sig := <-c
		emit("Received: %v", sig)
		if done != nil {
			emit("Signaling shutdown")
			done <- struct{}{}
		} else {
			fmt.Fprintf(os.Stderr, "aborting: due to %v signal", sig)
		}
	}()
}

func main() {
	flag.Parse()
	defer clean()

	setupInterupt()

	L, tbl, cmd := setupEnv()

	if flg.genBashConf {
		generateBashConfig()
		return
	}

	if flg.compgen {
		compgen(L, subcommands)
		return
	}

	if flag.Arg(0) == "help" {
		printHelp(L)
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
		if done != nil {
			emit("Waiting for done signal")
			<-done
		}
		return
	}
}

func clean() {
	for {
		var fn func()
		select {
		case fn = <-cleanup:
			emit("Running cleanup function")
			fn()
		default:
			emit("All cleaned up")
			return
		}
	}
}

func emit(msgfmt string, args ...interface{}) {
	if flg.debug == false {
		return
	}
	log.Printf(msgfmt, args...)
}
