package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

var origArgs = os.Args

func captureStdOut() func() string {
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	return func() string {
		w.Close()
		os.Stdout = old // restoring the real stdout
		return <-outC
	}
}

func captureStderr() func() string {
	old := os.Stderr // keep backup of the real Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	return func() string {
		w.Close()
		os.Stderr = old // restoring the real stdout
		return <-outC
	}
}

func doString(t *testing.T, src string, args ...string) (stdout, stderr string) {
	os.Args = origArgs
	os.Args = append(os.Args, "-c", src)
	os.Args = append(os.Args, args...)
	restoreStdout := captureStdOut()
	restoreStderr := captureStderr()
	main()
	stdout = restoreStdout()
	stderr = restoreStderr()
	if len(stdout) == 0 || string(stdout[len(stdout)-1:len(stdout)]) != "\n" {
		return stdout, stderr
	}

	return stdout[0 : len(stdout)-1], stderr
}

func check(t *testing.T, expected, got, src string) {
	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestCallTarget(t *testing.T) {
	src := `
    function target.foo()
      print("foo bar")
    end
  `
	expected := "foo bar"
	got, _ := doString(t, src, "foo")
	check(t, expected, got, src)
}

func TestCallTargetWithArgument(t *testing.T) {
	src := `
    function target.foo(name)
      print(name)
    end
  `
	expected := "bar"
	got, _ := doString(t, src, "foo", "bar")
	check(t, expected, got, src)
}

func TestSetup(t *testing.T) {
	src := `
    function blade.setup(target)
      print(target)
    end

    function target.foo()
      print("target")
    end
  `
	expected := "foo\ntarget"
	got, _ := doString(t, src, "foo")
	check(t, expected, got, src)
}

func TestTeardown(t *testing.T) {
	src := `
    function blade.teardown(target)
      print(target)
    end

    function target.foo()
      print("target")
    end
  `
	expected := "target\nfoo"
	got, _ := doString(t, src, "foo")
	check(t, expected, got, src)
}

func TestOverrideDefault(t *testing.T) {
	src := `
    function blade.default()
      print("default")
    end

    function target.foo()
      print("target")
    end
  `
	expected := "default"
	got, _ := doString(t, src)
	check(t, expected, got, src)
}

func TestGlobCommandLineParameters(t *testing.T) {
	src := `
    function target.glob(...)
      print("inputs: " .. arg.n)
      for k, v in ipairs(arg) do
        print(k .. " = " .. v)
      end
    end
  `
	inputs := []string{"a", "b", "c", "d"}
	expected := fmt.Sprintf("inputs: %v", len(inputs))
	for i, input := range inputs {
		expected = fmt.Sprintf("%v\n%v = %v", expected, i+1, input)
	}
	got, _ := doString(t, src, "glob", "a", "b", "c", "d")
	check(t, expected, got, src)
}

func TestGluash(t *testing.T) {
	src := `
		local sh = require("sh")
    function target.fooer()
      sh.echo("foo"):print()
    end
  `
	expected := "foo"
	got, _ := doString(t, src, "fooer")
	check(t, expected, got, src)
}

func TestFlags(t *testing.T) {
	src := `
		function target.fooer(flags)
			print("foo " .. flags.name)
    end

		blade.flag(target.fooer, function(flag)
			flag:string("name", "John Dow", "How to foo")
		end)
  `
	expected := "foo bar"
	got, _ := doString(t, src, "fooer", "-name", "bar")
	check(t, expected, got, src)
}

func TestFailFlags(t *testing.T) {
	src := `
		function target.fooer(flags)
			print("foo " .. flags.name)
    end

		blade.flag(target.fooer, function(flag)
			flag:string("name", "John Dow", "How to foo")
		end)
  `
	expected := ""
	expectedStderr := strings.Join([]string{
		"flag provided but not defined: -foo",
		"Usage:",
		"  -name string",
		"    	How to foo (default \"John Dow\")",
		"",
	}, "\n")
	got, stderr := doString(t, src, "fooer", "-foo", "bar")
	check(t, expected, got, src)
	check(t, expectedStderr, stderr, src)
}

func TestFlagsCompgen(t *testing.T) {
	src := `
		function target.fooer(flags)
			print("foo " .. flags.name)
    end

		blade.flag(target.fooer, function(flag)
			flag:string("name", "John Dow", "How to foo", function()
				return "fi fi fo fum"
			end)
		end)
  `
	expected := "fi fi fo fum"
	expectedStderr := strings.Join([]string{""}, "\n")
	got, stderr := doString(t, src, "-compgen", "-comp-cwords", "3", "blade", "fooer", "-name")
	check(t, expected, got, src)
	check(t, expectedStderr, stderr, src)
}
