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

	LPrintHelp = L.NewFunction(printHelp)

	emit("Setting up runner\n")
	blade := L.NewTable()
	blade.RawSetString("sh", L.NewFunction(func(L *lua.LState) int {
		return Sh(L)
	}))
	blade.RawSetString("_sh", L.NewFunction(func(L *lua.LState) int {
		return Sh(L, shNoEcho)
	}))
	blade.RawSetString("exec", L.NewFunction(func(L *lua.LState) int {
		return Sh(L, shNoAbort)
	}))
	blade.RawSetString("_exec", L.NewFunction(func(L *lua.LState) int {
		return Sh(L, shNoEcho, shNoAbort)
	}))
	blade.RawSetString("system", L.NewFunction(func(L *lua.LState) int {
		return Sh(L, shNoEcho, shNoAbort, shNoStdout)
	}))
	blade.RawSetString("shell", L.NewFunction(SetShell))
	blade.RawSetString("printStatus", L.NewFunction(printStatus))
	blade.RawSetString("compgen", L.NewFunction(Compgen))
	blade.RawSetString("help", L.NewFunction(Help))
	blade.RawSetString("setup", L.NewFunction(func(L *lua.LState) int { return 0 }))
	blade.RawSetString("teardown", L.NewFunction(func(L *lua.LState) int { return 0 }))
	blade.RawSetString("default", LPrintHelp)
	L.SetGlobal("blade", blade)

	plugin := L.NewTable()
	plugin.RawSetString("watch", L.NewFunction(watch))
	blade.RawSetString("plugin", plugin)

	emit("Setting up cmd\n")
	cmds := L.NewTable()
	L.SetGlobal("cmd", cmds)
	L.SetGlobal("target", cmds)

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
		if f, ok := value.(*lua.LFunction); ok {
			emit(" * %v [target]", key)
			cmd := target{cmd: f}
			if c, ok := compgens[f]; ok {
				emit(" * %v [compgen]", key)
				cmd.compgen = c
			}
			if h, ok := helps[f]; ok {
				emit(" * %v [help]", key)
				cmd.help = h
			}
			subcommands[key.String()] = cmd
		}
	})

	return L, blade, cmds
}

func setup(L *lua.LState, runner *lua.LTable) {
	emit("Running blade setup")
	if err := L.CallByParam(lua.P{
		Fn:      runner.RawGetString("setup"),
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

func teardown(L *lua.LState, runner *lua.LTable) {
	emit("Running blade teardown")
	if err := L.CallByParam(lua.P{
		Fn:      runner.RawGetString("teardown"),
		NRet:    1,
		Protect: true,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	res := L.Get(-1)
	L.Pop(1)
	if b, ok := res.(lua.LBool); ok && b == false {
		emit("Aborting execution: Returned value not ok: %v", b)
		os.Exit(1)
	}

}

func defaultTarget(L *lua.LState, runner *lua.LTable) bool {
	emit("Running default target")
	if err := L.CallByParam(lua.P{
		Fn:      runner.RawGetString("default"),
		NRet:    1,
		Protect: true,
	}); err != nil {
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

func customTarget(L *lua.LState, runner *lua.LTable, target string, args []string) bool {
	emit("Looking up target: %v", target)
	fn := runner.RawGetString(target)
	if fn.Type() == lua.LTNil {
		emit("Unable to find target: %v, aborting...", target)
		return false
	}
	emit("Running target: %v", target)

	currentTarget = target

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
