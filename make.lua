
function runner.setup()
	print("setup")
end

function runner.teardown()
	print("teardown")
end

function runner.default()
	print("======= default =======")
	print("ENV: ", os.getenv("HOME"))
	print("=======================")
end

function cmds.watch()
	runner.plugin.watch(function()
		print("I'm run by the watch plugin")
	end)
	runner.interpreter()
end

function cmds.test(...)
  print("Number of inputs: ", arg.n)
  for k, v in ipairs(arg) do
    print(k, "=", v)
  end

	ok, exitStatus = runner.sh("echo 'running shell command'; for i in `seq 1 4`; do echo 'loop ' $i; done")
	ok, exitStatus = runner.sh("date -r")
  runner.sh([[echo "this should not run"]])
end

runner.compgen(cmds.test, function(compWords, compCWords)
  if compCWords == 1 then
	   return "unit system"
   end
   return ""
end)

runner.help(cmds.test, "<unit|system>")
