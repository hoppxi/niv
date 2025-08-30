# Niv

Just making some widget built with Go and `EWW` for Hyprland. No big deal :)

# ToDo `frontend only`

- [x] Bar
- [x] Clock
- [x] Wlogout
- [x] Media Controls
- [x] Notification
- [ ] Planner
  - [x] Calendar
  - [ ] Todo App
  - [ ] Note App
- [ ] Quick Settings
  - [x] Landing
  - [x] Network
  - [x] Bluetooth
  - [x] Volume Mixer
  - [x] Power
  - [x] Display
  - [x] Notification
- [x] System Monitor
  - [x] CPU
  - [x] Memory
  - [x] Network download/upload
  - [x] Disk usage
  - [x] Battery
- [x] Launcher
<!-- - [ ] Clipboard (`:ch`, `:clipboard`)
- [ ] App (`:app`, none)
- [ ] Emoji Picker (`:emoji`, `:e`)
- [ ] URL (`:url https://...`)
- [ ] Search (`:g google-search`, `:b bing-search`)
- [ ] Calculator (`:cal 10 -20`)
- [ ] Note Search (`:notes`)
- [ ] Todo Search (`:todo`)
- [ ] Bookmarks (`:bk`, `bookmarks`)
- [ ] Wallpaper (`:wp`, `wallpapers`)
- [ ] File Search (`:file`, `:f`)
- [ ] Packages (`:pkg pkg-name`, `:p`)
- [ ] Weather (`:weather`, `:w`)
- [ ] Dictionary (`:dict word`, `:d`)
- [ ] Translate (`:trans text`, `:t`)
- [ ] Terminal (`:term termianl-app`, `:terminal`)
- [ ] Command (`:cmd command`)
- [ ] Widgets (`:w bar`)
- [ ] System info (`:sysinfo key`) -->
- [x] Screenshot Utils
  - [x] Screen Recorder
  - [x] Screenshot
- [x] Lockscreen <!-- just overlay over hyprlock or anyother -->
- [x] Shelf <!-- quick links -->
  - [x] Landing
  - [x] Bookmarks
  - [x] Wallpapers
  - [x] Save workspaces
  - [x] Utilities
  - [x] Pinned apps
  - [x] About

<!--
- [ ] Wallpaper
- [ ] Wallpaper Manager
      Use swww -->

# Folder structure

niv/
├── cmd/
│ └── niv/
│ └── main.go # CLI entrypoint
│
├── internal/
│ ├── cli/
│ │ ├── open.go
│ │ ├── toggle.go
│ │ ├── reload.go
│ │ ├── kill.go
│ │ ├── config.go
│ │ ├── db.go
│ │ ├── internals.go # routes `niv internals <name> --options`
│ │ ├── start.go
│ │ └── utils.go
│ │
│ ├── eww/
│ │ ├── runner.go # starts/stops widgets
│ │ └── ipc.go # IPC with EWW
│ │
│ ├── services/
│ │ ├── media/
│ │ │ └── media.go # music control / media info
│ │ ├── icon/
│ │ │ └── icon.go # wifi/volume/bluetooth icons
│ │ ├── cava/
│ │ │ └── cava.go # audio visualization
│ │ ├── network/
│ │ │ └── network.go
│ │ ├── battery/
│ │ │ └── battery.go
│ │ ├── info/
│ │ │ └── info.go
│ │ ├── wallpaper/
│ │ │ └── wallpaper.go
│ │ └── ...
│ │
│ ├── backends/
│ │ ├── workspace/
│ │ │ ├── workspace-bar.go
│ │ │ └── workspace-shelf.go
│ │ ├── notification
│ │ │ ├── history.go
│ │ │ └── notification.go
│ │ ├── clearer/
│ │ │ └── clearer.go
│ │ ├── network-qs/
│ │ │ └── network.go
│ │ ├── volume/
│ │ │ └── volume.go
│ │ ├── bluetooth/
│ │ │ └── bluetooth.go
│ │ ├── display/
│ │ │ └── display.go
│ │ ├── cpu/
│ │ │ └── cpu.go
│ │ ├── memory/
│ │ │ └── memory.go
│ │ ├── search/
│ │ │ └── search.go # search apps/files/emoji/wallpaper
│ │ ├── calculator/
│ │ │ └── calculator.go
│ │ ├── url/
│ │ │ └── url.go
│ │ ├── dictionary/
│ │ │ └── dictionary.go
│ │ └── translator/
│ │ └── translator.go
│ │
│ ├── internals/
│ │ ├── audio.go # music-control, device-info, volume-ctl
│ │ ├── display.go # brightness, nightlight
│ │ ├── notification.go # history, clear
│ │ ├── hardware.go # cpu/memory/battery/network info
│ │ └── about.go # host & user software info
│ │
│ ├── config/
│ │ ├── config.go
│ │ ├── defaults.go
│ │ └── paths.go
│ │
│ ├── db/
│ │ ├── db.go
│ │ ├── schema.go
│ │ └── migrations.go
│ │
│ └── startup/
│ └── startup.go # logic for `niv start`
│
├── pkg/ # optional: reusable public packages
│
├── scripts/
│ └── install.sh # copies EWW + configs + db
│
├── eww/ # frontend (XDG_CONFIG_HOME/eww)
│ ├── widgets/
│ └── style/
│ └── components/
│
├── configs/
│ └── niv/
│ ├── config.yml
│ └── default.db
│
├── go.mod
├── go.sum
└── README.md
