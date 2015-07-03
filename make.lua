
function blade.setup()
	print("setup")
end

function blade.teardown()
	print("teardown")
end

function blade.default()
	print("======= default =======")
	print("ENV: ", os.getenv("HOME"))
	print("=======================")
end

function cmd.watch()
	blade.plugin.watch{callback=notExported, dir="test", recursive=true, filter="*.go"}
end

function notExported(file, op)
  print("Handle watched call: " .. file .. " (" .. op .. ")")
end

function cmd.test(...)
  -- Running lua code
  -- This shows how to handle arguments from the command line
  print("Number of inputs: ", arg.n)
  for k, v in ipairs(arg) do
    print(k, "=", v)
  end

  -- output command status
  blade.printStatus("true", true)
  blade.printStatus("false", false)
  blade.printStatus("0", 0)
  blade.printStatus("1", 1)
  blade.printStatus("nil")


  -- capture exit status and command output
  exitStatus, out = blade.sh("echo 'Blade runner'")
  print("Exit: " .. exitStatus .. ", Output: ".. out)

  -- no echo of running command
  blade._sh("echo 'no cmd echo'")

  -- do not break on command error
  blade.exec("false")

  -- automaticly break on errors
	blade.sh("date -r")

  -- this statement is unreachable due to error on previus statement
  blade.sh([[echo "this should not run"]])
end

blade.compgen(cmd.test, function(compWords, compCWords)
  if compCWords == 1 then
	   return "unit system"
   end
   return ""
end)

blade.help(cmd.test, "<unit|system>")
