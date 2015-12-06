package luasrc 

const (
Bladeinit = `local sh = require('sh')

-- demo [name] - a short blade demo
function target.demo(name)

  -- set a default value to name
  name = name or "unknown"

  -- run echo and print the result
  sh.echo("Hi", name):print()

  -- to run non compatible commands just call sh directly
  sh("echo", "- With blade it is easy to run shell commands"):print()

  -- to automaticly abort the task if a shell command fails use "ok()"
  sh.echo("- There are several helper functions for instance:"):print():ok()

  for method, help in pairs({print="Print stdout and stderr", ok="Check exit code, andn abort if non zero", success="Returns true if exitcode==0, false otherwise"}) do
    print(method, help)
  end

  -- to check if a command executed correctly use success()
  if not sh("false"):success() then
    print("false did not execute successfully")
  end

  -- iterating lines, and colomns
  for line in sh.echo("foo", "bar", "\n", "biz", "buz"):lines() do
    print(line:c(2))
    -- output:
    -- bar
    -- buz
  end

end
`
)
