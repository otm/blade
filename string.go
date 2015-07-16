package main

import (
	"strings"

	"github.com/yuin/gopher-lua"
)

func decorateStringLib(L *lua.LState) {
	strlib := L.GetGlobal("string")
	if strlib, ok := strlib.(*lua.LTable); ok {
		index := strlib.RawGetString("__index")
		if index, ok := index.(*lua.LTable); ok {
			index.RawSetString("split", L.NewClosure(split))
		}
	}
}

func split(L *lua.LState) int {
	s := L.CheckString(1)
	sep := L.CheckString(2)
	parts := strings.Split(s, sep)

	if L.GetTop() == 3 {
		fn := L.CheckFunction(3)
		p := lua.P{Fn: fn, NRet: 0, Protect: true}
		for i, str := range parts {
			if err := L.CallByParam(p, lua.LString(str), lua.LNumber(i)); err != nil {
				emitFatal("%v", err)
			}
		}
	}

	i := 0
	iterator := func(L *lua.LState) int {
		if i == len(parts) {
			return 0
		}

		L.Push(lua.LString(parts[i]))
		L.Push(lua.LNumber(i))
		i++
		return 2
	}

	L.Push(L.NewClosure(iterator))
	return 1
}
