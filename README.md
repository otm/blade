## Blade
Blade is a task runner designed to be easy, small, highly powerful, and with built in Bash completion and documentation. It is portable and easy to install, only a single binary.

## Features
* Easy install - one binary
* Automatic generated documentation
* Automatic bash completion for defined tasks
* Command line parameters are passed to the task
* Create custom help messages for tasks with comments
* Create custom bash completion for build targets
* Call any program as it were a function with `sh` module
* Built in file watcher
* Easy and expressive as tasks are defined in Lua


## Contents
<!-- TOC depth:3 withLinks:1 updateOnSave:0 orderedList:0 -->

- [Blade](#blade)
- [Features](#features)
- [Contents](#contents)
- [Install](#install)
- [Bash Completion](#bash-completion)
- [Getting Started](#getting-started)
- [Targets](#targets)
	- [target: help](#target-help)
	- [target: <blank>](#target-blank)
- [Setup and teardown](#setup-and-teardown)
	- [blade.setup(target)](#bladesetuptarget)
	- [teardown(target)](#teardowntarget)
- [Shell Module](#shell-module)
	- [Multiple arguments](#multiple-arguments)
	- [Background Processing](#background-processing)
	- [Capturing and Printing Output](#capturing-and-printing-output)
	- [Piping](#piping)
	- [Aborting Execution when Commands Fails](#aborting-execution-when-commands-fails)
- [Blade API](#blade-api)
	- [blade.printStatus(message, status)](#bladeprintstatusmessage-status)
	- [blade.help(target, message)](#bladehelptarget-message)
	- [blade.compgen(target, optsOrFunction)](#bladecompgentarget-optsorfunction)
- [Plugins](#plugins)
	- [blade.plugin.watch{callback, dir, recursive, filter, exclude}](#bladepluginwatchcallback-dir-recursive-filter-exclude)
- [Lua](#lua)
	- [string:split(sep, cb) => iterator](#stringsplitsep-cb-iterator)
- [Build from Source](#build-from-source)
- [Cross Compile](#cross-compile)

<!-- /TOC -->

## Install
Pre built binaries can be downloaded at
https://github.com/otm/blade/releases/latest

Download the binary and copy it in your path.

If you prefer to build from source please read the section: [Build from Soruce](#build-from-source)

## Bash Completion
The -generate-bash-conf option outputs the bash completion configuration to stdout. Either manually copy it or you can for instance use `tee`:

```
blade -generate-bash-conf | sudo tee /etc/bash_completion.d/blade
```

**Note:** The location of the bash completion configuration might differ depending on distribution and platform

**Note:** zsh can also run bash completion commands.

## Getting Started
Create a `Bladefile` file in the current directory, the easiest way is to use the `blade` command.

``` sh
blade -init
```

This will create a minimal `Bladefile` with one target called `demo`. Tasks in blade are called targets. To execute the target demo target run:

``` sh
blade demo
```
The demo showcases some important features:
* Documentation of targets. Access documentation by running `blade` with no arguments.
* Receive command line arguments
* Execute shell commands

## Targets
Defining new blade targets is done by adding functions to the target table.

***Example:***
``` lua
function target.build()
  -- build target code
end
```

***Example: arguments***
``` lua
function target.install(devDeps)
  -- install target code
  -- example setting default values
  devDeps = devDeps or "true"
end
```

***Example: variable arguments***
``` lua
function target.install(...)
  -- If the ... notation is used arguments are assigned to the arg variable
  -- arg.n is special and returns the number of elements in arg
  -- To test: blade install -i --dev /var/log
  print("Number of inputs: ", arg.n)
  for index, value in ipairs(arg) do
    print(index, "=", value)
  end
end
```

### target: help
The only built in target is `help`. It will print an automatically generated help message. It is possible to target help messages, see `blade.help`

``` lua
blade help
```

### target: <blank>
If not defining a target when running blade the `help` target will be executed. This can be overridden by setting `blade.default`.

***Example:***
``` lua
-- set the default target to `test`
blade.default = target.test

-- run a custom function for the default target
function blade.default()
  -- default target code
end
```

## Setup and teardown
It is possible to run setup and teardown code that is run before and after the blade target. Both setup and teardown receive a `target` argument with the name of the current target to be run. If no target has been defined at the command line target will be an empty string. Returning false in the setup or teardown will abort the target execution.

### blade.setup(target)
***Example:***
``` lua
function blade.setup(target)
  -- setup code
end
```

### teardown(target)
***Example:***
``` lua
function blade.teardown(target)
  -- teardown code
end
```

## Shell Module
sh is a interface to call any program as it were a function. Programs are executed asynchronously to enable streaming of data in pipes. Therefor it is necessary to manually wait on programs.

``` lua
local sh = require("sh")

sh.echo("hello", "world"):print()
```

Output:
```
hello world
```

For commands with exotic names or names which are reserved words call `sh` directly.
``` lua
sh(./script-in-my-directory)
```
### Multiple arguments
Commands that take multiple arguments needs to be invoked with separate strings for each arguments. That is, `sh.tar("xzf", "test.tar")` will work; however, `sh.tar("xzf test.tar")` will not.

### Background Processing
By default all commands are executed in the background.
``` lua
-- non blocking
sh.sleep(3)
print("prints immediately")

-- block
sh.sleep(3):success()
print("...3 seconds later")

-- utilizing async
sleep = sh.sleep(3)
print("prints immediately")
sleep.success()
print("...3 seconds later")
```

### Capturing and Printing Output

#### print()
`print()` prints the command's combined output.

``` lua
-- print output of command
sh.echo("hello world"):print()
```

***Note:*** `print()` has to be called before any method that waits. For instance: `ok()`, `success()`, or `exitcode()`,

#### stdout([filename]), stderr([filename]), combinedOutput([filename])
All these three methods takes an optional filename argument. If the filename is omitted the function returns the output of the command.

If `filename` is given the output will be written to the file and returned.
``` lua
-- print output of command
output = sh.echo("hello world"):combinedOutput("/tmp/output")
print(output)
```

The example above will print `hello world` and it will write it to `/tmp/output`

### Piping
Bash like piping is done by calling methods on the previous commands.
``` lua
sh.du("-sb"):sort("-rn"):print()
```

### Aborting Execution when Commands Fails
There are several ways to wait for a command.

#### ok()
Waits for the command to finish and aborts execution if the command returns a non zero exit code. Example: `sh.ls():ok()`

#### success()
Returns true if the exit code of the command is zero, false otherwise. Example: `sh.ls():success()`

#### exitcode()
To access the exit code of a command call the method `exitcode()`. Example:
`sh.ls():exitcode()`

## Blade API
A small set of convince functions are provided, attached to a lua table called `blade`.

### blade.printStatus(message, status)
Prints a pretty printed status message to the terminal, normaly used for printing execution status.

***Example:***
``` lua
local sh = require("sh")

blade.printStatus("true", true)
blade.printStatus("false", false)
blade.printStatus("0", 0)
blade.printStatus("1", 1)
blade.printStatus("nil")
blade.printStatus("true (shell)", sh.date():success())
blade.printStatus("false (shell)", sh("false"):success())
-- outputs:
-- true                                                                  [ ok ]
-- false                                                                 [fail]
-- 0                                                                     [ ok ]
-- 1                                                                     [fail]
-- nil                                                                   [udef]
-- true (shell)                                                          [ ok ]
-- false (shell)                                                         [fail]
```

### blade.help(target, message)
blade.help associates a message with a target.

***Example:***
``` lua
function target.build()
  -- build target code
end

blade.help(target.build, "<dev|prod>")
```

### blade.compgen(target, optsOrFunction)
blade.compgen associates an opts string or a function that will be executed by blade if bash completion is set up. The function signature is:

function(compWords, compCWord)
* compWords: a table containing the arguments on the command line
* compCWord: a int pointing to the cursor position (zero indexed)

**Note on cursor position:** (cursor denoted by "|")
* blade target | ==> compWords = { target }, compCWord = 1  
* blade target opt1| ==> compWords = { target, opt1 }, compCWord = 1
* balde target opt1 | ==> compWords = { target, opt1 }, compCWord = 2

***Example:***
``` lua
function target.build()
  -- build target code
end

-- bind a static string
blade.compgen(target.build, "dev prod")

-- bind a function
blade.compgen(target.build, function(compWords, compCWord)
if compCWord == 1 then
   return "dev prod"
 end
 return ""
end)
```

## Plugins

### blade.plugin.watch{callback, dir, recursive, filter, exclude}
blade have a built-in simple file watcher.

* ***callback - function(file, op):*** function for processing file events
* ***dir - string:*** the directory to watch
* ***recursive - bool:*** watch sub directories recursively
* ***filter - string:*** files matching regexp will be sent processed
* ***exclude - {string, ...}:*** a table of strings of directories to exclude

***Note:*** Several watch statements can be specified in one target

``` lua
function cmd.watch()
	blade.plugin.watch{callback=onFileEvent, dir="."}
end

function onFileEvent(file, op)
  print("File: " .. file .. ", Operation: " .. op)
end
```

## Lua
This section contains some Lua tips for new users

* Define strings: `"str"`, `'str'` or `[[str]]`
* Read environment variables: `os.getenv("HOME")`
* if-else: `if <statement> then <code> elseif <statement> then <code> else <code> end`
* named function variables: `fn{key=name, ...}`
 equivalent: `arg = {key=name, ...}; fn(arg)`
* reading files in directory:
``` lua
for file in io.popen("ls -1 *.go"):lines() do
  --use file
end
```

### string:split(sep, cb) => iterator
Splitting strings can be done in many ways in Lua but they are all quite cumbersome. To aid this there is a non standard Lua function for splitting strings in blade

Example:
``` lua
out = "first\nsecond"
for line in out:split("\n") do
  print("i", line)
end

out:split("\n", function(line)
  print("cb", line)
end)
```

## Build from Source
To build from source you need a working Go installation, see https://golang.org/doc/install

```
go get github.com/otm/blade
go install github.com/otm/blade
```

Pre built binaries can be downloaded at
https://github.com/otm/blade/releases/latest

## Cross Compile
Getting blade to all your favorite platforms. Cross compiling can easily be done with gox. See https://github.com/mitchellh/gox for information about the tool. To setup and cross compile you can run.

```
blade goxSetup
blade build
```
