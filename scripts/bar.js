import { createConnection } from "net";
import { exec } from "child_process";

const runtimeDir = process.env.XDG_RUNTIME_DIR;
const instanceSig = process.env.HYPRLAND_INSTANCE_SIGNATURE;

if (!runtimeDir || !instanceSig) {
  console.error(
    "Error: environment variables XDG_RUNTIME_DIR or HYPRLAND_INSTANCE_SIGNATURE not found."
  );
  process.exit(1);
}

const socketPath = `${runtimeDir}/hypr/${instanceSig}/.socket2.sock`;

const client = createConnection(socketPath, () => {
  console.log("Connected to Hyprland IPC socket, subscribing to events...");
  client.write("subscribe:workspace\0");
  client.write("subscribe:activewindow\0");
  client.write("subscribe:activeworkspace\0");

  for (const event of Object.keys(COMMANDS)) {
    runShellCommand(event);
  }
});

client.on("error", (err) => {
  console.error(`Failed to connect to Hyprland IPC socket: ${err.message}`);
  process.exit(1);
});

const COMMANDS = {
  workspace: `
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
  `,
  activewindow: `
    active_win=$(hyprctl activewindow -j | jq -c .); 
    eww update ACTIVE_WINDOW="$active_win"
  `,
};

function runShellCommand(eventType) {
  const cmd = COMMANDS[eventType];
  if (!cmd) return;

  exec(
    `/usr/bin/env bash -c '${cmd.replace(/'/g, `'\\''`)}'`,
    { timeout: 10000 },
    (error, stdout, stderr) => {
      if (error) {
        console.error(
          `Subprocess error for ${eventType}: ${stderr || error.message}`
        );
      } else {
        process.stdout.write(stdout);
      }
    }
  );
}

let buffer = "";

client.on("data", (data) => {
  buffer += data.toString();

  let index;
  while ((index = buffer.indexOf("\n")) >= 0) {
    const message = buffer.slice(0, index);
    buffer = buffer.slice(index + 1);

    for (const eventType of Object.keys(COMMANDS)) {
      if (message.includes(eventType)) {
        runShellCommand(eventType);
        break;
      }
    }
  }
});

client.on("end", () => {
  console.log("Disconnected from Hyprland IPC.");
});

process.on("SIGINT", () => {
  console.log("Exiting...");
  client.end();
  process.exit();
});
