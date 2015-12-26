package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/otm/blade/luasrc"
	"github.com/yuin/gopher-lua"
)

//go:generate go run scripts/include.go

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
	flagFn  *lua.LFunction
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
	init        bool
	compCWords  int
	bladefile   string
	src         string
}

func init() {
	flg = &flags{}
	flag.BoolVar(&flg.debug, "debug", false, "Enable debug output")
	flag.BoolVar(&flg.compgen, "compgen", false, "Used for bash compleation")
	flag.BoolVar(&flg.genBashConf, "generate-bash-conf", false, "Generate bash completion configuration")
	flag.IntVar(&flg.compCWords, "comp-cwords", 0, "Used for bash compleation")
	flag.StringVar(&flg.bladefile, "f", "", "Absolute path to non default blade file")
	flag.StringVar(&flg.src, "c", "", "Read bladefile from commandline")
	flag.BoolVar(&flg.init, "init", false, "Create a Bladefilein the current directory")
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
			close(done)
		} else {
			fmt.Fprintf(os.Stderr, "aborting: due to %v signal\n", sig)
		}
	}()
}

func writeFile() {
	file := "Bladefile"
	_, err := os.Stat(file)
	if err == nil {
		emitFatal("Configuration file exits: %v\n", file)
	}
	if !os.IsNotExist(err) {
		emitFatal("Error checking file existens: %v\n", err)
	}

	//fmt.Println(luasrc.Bladeinit)
	err = ioutil.WriteFile(file, []byte(luasrc.Bladeinit), 0644)
	if err != nil {
		emitFatal("Could not write configuration: %v", err)
	}
}

func main() {
	flag.Parse()

	if flg.genBashConf {
		generateBashConfig()
		return
	}

	if flg.init {
		writeFile()
		return
	}

	if flg.compgen {
		compgen()
		return
	}

	setupInterupt()

	L, blade, cmd := setupEnv(flg.src)

	if flag.Arg(0) == "help" {
		printHelp(L)
		return
	}

	target := flag.Arg(0)

	err := setup(L, blade, target)
	if err != nil {
		emitErr("%v: setup", err)
		return
	}
	defer teardown(L, blade, target)

	if flag.NArg() == 0 {
		defaultTarget(L, blade)
		return
	}

	if flag.NArg() > 0 {
		err := lookupLFunc(L, cmd, target)
		if err != nil {
			emitFatal("%v\n", err)
		}

		err = customTarget(L, cmd, target, flag.Args()[1:])
		if err != nil {
			// TODO: Make error message look nicer
			// emitErr("%v\n\n", err)
			printSubcommandHelp(target, L)
			return
		}
		wait(done)
		return
	}
}

func pause() {
	if done != nil {
		return
	}

	emit("Setting up done chanel")
	done = make(chan struct{})
}

func wait(ch chan struct{}) {
	if ch == nil {
		return
	}

	emit("Waiting for done signal")
	fmt.Printf("watching: ctrl-c to abort\n")
	for {
		select {
		case <-ch:
			emit("Done waiting")
			return
		default:
			time.Sleep(10 * time.Millisecond)
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

var emitFatal = func(msgfmt string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, msgfmt, args...)
	os.Exit(1)
}
