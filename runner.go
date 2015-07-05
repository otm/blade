package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"

	"github.com/yuin/gopher-lua"
)

var (
	// shell contains the current shell used when running shell commands
	shell = "bash"

	// flg contains all flags from the command line
	flg *flags

	// subcommands are the targets defiend in Lua
	subcommands = make(targets)

	// done is used for signaling that we should abort the blade target execution
	// and quit
	done chan struct{}

	// cleanup is used by go routines to receive notification that blade is shutting
	// down so they can clean up and shutdown
	cleanup = make(chan func(), 10)

	// currentTarget is the name of the curret running target
	// This is used when generating error messages for fatal errors when running
	// targets.
	currentTarget string

	errAbort           = errors.New("user: abort")
	errUndefinedTarget = errors.New("fatal: undefined target")
)

type target struct {
	cmd     *lua.LFunction
	help    string
	compgen compgenerator
	valid   bool
}

type targets map[string]*target

func (t targets) get(fn *lua.LFunction) (subcmd *target, name string) {
	for name, subcmd := range t {
		if subcmd.cmd == fn {
			return subcmd, name
		}
	}

	subcmd = &target{cmd: fn}
	name = "tmp-dummy-" + strconv.Itoa(len(t))
	t[name] = subcmd
	return subcmd, name
}

func (t targets) rename(from, to string) error {
	if _, ok := t[from]; !ok {
		return fmt.Errorf("Target not found: %v", from)
	}

	t[to] = t[from]
	delete(t, from)
	t[to].valid = true
	return nil
}

func (t targets) validate() {
	for _, subcmd := range t {
		if !subcmd.valid {
			fmt.Fprintf(os.Stderr, "fatal: no command bound to target\n")
			fmt.Fprintf(os.Stderr, "fatal: check blade.help and blade.compgen calls\n")
			fmt.Fprintf(os.Stderr, "debug: the following functions are ok:\n")
			t.printValidTargets()
			os.Exit(1)
		}
	}
}

func (t targets) printValidTargets() {
	for name, subcmd := range t {
		if subcmd.valid {
			fmt.Fprintf(os.Stderr, "debug:  * %v\n", name)
		}
	}
}

type flags struct {
	debug       bool
	genBashConf bool
	compgen     bool
	compCWords  int
}

func init() {
	flg = &flags{}
	flag.BoolVar(&flg.debug, "debug", false, "Enable debug output")
	flag.BoolVar(&flg.compgen, "compgen", false, "Used for bash compleation")
	flag.BoolVar(&flg.genBashConf, "generate-bash-conf", false, "Generate bash completion configuration")
	flag.IntVar(&flg.compCWords, "comp-cwords", 0, "Used for bash compleation")
}

// setupInterupt is used for catching ctrl-c when we want to abort the current
// running target.
func setupInterupt() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		sig := <-c
		emit("Received: %v", sig)
		if done != nil {
			emit("Signaling shutdown")
			done <- struct{}{}
		} else {
			fmt.Fprintf(os.Stderr, "aborting: due to %v signal", sig)
		}
	}()
}

func main() {
	flag.Parse()
	defer clean()

	setupInterupt()

	L, tbl, cmd := setupEnv()

	if flg.genBashConf {
		generateBashConfig()
		return
	}

	if flg.compgen {
		compgen(L, subcommands)
		return
	}

	if flag.Arg(0) == "help" {
		printHelp(L)
		return
	}

	target := flag.Arg(0)

	err := setup(L, tbl, target)
	if err != nil {
		emitErr("%v: setup", err)
		return
	}
	defer teardown(L, tbl, target)

	if flag.NArg() == 0 {
		defaultTarget(L, tbl)
		return
	}

	if flag.NArg() > 0 {
		err := lookupLFunc(L, cmd, target)
		if err != nil {
			emitFatal("%v", err)
		}

		customTarget(L, cmd, target, flag.Args()[1:])
		if done != nil {
			emit("Waiting for done signal")
			<-done
		}
		return
	}
}

// clean is used for signals registered go routins that they should shutdown
func clean() {
	for {
		var fn func()
		select {
		case fn = <-cleanup:
			emit("Running cleanup function")
			fn()
		default:
			emit("All cleaned up")
			return
		}
	}
}

func emit(msgfmt string, args ...interface{}) {
	if flg.debug == false {
		return
	}
	log.Printf(msgfmt, args...)
}

func emitErr(msgfmt string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, msgfmt, args...)
}

func emitFatal(msgfmt string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, msgfmt, args...)
	os.Exit(1)
}
