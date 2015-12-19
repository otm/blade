package main

import "github.com/yuin/gopher-lua"

// NewFlag creates a flagset
func NewFlag(L *lua.LState) int {
	targetFunc := L.CheckFunction(1)
	setupFn := L.CheckFunction(2)

	if isDefault(L, targetFunc) {
		L.RaiseError("Can not set flags on default function")
	}
	subcmd, _ := subcommands.get(targetFunc)
	subcmd.flagFn = setupFn

	return 0
}
