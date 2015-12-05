package sh

import "github.com/yuin/gopher-lua"

var exports = map[string]lua.LGFunction{}

// Loader is used for preloading a module
func Loader(L *lua.LState) int {

	// register functions to the table
	mod := L.SetFuncs(L.NewTable(), exports)

	// set up meta table
	mt := L.NewTable()
	L.SetField(mt, "__index", L.NewClosure(moduleIndex))
	L.SetField(mt, "__call", L.NewClosure(moduleCall))
	L.SetMetatable(mod, mt)

	shMetaTable := L.NewTypeMetatable(luaShTypeName)
	L.SetField(shMetaTable, "__call", L.NewFunction(shCall))
	L.SetField(shMetaTable, "__index", L.NewFunction(shIndex))

	// returns the module
	L.Push(mod)
	return 1
}

// moduleIndex creates and returns userdata shell command (sh) defined by the
// index.
func moduleIndex(L *lua.LState) int {
	index := L.CheckString(2)

	cmd := &shellCommand{
		path: index,
	}

	L.Push(cmd.UserData(L))
	return 1
}

func moduleCall(L *lua.LState) int {
	path := L.CheckString(2)
	args := checkStrings(L, 3)

	cmd, err := newShellCommand(path, args...)
	checkError(L, err)

	err = cmd.command.Start()
	checkError(L, err)

	L.Push(cmd.UserData(L))
	return 1
}
