local sh = require('sh')

-- demo [name] - a short blade demo
function target.demo(name)

  -- set a default value to name
  name = name or "please enter your name: blade demo <name>"

  sh.echo("Hi, " .. name .. "!"):print()
end
