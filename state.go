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
	if flg.bladefile != "" {
		filename = flg.bladefile
	}

	for {
		wd, _ := os.Getwd()
		emit("Looking for blade file: %v", wd)
		if _, err := os.Stat(filename); err == nil {
			emit("Found blade file: %v", filename)
			break
		}

		if wd == "/" {
			if flg.compgen {
				emit("fatal: No blade file (or in any parent directory): %v\n", filename)
			} else {
				fmt.Printf("fatal: No blade file (or in any parent directory): %v\n", filename)
			}
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
			_, name := subcommands.get(f)
			subcommands.rename(name, key.String())
		}
	})
	subcommands.validate()

	return L, blade, cmds
}

func runLFunc(L *lua.LState, tbl *lua.LTable, fn string, args ...lua.LValue) error {
	if err := L.CallByParam(lua.P{
		Fn:      tbl.RawGetString(fn),
		NRet:    1,
		Protect: true,
	}, args...); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	res := L.Get(-1)
	L.Pop(1)
	if b, ok := res.(lua.LBool); ok && b == false {
		return errAbort
	}

	return nil
}

func setup(L *lua.LState, blade *lua.LTable, target string) error {
	emit("Running blade setup")
	return runLFunc(L, blade, "setup", lua.LString(target))
}

func teardown(L *lua.LState, blade *lua.LTable, target string) error {
	emit("Running blade teardown")
	return runLFunc(L, blade, "teardown", lua.LString(target))
}

func defaultTarget(L *lua.LState, blade *lua.LTable) error {
	emit("Running default target")
	return runLFunc(L, blade, "default")
}

func lookupLFunc(L *lua.LState, tbl *lua.LTable, key string) error {
	emit("Looking up target: %v", key)
	value := tbl.RawGetString(key)
	if value.Type() == lua.LTNil {
		emit("Unable to find target: %v, aborting...", key)
		return errUndefinedTarget
	}

	return nil
}

func customTarget(L *lua.LState, cmds *lua.LTable, target string, args []string) error {
	emit("Running target: %v", target)
	currentTarget = target

	// preparing variables to function
	var lvArgs []lua.LValue
	for _, arg := range args {
		lvArgs = append(lvArgs, lua.LString(arg))
	}

	return runLFunc(L, cmds, target, lvArgs...)
}
