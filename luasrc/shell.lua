local sh = require('sh')

local M = {}

-- get global metatable
local mt = getmetatable(_G)
if mt == nil then
  mt = {}
  setmetatable(_G, mt)
end

-- set hook for undefined variables
mt.__index = function(t, cmd)
	return sh.command(cmd)
end

git = sh.git
sudo = sh.sudo

-- export command() function and configurable temporary "input" file
M.command = sh.command
M.subcommand = sh.subcommand
M.tmpfile = '/tmp/shluainput'

-- allow to call sh to run shell commands
setmetatable(M, {
	__call = function(_, cmd, ...)
		return sh.command(cmd, ...)
	end,
	__index = function(t, cmd)
		return sh.command(cmd)
	end
})

return M
