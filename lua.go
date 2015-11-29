package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/yuin/gopher-lua"
	"golang.org/x/crypto/ssh/terminal"
)

func loader(src string) lua.LGFunction {
	var loader lua.LGFunction = func(L *lua.LState) int {
		fn, err := L.LoadString(src)
		if err != nil {
			emitFatal("unable to load source: %v", err)
		}

		L.Push(fn)
		err = L.PCall(0, 1, nil)
		if err != nil {
			emitFatal("unable to run module source %v", err)
		}

		mod := L.Get(-1)

		L.Push(mod)
		return 1
	}

	return loader
}

// Help registers help commands in lua
func Help(L *lua.LState) int {
	targetFunc := L.CheckFunction(1)
	if v := L.ToString(2); v != "" {
		subcmd, name := subcommands.get(targetFunc)
		subcmd.help = v
		if isDefault(L, targetFunc) {
			subcommands.rename(name, "")
		}
	} else {
		L.TypeError(2, lua.LTString)
	}

	return 0
}

// isDefault checks if the fn is the default target function
func isDefault(L *lua.LState, fn *lua.LFunction) bool {
	blade := L.GetGlobal("blade").(*lua.LTable)
	defaultTarget := blade.RawGetString("default").(*lua.LFunction)
	if defaultTarget == fn {
		return true
	}
	return false
}

// Compgen registers autocompletion help for sub commands
func Compgen(L *lua.LState) int {
	targetFunc := L.CheckFunction(1)

	set := func(fn *lua.LFunction, c compgenerator) {
		subcmd, name := subcommands.get(fn)
		subcmd.compgen = c
		if isDefault(L, targetFunc) {
			subcommands.rename(name, "")
		}
	}

	if v := L.ToFunction(2); v != nil {
		var c compgenerator = &funcCompgen{f: v}
		set(targetFunc, c)
	} else if v := L.ToString(2); v != "" {
		var c compgenerator = &strCompgen{s: v}
		set(targetFunc, c)
	} else {
		L.TypeError(2, lua.LTString)
	}

	return 0
}

// shOpts is the Sh function's options
type shOpts struct {
	noEcho  bool
	noAbort bool
	stdout  io.Writer
}

// shNoEcho turns off echo of command
func shNoEcho(opts *shOpts) {
	opts.noEcho = true
}

// shNoAbort prevents abort on non zero exit status
func shNoAbort(opts *shOpts) {
	opts.noAbort = true
}

// shNoStdout turns off command output to stdout
func shNoStdout(opts *shOpts) {
	opts.stdout = ioutil.Discard
}

// SetShell sets/returns the current shell
func SetShell(L *lua.LState) int {
	shell = L.CheckString(1)
	L.Push(lua.LString(shell))
	return 1
}

// Sh runs a shell command
func Sh(L *lua.LState, options ...func(opts *shOpts)) int {
	stdoutBuf := new(bytes.Buffer)
	stderrBuf := new(bytes.Buffer)
	opts := &shOpts{stdout: os.Stdout}
	for _, option := range options {
		option(opts)
	}

	cmd := exec.Command(shell, "-c", L.ToString(1))
	cmd.Stdout = io.MultiWriter(stdoutBuf, opts.stdout)
	cmd.Stderr = io.MultiWriter(stderrBuf, os.Stderr)
	if !opts.noEcho {
		fmt.Printf("%v\n", L.ToString(1))
	}
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				if opts.noAbort {
					L.Push(lua.LNumber(status.ExitStatus()))
					L.Push(lua.LString(stdoutBuf.String()))
					L.Push(lua.LString(stderrBuf.String()))
					return 3
				}

				L.Error(lua.LString(fmt.Sprintf("blade: Target: [%v] Error: %v", currentTarget, status.ExitStatus())), 0)
				os.Exit(1)
			}
		}
	}
	L.Push(lua.LNumber(0))
	L.Push(lua.LString(stdoutBuf.String()))
	L.Push(lua.LString(stderrBuf.String()))
	return 3
}

// printStatus pretty prints a status message
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
