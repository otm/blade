package sh

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/yuin/gopher-lua"
)

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

func doString(src string, t *testing.T) string {
	L := lua.NewState()
	defer L.Close()
	L.PreloadModule("sh", Loader)

	restorer := captureStdOut()
	err := L.DoString(src)
	out := restorer()
	if err != nil {
		t.Errorf("unable to run source: %v", err)
	}
	if len(out) == 0 {
		return out
	}

	return out[0 : len(out)-1]
}

func TestModuleCall(t *testing.T) {
	src := `
    local sh = require('sh')
    sh("echo", "foo", "bar"):print()
  `
	expected := "foo bar"
	got := doString(src, t)

	if got != expected {
		t.Errorf("expected: %v, got: %v\nsrc: %v", expected, got, src)
	}
}

func TestIndexCall(t *testing.T) {
	src := `
    local sh = require('sh')
    sh.echo("foo", "bar"):print()
  `
	expected := "foo bar"
	got := doString(src, t)

	if got != expected {
		t.Errorf("expected: %v, got: %v\nsrc: %v", expected, got, src)
	}
}

func TestPipe(t *testing.T) {
	src := `
    local sh = require('sh')
    sh.echo("foo", "bar\n", "biz", "buz"):grep("foo"):print()
  `
	expected := "foo bar"
	got := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestLines(t *testing.T) {
	src := `
    local sh = require('sh')
    for line in sh.echo("foo bar\nbiz", "buz"):lines() do
      print(line)
    end
  `
	expected := "foo bar\nbiz buz"
	got := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestOK(t *testing.T) {
	src := `
    local sh = require('sh')
    sh.echo("foo"):ok()
    print("ok")
  `
	expected := "ok"
	got := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestOKChaining(t *testing.T) {
	src := `
    local sh = require('sh')
    function fail()
      sh.echo("foo"):ok():print()
    end

    ok, err = pcall(fail)
    print(ok)
    print(err)
  `
	expected := "false" + "\n" + "<string>:4: Do not call `ok` or `success` before print"
	got := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestNotOK(t *testing.T) {
	src := `
    local sh = require('sh')
    function fail()
      sh.grep("-d"):ok()
    end

    ok, err = pcall(fail)
    print(ok)
    print(err)
  `
	expected := `false
<string>:4: exit status 2`
	got := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestSuccess(t *testing.T) {
	src := `
    local sh = require('sh')
    ok = sh.echo("foo"):success()
    print(ok)
  `
	expected := "true"
	got := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestNotSuccess(t *testing.T) {
	src := `
    local sh = require('sh')
    ok = sh.grep("-d"):success()

    print(ok)
  `
	expected := "false"
	got := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestExitcode(t *testing.T) {
	src := `
    local sh = require('sh')
    exitcode = sh.echo("foo"):exitcode()
    print(exitcode)
  `
	expected := "0"
	got := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestNotExitcode(t *testing.T) {
	src := `
    local sh = require('sh')
    exitcode = sh.grep("-d"):exitcode()

    print(exitcode)
  `
	expected := "2"
	got := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestStdout(t *testing.T) {
	src := `
    local sh = require('sh')
    out = sh.echo("foo"):stdout()
    print(out)
  `
	expected := "foo" + "\n"
	got := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestStderr(t *testing.T) {
	src := `
    local sh = require('sh')
    out = sh("./stderr.test.sh"):stderr()
    print(out)
  `
	expected := "foo" + "\n"
	got := doString(src, t)

	if got != expected {
		t.Errorf("expected: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestWriteStdoutToFile(t *testing.T) {
	src := `
    local sh = require('sh')
    tmp = "./remove.me"
    out = sh.echo("foo"):stdout(tmp)
    print(out)
  `
	expected := "foo" + "\n"
	file := "./remove.me"
	defer os.Remove(file)
	got := doString(src, t)

	if got != expected {
		t.Errorf("expected stdout: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
	dat, err := ioutil.ReadFile(file)
	if err != nil {
		t.Errorf("unable to read file: `%v`", file)
	}
	if string(dat) != expected {
		t.Errorf("expected file: `%v`, got: `%v`\nsrc: %v", expected, string(dat), src)
	}
}

func TestWriteStderrToFile(t *testing.T) {
	src := `
    local sh = require('sh')
    tmp = "./remove.me"
    out = sh("./stderr.test.sh"):stderr(tmp)
    print(out)
  `
	expected := "foo" + "\n"
	file := "./remove.me"
	defer os.Remove(file)
	got := doString(src, t)

	if got != expected {
		t.Errorf("expected stdout: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
	dat, err := ioutil.ReadFile(file)
	if err != nil {
		t.Errorf("unable to read file: `%v`", file)
	}
	if string(dat) != expected {
		t.Errorf("expected file: `%v`, got: `%v`\nsrc: %v", expected, string(dat), src)
	}
}

func TestCombindedOutput(t *testing.T) {
	src := `
    local sh = require('sh')
    out = sh("./stderr.test.sh"):combinedOutput()
    print(out)
  `
	expected := "foo" + "\n"
	got := doString(src, t)

	if got != expected {
		t.Errorf("expected stdout: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestWriteCombindedOutputToFile(t *testing.T) {
	src := `
    local sh = require('sh')
    tmp = "./remove.me"
    out = sh("./stderr.test.sh"):combinedOutput(tmp)
    print(out)
  `
	expected := "foo" + "\n"
	file := "./remove.me"
	defer os.Remove(file)
	got := doString(src, t)

	if got != expected {
		t.Errorf("expected stdout: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
	dat, err := ioutil.ReadFile(file)
	if err != nil {
		t.Errorf("unable to read file: `%v`", file)
	}
	if string(dat) != expected {
		t.Errorf("expected file: `%v`, got: `%v`\nsrc: %v", expected, string(dat), src)
	}
}

func TestSetGlobalAbort(t *testing.T) {
	src := `
    local sh = require('sh')
    sh{abort=true}
    conf = sh{}

    for k, v in pairs(conf) do
      print(k .. "=" .. tostring(v))
    end
    `
	expected := "abort=true"
	got := doString(src, t)

	if got != expected {
		t.Errorf("expected stdout: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}

func TestGlobalAbort(t *testing.T) {
	src := `
    local sh = require('sh')
    sh{abort=true}

    function fail()
      print("this should print")
      sh("false"):print()
      print("but not this")
    end

    function fail2()
      print("this should print")
      sh("false"):ok()
      print("but not this")
    end

    function fail3()
      print("this should print")
      sh("false"):success()
      print("but not this")
    end

    function fail4()
      print("this should print")
      sh("false")
      print("and this")
    end

    ok, err = pcall(fail)
    ok, err = pcall(fail2)
    ok, err = pcall(fail3)
    ok, err = pcall(fail4)
    `
	expected := strings.Repeat("this should print\n", 4) + "and this"
	got := doString(src, t)

	if got != expected {
		t.Errorf("expected stdout: `%v`, got: `%v`\nsrc: %v", expected, got, src)
	}
}
