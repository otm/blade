package main

import (
	"strings"

	"github.com/yuin/gopher-lua"
)

func decorateStringLib(L *lua.LState) {
	mt := L.GetMetatable(lua.LString("")).(*lua.LTable)
	mt.RawSetString("split", L.NewClosure(split))
	mt.RawSetString("c", L.NewClosure(word))
	mt.RawSetString("trim", L.NewClosure(trim))
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

func word(L *lua.LState) int {
	s := L.CheckString(1)
	i := L.CheckInt(2)
	if int(i) <= 0 {
		L.RaiseError("Index mus be greater then 0")
	}

	parts := strings.Fields(s)
	if int(i) > len(parts) {
		L.Push(lua.LString(""))
		return 1
	}

	L.Push(lua.LString(parts[i-1]))
	return 1
}

func trim(L *lua.LState) int {
	s := L.CheckString(1)
	cutset := L.OptString(2, "\n ")

	s = strings.Trim(s, cutset)
	L.Push(lua.LString(s))
	return 1
}
