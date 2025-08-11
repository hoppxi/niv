local socket = require("socket")
local unix = socket.unix

local runtimeDir = os.getenv("XDG_RUNTIME_DIR")
local instanceSig = os.getenv("HYPRLAND_INSTANCE_SIGNATURE")

if not runtimeDir or not instanceSig then
  io.stderr:write("Error: XDG_RUNTIME_DIR or HYPRLAND_INSTANCE_SIGNATURE not found.\n")
  os.exit(1)
end

local socketPath = runtimeDir .. "/hypr/" .. instanceSig .. "/.socket2.sock"

local client, err = unix()
if not client then
  io.stderr:write("Failed to create unix socket: " .. err .. "\n")
  os.exit(1)
end

local ok, err = client:connect(socketPath)
if not ok then
  io.stderr:write("Failed to connect: " .. err .. "\n")
  os.exit(1)
end

client:send("subscribe:workspace\0")
client:send("subscribe:activewindow\0")
client:send("subscribe:activeworkspace\0")

local COMMANDS = {
  workspace = [[
    workspaces=$(
      hyprctl workspaces -j | jq -c '
        . as $ws
        | ($ws | map(.id)) as $existing_ids
        | [range(1;6)] as $all_ids
        | ($all_ids - $existing_ids) as $missing_ids
        | ($missing_ids | map({id: ., windows: 0})) as $missing_ws
        | ($ws + $missing_ws)
        | sort_by(.id)
      '
    );
    current_ws=$(hyprctl activeworkspace -j | jq -c .); 
    eww update WORKSPACES="$workspaces"; 
    eww update FOCUSED_WORKSPACE="$current_ws"
  ]],
  activewindow = [[
    active_win=$(hyprctl activewindow -j | jq -c .); 
    eww update ACTIVE_WINDOW="$active_win"
  ]],
}

local function runShellCommand(eventType)
  local cmd = COMMANDS[eventType]
  if not cmd then return end
  local handle = io.popen("/usr/bin/env bash -c '" .. cmd:gsub("'", "'\\''") .. "'")
  if handle then
    local result = handle:read("*a")
    handle:close()
    io.write(result)
  else
    io.stderr:write("Failed to execute command for " .. eventType .. "\n")
  end
end

local buffer = ""

while true do
  local data, err, partial = client:receive("*a")
  if not data and err ~= "timeout" then
    io.stderr:write("Socket error or closed: " .. tostring(err) .. "\n")
    break
  end
  buffer = buffer .. (data or partial or "")
  -- Process lines in buffer
  while true do
    local lineEnd = buffer:find("\n", 1, true)
    if not lineEnd then break end
    local line = buffer:sub(1, lineEnd - 1)
    buffer = buffer:sub(lineEnd + 1)
    for eventType, _ in pairs(COMMANDS) do
      if line:find(eventType) then
        runShellCommand(eventType)
        break
      end
    end
  end
  socket.sleep(0.01) -- avoid busy loop
end

client:close()
