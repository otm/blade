package main

import (
	"fmt"
	"os"

	"github.com/otm/blade/parser"
	"github.com/otm/gluaflag"
	"github.com/otm/gluash"
	"github.com/yuin/gopher-lua"
)

// LPrintHelp prints the help message
var LPrintHelp *lua.LFunction

func setupEnv(src string) (L *lua.LState, runner *lua.LTable, cmd *lua.LTable) {
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
	blade.RawSetString("flag", L.NewFunction(NewFlag))
	blade.RawSetString("help", L.NewFunction(Help))
	blade.RawSetString("setup", L.NewFunction(func(L *lua.LState) int { return 0 }))
	blade.RawSetString("teardown", L.NewFunction(func(L *lua.LState) int { return 0 }))
	blade.RawSetString("default", LPrintHelp)
	blade.RawSetString("plugin", plugin)
	L.SetGlobal("blade", blade)

	emit("Preloading module: sh")
	L.PreloadModule("sh", gluash.Loader)

	emit("Preloading module: flag")
	L.PreloadModule("flag", gluaflag.Loader)

	emit("Setting up cmd\n")
	cmds := L.NewTable()
	L.SetGlobal("cmd", cmds)
	L.SetGlobal("target", cmds)

	emit("Decorating string library")
	decorateStringLib(L)

	// Search for Bladerunner file
	filename := findBladefile(flg.bladefile)
	var comments parser.Comments
	var err error

	if src == "" {
		emit("Parsing blade file\n")
		if err := L.DoFile(filename); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}

		emit("Parsing comments")
		comments, err = parser.File(filename)
		if err != nil {
			emitFatal("%v", err)
		}
	} else {
		emit("Parsing blade string\n")
		if err := L.DoString(src); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}

		emit("Parsing comments")
		comments = parser.String(src)
	}

	emit("Registring blade targets:\n")
	cmds.ForEach(func(key, value lua.LValue) {
		if f, ok := value.(*lua.LFunction); ok {
			emit(" * %v [target]", key)
			subcommand, name := subcommands.get(f)
			subcommands.rename(name, key.String())

			if comment, err := comments.Get(f.Proto.LineDefined - 1); err == nil && subcommand.help == "" {
				subcommand.help = comment.Value
			}

		}
	})

	// Check if we have a default target defined
	if blade.RawGetString("default").(*lua.LFunction).Proto != nil {
		if _, ok := subcommands[""]; !ok {
			subcommand, name := subcommands.get(blade.RawGetString("default").(*lua.LFunction))
			subcommands.rename(name, "")

			if comment, err := comments.Get(subcommand.cmd.Proto.LineDefined - 1); err == nil && subcommand.help == "" {
				subcommand.help = comment.Value
			}
		}
		emit("Add default target to subcommands, l: %v", blade.RawGetString("default").(*lua.LFunction).Proto.LineDefined)

	}
	//subcommands = append(subcommands, )

	subcommands.validate()

	return L, blade, cmds
}

func findBladefile(filename string) string {
	files := []string{"Bladerunner", "Bladefile"}
	if filename != "" {
		files = []string{flg.bladefile}
	}

	// Walk the file tree towards the root
	for {
		wd, _ := os.Getwd()
		emit("Looking for blade file: %v", wd)
		for _, file := range files {
			emit(" - checking: %v", file)
			if _, err := os.Stat(file); err == nil {
				emit("Found blade file: %v", file)
				return file
			}
		}

		if wd == "/" {
			if flg.compgen {
				emit("fatal: No blade file (or in any parent directory): %v\n", files)
			} else {
				fmt.Printf("fatal: No blade file (or in any parent directory): %v\n", files)
			}
			os.Exit(1)
		}

		os.Chdir("..")
	}
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

func require(L *lua.LState, module string) lua.LValue {
	fn := L.GetGlobal("require").(*lua.LFunction)

	if err := L.CallByParam(lua.P{
		Fn:      fn,
		NRet:    1,
		Protect: true,
	}, lua.LString(module)); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	res := L.Get(-1)
	L.Pop(1)
	return res
}

func customTarget(L *lua.LState, cmds *lua.LTable, target string, args []string) error {
	emit("Running target: %v", target)
	currentTarget = target

	if fn := subcommands[currentTarget].flagFn; fn != nil {
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

		t, err := gluaflag.Parse(L, ud, args)
		if err != nil {
			return err
		}
		return runLFunc(L, cmds, target, t)
	}

	// preparing variables to function
	var lvArgs []lua.LValue
	for _, arg := range args {
		lvArgs = append(lvArgs, lua.LString(arg))
	}

	return runLFunc(L, cmds, target, lvArgs...)
}
