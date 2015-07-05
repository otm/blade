package main

import (
	"fmt"
	"os"

	"github.com/yuin/gopher-lua"
)

// LPrintHelp prints the help message
var LPrintHelp *lua.LFunction

func setupEnv() (L *lua.LState, runner *lua.LTable, cmd *lua.LTable) {
	emit("Setting up environment\n")
	L = lua.NewState()
	defer L.Close()

	// TODO (nils): Remove LPrintHelp and set the default function to lua.LNil
	LPrintHelp = L.NewFunction(printHelp)

	emit("Setting up runner\n")
	plugin := L.NewTable()
	plugin.RawSetString("watch", L.NewFunction(watch))

	blade := L.NewTable()
	blade.RawSetString("sh", L.NewFunction(func(L *lua.LState) int { return Sh(L) }))
	blade.RawSetString("_sh", L.NewFunction(func(L *lua.LState) int { return Sh(L, shNoEcho) }))
	blade.RawSetString("exec", L.NewFunction(func(L *lua.LState) int { return Sh(L, shNoAbort) }))
	blade.RawSetString("_exec", L.NewFunction(func(L *lua.LState) int { return Sh(L, shNoEcho, shNoAbort) }))
	blade.RawSetString("system", L.NewFunction(func(L *lua.LState) int { return Sh(L, shNoEcho, shNoAbort, shNoStdout) }))
	blade.RawSetString("shell", L.NewFunction(SetShell))
	blade.RawSetString("printStatus", L.NewFunction(printStatus))
	blade.RawSetString("compgen", L.NewFunction(Compgen))
	blade.RawSetString("help", L.NewFunction(Help))
	blade.RawSetString("setup", L.NewFunction(func(L *lua.LState) int { return 0 }))
	blade.RawSetString("teardown", L.NewFunction(func(L *lua.LState) int { return 0 }))
	blade.RawSetString("default", LPrintHelp)
	blade.RawSetString("plugin", plugin)
	L.SetGlobal("blade", blade)

	emit("Setting up cmd\n")
	cmds := L.NewTable()
	L.SetGlobal("cmd", cmds)
	L.SetGlobal("target", cmds)

	// Search for Bladerunner file
	filename := "Bladerunner"
	for {
		wd, _ := os.Getwd()
		emit("Looking for blade file: %v", wd)
		if _, err := os.Stat(filename); err == nil {
			emit("Found blade file")
			break
		}

		if wd == "/" {
			fmt.Printf("fatal: No blade file (or in any parent directory): %v\n", filename)
			os.Exit(1)
		}

		os.Chdir("..")
	}

	emit("Parsing blade file\n")
	if err := L.DoFile(filename); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	emit("Registring blade targets:\n")
	cmds.ForEach(func(key, value lua.LValue) {
		if f, ok := value.(*lua.LFunction); !ok {
			emit(" * %v [target]", key)
			_, name := subcommands.get(f)
			subcommands.rename(name, key.String())
		}
	})
	subcommands.validate()

	return L, blade, cmds
}

func runLFunc(L *lua.LState, runner *lua.LTable, fn string) {
	if err := L.CallByParam(lua.P{
		Fn:      runner.RawGetString(fn),
		NRet:    1,
		Protect: true,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	res := L.Get(-1)
	L.Pop(1)
	if b, ok := res.(lua.LBool); ok && b == false {
		emit("Aborting execution: result = %v", b)
		os.Exit(1)
	}
}

// TODO (nils): rewrite these function
func setup(L *lua.LState, runner *lua.LTable) {
	emit("Running blade setup")
	runLFunc(L, runner, "setup")
}

func teardown(L *lua.LState, runner *lua.LTable) {
	emit("Running blade teardown")
	runLFunc(L, runner, "teardown")
}

func defaultTarget(L *lua.LState, runner *lua.LTable) bool {
	emit("Running default target")
	runLFunc(L, runner, "default")
	// TODO(nils): Align setup, teardown and defaultTarget functions
	return true
}

func customTarget(L *lua.LState, runner *lua.LTable, target string, args []string) bool {
	emit("Looking up target: %v", target)
	fn := runner.RawGetString(target)
	if fn.Type() == lua.LTNil {
		emit("Unable to find target: %v, aborting...", target)
		return false
	}

	emit("Running target: %v", target)
	currentTarget = target

	// preparing variables to function
	var a []lua.LValue
	for _, arg := range args {
		a = append(a, lua.LString(arg))
	}

	if err := L.CallByParam(lua.P{
		Fn:      fn,
		NRet:    1,
		Protect: true,
	}, a...); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	res := L.Get(-1) // returned value
	L.Pop(1)         // remove received value
	if b, ok := res.(lua.LBool); ok && b == false {
		emit("Aborting execution: result = %v", b)
		return false
	}

	return true
}
