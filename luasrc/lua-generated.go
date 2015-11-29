package luasrc 

const (
Sh = `--[[
The MIT License (MIT)

Copyright (c) 2015 Serge Zaitsev

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
]]
local M = {}
local abort = true

local function copy(obj, seen)
	if type(obj) ~= 'table' then return obj end
	if seen and seen[obj] then return seen[obj] end
	local s = seen or {}
	local res = setmetatable({}, getmetatable(obj))
	s[obj] = res
	for k, v in pairs(obj) do res[copy(k, s)] = copy(v, s) end
	return res
end

-- converts key and it's argument to "-k" or "-k=v" or just ""
local function arg(k, a)
	if not a then return k end
	if type(a) == 'string' and #a > 0 then return k..'='..a end
	if type(a) == 'number' then return k..'='..tostring(a) end
	if type(a) == 'boolean' and a == true then return k end
	if type(a) == 'function' then return "" end
	error('invalid argument type', type(a), a)
end

-- converts nested tables into a flat list of arguments and concatenated input
local function flatten(t)
	local result = {args = {}, input = ''}

	local function f(t)
		local keys = {}
		for k, v in ipairs(t) do
			keys[k] = true
			if type(v) == 'table' then
				f(v)
			else
				table.insert(result.args, v)
			end
		end
		for k, v in pairs(t) do
			if k == '__input' then
				result.input = result.input .. v
			elseif k == 'cmd' then
			elseif k == 'stdout' then
			elseif k == 'stderr' then
			elseif k == 'exitcode' then
			elseif not keys[k] and k:sub(1, 1) ~= '_' then
				local key = '-'..k
				if #k > 1 then key = '-' ..key end
				table.insert(result.args, arg(key, v))
			end
		end
	end

	f(t)
	return result
end

-- returns a function that executes the command with given args and returns its
-- output, exit status etc
local function command(cmd, ...)
	local prearg = {...}
	return function(...)
		local args = flatten({...})
		local s = cmd
		for _, v in ipairs(prearg) do
			s = s .. ' ' .. v
		end
		for k, v in pairs(args.args) do
			s = s .. ' ' .. v
		end

		if args.input then
			local f = io.open(M.tmpfile, 'w')
			f:write(args.input)
			f:close()
			s = s .. ' <'..M.tmpfile
		end
		print("cmd", s)

		local exit, output, stderr = blade.system(s)
		os.remove(M.tmpfile)

		local t = {
			__input = output,
			cmd = cmd,
			stdout = output,
			stderr = stderr,
			exitcode = exit,
			print = function(self)
				io.write(self.__input)
				return self
			end,
			lines = function(self)
				s = tostring(self.__input)
				if s:sub(-1)~="\n" then s=s.."\n" end
				return s:gmatch("(.-)\n")
			end,
		}
		local mt = {
			__index = function(self, k, ...)
				if self.exitcode ~= 0 then
						if abort then
							os.exit(self.exitcode)
						end
						return function()
							return self
						end
				end
				return command(k)
			end,
			__tostring = function(self)
				-- return trimmed command output as a string
				return self.__input:match('^%s*(.-)%s*$')
			end
		}
		return setmetatable(t, mt)
	end
end

-- creates sub commands
local function subcommand(...)
	local prearg = {...}
	return setmetatable({}, {
		__call = function(_, cmd, ...)
			local foo = copy(prearg)
			table.insert(foo, cmd)
			return command(unpack(foo))(...)
		end,
		__index = function(t, cmd)
			local foo = copy(prearg)
			table.insert(foo, cmd)
			return command(unpack(foo))
		end
	})
end

-- get global metatable
local mt = getmetatable(_G)
if mt == nil then
	mt = {}
	setmetatable(_G, mt)
end

-- export command() function and configurable temporary "input" file
M.command = command
M.abort = function(abrt)
	abort = abrt
end
M.subcommand = subcommand
M.tmpfile = '/tmp/shluainput'
M.git = subcommand('git')
M.sudo = subcommand('sudo')

-- allow to call sh to run shell commands
setmetatable(M, {
	__call = function(_, cmd, ...)
		return command(cmd)(...)
	end,
	__index = function(t, cmd)
		return command(cmd)
	end
})

return M
`
Shell = `local sh = require('sh')

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
`
)
