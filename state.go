package main

import (
	"os"

	"github.com/yuin/gopher-lua"
)

func setupEnv() (L *lua.LState, runner *lua.LTable, cmd *lua.LTable) {
	emit("Setting up environment\n")
	L = lua.NewState()
	defer L.Close()

	emit("Setting up runner\n")
	tbl := L.NewTable()
	tbl.RawSetString("sh", L.NewFunction(Sh))
	tbl.RawSetString("compgen", L.NewFunction(Compgen))
	tbl.RawSetString("help", L.NewFunction(Help))
	tbl.RawSetString("setup", L.NewFunction(func(L *lua.LState) int { return 0 }))
	tbl.RawSetString("teardown", L.NewFunction(func(L *lua.LState) int { return 0 }))
	tbl.RawSetString("default", L.NewFunction(printHelp))
	L.SetGlobal("runner", tbl)

	emit("Setting up cmd\n")
	cmds := L.NewTable()
	L.SetGlobal("cmds", cmds)

	emit("Parsing blade file\n")
	if err := L.DoFile("make.lua"); err != nil {
		panic(err)
	}

	emit("Registring blade targets:\n")
	cmds.ForEach(func(key, value lua.LValue) {
		if f, ok := value.(*lua.LFunction); ok {
			emit(" * %v [target]", key)
			cmd := command{cmd: f}
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

	return L, tbl, cmds
}

func setup(L *lua.LState, runner *lua.LTable) {
	emit("Running blade setup")
	if err := L.CallByParam(lua.P{
		Fn:      runner.RawGetString("setup"),
		NRet:    1,
		Protect: true,
	}); err != nil {
		panic(err)
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
		panic(err)
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
		panic(err)
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
	if fn == nil {
		emit("Unable to find target: %v, aborting...", target)
		return false
	}
	emit("Running target: %v", target)

	var a []lua.LValue
	for _, arg := range args {
		a = append(a, lua.LString(arg))
	}

	if err := L.CallByParam(lua.P{
		Fn:      fn,
		NRet:    1,
		Protect: true,
	}, a...); err != nil {
		panic(err)
	}
	res := L.Get(-1) // returned value
	L.Pop(1)         // remove received value
	if b, ok := res.(lua.LBool); ok && b == false {
		emit("Aborting execution: result = %v", b)
		return false
	}

	return true
}
