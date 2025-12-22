# Wigo

wigo is a widget system written in Go.
It currently uses eee as its frontend, with plans to migrate to a pure **gotk4-layer-shell** or **gotk3-layer-shell** implementation in the future. Wigo acts as the main controller and wrapper around Eww.

## Showcase

<p align="center">
  <img src=".github/2025-12-20_19-10.png" width="30%" />
  <img src=".github/2025-12-20_19-11.png" width="30%" />
  <img src=".github/2025-12-20_19-11_1.png" width="30%" />
</p>

## Installation

### NixOS

```sh
nix develop

wigo setup
wigo start &!
```

To add Wigo as an input:

```nix
inputs.wigo.url = "github:hoppxi/wigo";

# Then in your configuration:
{
  imports = [
    inputs.wigo.homeModules.wigo
  ];

  programs.wigo = {
    enable = true;
    notification = true;

    settings = {
      apps = {
        terminal = "alacritty";
        editor = "code";
        system_monitor = "alacritty --hold -e btop";
        file_manager = "nautilus";
        system_info = "alacritty --hold -e neofetch";
        screenshot_tool = "flameshot launcher";
      };

      general = {
        display_name = "Ermiyas";
        top_left_icon = "distributor-logo-nixos";
        profile_pic = "";
      };

      wallpapers_path = "/home/hoppxi/Pictures/Wallpapers";
      
      # example
      launcher-ext = [
        {
          name = "Wallpapers";
          trigger = ":wallpapers";
          trigger_short = ":wp";
          from_folder = "~/Pictures/Wallpapers";
          exclude_type = "webp";
          on_select = "wigo wallpaper --set {}";
          help_text = "Quickly set your desktop background";
          limit = 20;
        }
      ];
    };
  };
}

```


### Non-NixOS

If you are **not** on NixOS, make sure you have the following installed:

* Hyprland
* eww
* cliphist
* pactl
* wl-clipboard
* zenity
* hyprsunset

Then proceed with the installation:

```sh
git clone git@github.com:hoppxi/wigo.git
cd wigo

chmod +x ./scripts/build.sh
./scripts/build.sh

# run the setup
wigo setup

# run wigo
wigo start &!          # start widgets
wigo notification &!  # start notification daemon (make sure no other daemon is running)

wigo help              # for further exploration
```

## Features

Wigo includes many built-in components, especially in the launcher:

* Emoji picker
* Clipboard manager
* Extension support
* App launcher
* Binary launcher
* Google search
* YouTube search
* Translation
* And more

You can find example configuration files at: **config/wigo.yaml**
