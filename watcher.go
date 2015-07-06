package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/yuin/gopher-lua"

	"gopkg.in/fsnotify.v1"
)

func watch(L *lua.LState) int {
	emit("Starting fs watcher setup")
	w := newWatcher(L.ToTable(1))
	w.start(L)

	return 0
}

type watcher struct {
	callback  lua.LValue
	dir       string
	recursive bool
	filter    *regexp.Regexp
	excludes  []string

	fsWatcher *fsnotify.Watcher
}

func newWatcher(args *lua.LTable) *watcher {
	var err error

	w := &watcher{
		callback:  args.RawGetString("callback"),
		dir:       lua.LVAsString(args.RawGetString("dir")),
		recursive: lua.LVAsBool(args.RawGetString("recursive")),
		filter:    regexp.MustCompile(lua.LVAsString(args.RawGetString("filter"))),
	}

	if tbl, ok := args.RawGetString("exclude").(*lua.LTable); ok {
		w.excludes = make([]string, tbl.Len())
		i := 0
		tbl.ForEach(func(key, value lua.LValue) {
			w.excludes[i] = lua.LVAsString(value)
			i++
		})
	}

	if w.callback.Type() != lua.LTFunction {
		emitFatal("fatal: callback not defined or not function")
	}

	w.fsWatcher, err = fsnotify.NewWatcher()
	if err != nil {
		emitFatal("fatal: %v", err)
	}

	return w
}

func (w *watcher) start(L *lua.LState) error {
	pause()
	go w.watch(L)
	err := w.addWatchers(w.dir)
	return err
}

func (w *watcher) watch(L *lua.LState) {
	for {
		select {
		case event := <-w.fsWatcher.Events:
			w.processFileEvent(L, event)
		case err := <-w.fsWatcher.Errors:
			emit("watcher: %v", err)
		case <-done:
			emit("Closing watcher")
			w.fsWatcher.Close()
			return
		}
	}
}

func (w *watcher) processFileEvent(L *lua.LState, event fsnotify.Event) {
	emit("event: %v", event)
	if event.Op&fsnotify.Write == fsnotify.Write {
		if w.filter.MatchString(event.Name) {
			emit("modified file: %v", event.Name)
			w.notify(L, event)
		}
	}
}

func (w *watcher) notify(L *lua.LState, event fsnotify.Event) {
	if err := L.CallByParam(lua.P{
		Fn:      w.callback,
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

func (w *watcher) addWatchers(name string) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return err
	}

	err = w.fsWatcher.Add(name)
	if err != nil {
		return err
	}

	if fi.Mode().IsRegular() {
		emit("watch: Adding regular file %v", name)
	} else {
		emit("watch: Adding directory %v", name)
	}

	if !w.recursive {
		return nil
	}

	emit("Watch: checking dir %v", name)
	files, err := ioutil.ReadDir(name)
	if err != nil {
		return fmt.Errorf("error reading dir %v: %v", name, err)
	}

Next:
	for _, file := range files {
		if !file.IsDir() || strings.HasPrefix(file.Name(), ".") {
			continue Next
		}

		for _, exclude := range w.excludes {
			if strings.Contains(file.Name(), exclude) {
				emit("Skipping: %v/%v", name, file.Name())
				continue Next
			}
		}

		err = w.addWatchers(fmt.Sprintf("%v/%v", name, file.Name()))
		if err != nil {
			return fmt.Errorf("error processing %v: %v", file.Name(), err)
		}
	}

	return nil
}
