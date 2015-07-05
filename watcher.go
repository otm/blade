package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/yuin/gopher-lua"

	"gopkg.in/fsnotify.v1"
)

func watch(L *lua.LState) int {
	var (
		args      = L.CheckTable(1)
		dir       = lua.LVAsString(args.RawGetString("dir"))
		recursive = lua.LVAsBool(args.RawGetString("recursive"))
		filter    = lua.LVAsString(args.RawGetString("filter"))
		rFilter   = regexp.MustCompile(filter)
		lvexclude = args.RawGetString("exclude")
		callback  = args.RawGetString("callback")
		excludes  []string

		// abort is used for signaling to go process to stop processing file events
		abort = make(chan struct{})
	)

	if exclude, ok := lvexclude.(*lua.LTable); ok {
		exclude.ForEach(func(key, value lua.LValue) {
			excludes = append(excludes, lua.LVAsString(value))
		})
	}
	emit("Excludes: %v", excludes)

	if callback.Type() != lua.LTFunction {
		emitFatal("fatal: callback not defined or not function")
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		emitFatal("fatal: %v", err)
	}

	// register a function to receive events that we are shuting down
	// TODO (nils): This should be done by closing a chanel instead
	cleanup <- func() {
		abort <- struct{}{}
		emit("Closing watcher")
		watcher.Close()
	}

	// start up the event processor
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				processFileEvent(L, callback, rFilter, event)
			case err := <-watcher.Errors:
				emit("watcher: %v", err)
			case <-abort:
				emit("Shuting down watch routine")
				return
			}
		}
	}()

	emit("Watching: %v", dir)
	err = watcher.Add(dir)
	if err != nil {
		log.Fatal(err)
	}

	if recursive {
		if err := addRecursivly(watcher, dir, excludes); err != nil {
			fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
			os.Exit(1)
		}
	}

	done = make(chan struct{})
	return 0
}

func processFileEvent(L *lua.LState, callback lua.LValue, filter *regexp.Regexp, event fsnotify.Event) {
	emit("event: %v", event)
	if event.Op&fsnotify.Write == fsnotify.Write {
		if filtered(event.Name, filter) {
			emit("modified file: %v", event.Name)
			notify(L, callback, event)
		}
	}
}

func filtered(file string, filter *regexp.Regexp) bool {
	if match := filter.MatchString(file); match {
		emit("%v: included", file)
		return true
	}
	emit("%v: skipping", file)
	return false
}

func addRecursivly(watcher *fsnotify.Watcher, dir string, excludes []string) error {
	emit("Watch: checking dir %v", dir)
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("error reading dir %v: %v", dir, err)
	}

Next:
	for _, file := range files {
		if file.IsDir() && !strings.HasPrefix(file.Name(), ".") {
			for _, exclude := range excludes {
				if strings.Contains(file.Name(), exclude) {
					emit("Skipping: %v/%v", dir, file.Name())
					continue Next
				}
			}

			emit("watching: %v", file.Name())
			err := watcher.Add(fmt.Sprintf("%v/%v", dir, file.Name()))
			if err != nil {
				return fmt.Errorf("error reading %v: %v", file.Name(), err)
			}
			err = addRecursivly(watcher, fmt.Sprintf("%v/%v", dir, file.Name()), excludes)
			if err != nil {
				return fmt.Errorf("error processing %v: %v", file.Name(), err)
			}
		}
	}

	return nil
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
	res := L.Get(-1)
	L.Pop(1)
	if b, ok := res.(lua.LBool); ok && b == false {
		emitFatal("%v", errAbort)
	}
}
