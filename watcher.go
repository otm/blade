package main

import (
	"fmt"
	"log"
	"os"

	"github.com/yuin/gopher-lua"

	"gopkg.in/fsnotify.v1"
)

func watch(L *lua.LState) int {
	args := L.CheckTable(1)
	dir := lua.LVAsString(args.RawGetString("dir"))
	recursive := lua.LVAsBool(args.RawGetString("recursive"))
	filter := lua.LVAsString(args.RawGetString("filter"))
	callback := args.RawGetString("callback")

	if callback.Type() != lua.LTFunction {
		fmt.Fprintf(os.Stdout, "fatal: callback not defined or not function\n")
		os.Exit(1)
	}
	_, _ = recursive, filter

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Fprintf(os.Stdout, "fatal: %v\n", err)
		os.Exit(1)
	}

	abort := make(chan struct{})
	cleanup <- func() {
		abort <- struct{}{}
		emit("Closing watcher")
		watcher.Close()
	}

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				emit("event: %v", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					emit("modified file: %v", event.Name)
					notify(L, callback, event)
				}
			case err := <-watcher.Errors:
				emit("watcher: %v", err)
			case <-abort:
				emit("Shuting down watch routine")
				return
			}
		}
	}()

	err = watcher.Add(dir)
	if err != nil {
		log.Fatal(err)
	}

	done = make(chan struct{})
	return 0
}

func notify(L *lua.LState, callback lua.LValue, event fsnotify.Event) {
	if err := L.CallByParam(lua.P{
		Fn:      callback,
		NRet:    1,
		Protect: true,
	}, lua.LString(event.Name), lua.LString("write")); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	res := L.Get(-1) // returned value
	L.Pop(1)         // remove received value
	if b, ok := res.(lua.LBool); ok && b == false {
		emit("Aborting execution: result = %v", b)
	}

}
