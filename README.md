## Blade
Blade is a task runner designed to be an small and easy to use task runner and a replacement when using makefiles for not intended use.

## Features
* Define tasks in lua
* Command line parameters are passed to the task
* Automatic generated documentation
* Create custom help messages for build targets
* Create custom bash completion for build targets
* Easily run shell commands
* Built in file watcher
* Easy install - one binary

## Install
To build from source you need Go

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

## Bash Completion
The -generate-bash-conf option outputs the bash completion configuration to stdout. Either manually copy it or you can for instance use `tee`:

```
blade -generate-bash-conf | sudo tee /etc/bash_completion.d/blade
```

**Note:** The location of the bash completion configuration might differ depending on distribution and platform

**Note:** zsh can also run bash completion commands.

## Getting Started
Create a `Bladerunner` file. All targets will be executed with the current directory set to the directory containing the `Bladerunner` file. The blade command will search for the Bladerunner file in the file tree.

To create a `hello` target define a function on the target table. In the function you can execute arbitrary lua code.

``` lua
function target.hello()
  print("hello world")
end
```

To execute the target run

```
blade hello
```

Lets expand the example by processing command line parameters.

``` lua
function target.hello(use)
  use = use or "lua"

  if use == "shell" then
    blade.sh([[echo "hello world"]])
  elseif use == "lua" then
    print("hello world")
  else
    print("unknown option: " .. use)
  end
end
```

Run the following to test the new target
```
blade hello
blade hello lua
blade hello shell
blade hello foo
```
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


## Blade API
A small set of convince functions are provided, attached to a lua table called `blade`.

### blade.shell(shell) => shell
The default shell used is bash, setting the `blade.shell` variable overrides that.

***Example:***
``` lua
sh = blade.shell("zsh")
print(sh)
-- prints 'zsh'
```

### blade.sh(command) => exitStatus, stdout, stderr
Run arbitrary shell commands, executed in bash by default. If the command returns a non zero exit code the target execution will be aborted. The command is echoed to stdout, to suppress this use `blade._sh` instead.

Returns the exit status and the standard output from the command

***Example:***
``` lua
exitStatus, out = blade.sh("echo 'Hello World'")
-- outputs:
-- echo 'Hello World'
-- Hello World

exitStatus, out = blade._sh("echo 'Hello World'")
-- outputs:
-- Helo World
```

### blade.exec(command) => exitStatus, stdout, stderr
`runner.exec`, does not abort target execution if the command returns a non zero exit code. `runner._exec` will suppress the command echo to stdout.

***Example:***
``` lua
blade.exec("false'")
print("command execution continues")

blade._exec("false")
print("commnd is not echoed to stdout, execution continues")

blade.sh("false")
print("This is not executed")
```

### blade.system(command) => exitStatus, stdout, stderr
Like `blade.sh` but does not check anything, or echo anything.

***Example:***
``` lua
code, out, err = blade.system("echo 'Hello World' && date -r")
print("code:", code)
print("stdout:", out)
print("stderr", err)
```

### blade.printStatus(message, status)
Prints a pretty printed status message to the terminal, normaly used for printing execution status.

***Example:***
``` lua
blade.printStatus("true", true)
blade.printStatus("false", false)
blade.printStatus("0", 0)
blade.printStatus("1", 1)
blade.printStatus("nil")
blade.printStatus("true (shell)", blade._exec("true"))
blade.printStatus("false (shell)", blade._exec("false"))
-- outputs
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
* equivalent: `arg = {key=name, ...}; fn(arg)`
