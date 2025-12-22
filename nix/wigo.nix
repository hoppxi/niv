{
  config,
  lib,
  pkgs,
  inputs,
  ...
}:

let
  cfg = config.programs.wigo;
  yaml = pkgs.formats.yaml { };
in
{
  options.programs.wigo = {
    enable = lib.mkEnableOption "Wigo widgets";

    notification = lib.mkOption {
      type = lib.types.bool;
      default = false;
      description = "Run `wigo notification` after startup";
    };

    package = lib.mkOption {
      type = lib.types.package;
      default = inputs.wigo.packages.${pkgs.system}.default;
      description = "Wigo package to use";
    };

    settings = lib.mkOption {
      type = lib.types.submodule {
        options = {
          apps = lib.mkOption {
            type = lib.types.submodule {
              options = {
                terminal = lib.mkOption { type = lib.types.str; };
                editor = lib.mkOption { type = lib.types.str; };
                system_monitor = lib.mkOption { type = lib.types.str; };
                file_manager = lib.mkOption { type = lib.types.str; };
                system_info = lib.mkOption { type = lib.types.str; };
                screenshot_tool = lib.mkOption { type = lib.types.str; };
              };
            };
          };

          general = lib.mkOption {
            type = lib.types.submodule {
              options = {
                display_name = lib.mkOption { type = lib.types.str; };
                top_left_icon = lib.mkOption { type = lib.types.str; };
                profile_pic = lib.mkOption { type = lib.types.path; };
              };
            };
          };

          wallpapers_path = lib.mkOption {
            type = lib.types.path;
          };

          launcher-ext = lib.mkOption {
            type = lib.types.listOf (
              lib.types.submodule {
                options = {
                  # Wallpaper launcher schema
                  name = lib.mkOption {
                    type = lib.types.nullOr lib.types.str;
                    default = null;
                  };
                  trigger = lib.mkOption {
                    type = lib.types.nullOr lib.types.str;
                    default = null;
                  };
                  trigger_short = lib.mkOption {
                    type = lib.types.nullOr lib.types.str;
                    default = null;
                  };
                  from_folder = lib.mkOption {
                    type = lib.types.nullOr lib.types.str;
                    default = null;
                  };
                  exclude_type = lib.mkOption {
                    type = lib.types.nullOr lib.types.str;
                    default = null;
                  };
                  on_select = lib.mkOption {
                    type = lib.types.nullOr lib.types.str;
                    default = null;
                  };
                  limit = lib.mkOption {
                    type = lib.types.nullOr lib.types.int;
                    default = null;
                  };
                  path = lib.mkOption {
                    type = lib.types.nullOr lib.types.path;
                    default = null;
                  };
                  help_text = lib.mkOption {
                    type = lib.types.nullOr lib.types.str;
                    default = null;
                  };
                };
              }
            );
            default = [ ];
            description = ''
              Optional launcher extensions. Each entry must be either:
              1) Wallpaper type: name, trigger, trigger_short, from_folder, exclude_type, on_select, limit
              2) Executable type: name, trigger, trigger_short, path
            '';
          };
        };
      };
    };
  };

  config = lib.mkIf cfg.enable {
    home.packages = [ cfg.package ];

    home.file.".config/eww/wigo.yaml".source = yaml.generate "wigo.yaml" cfg.settings;

    systemd.user.services.wigo = {
      Unit = {
        Description = "Wigo panel";
        After = [ "graphical-session.target" ];
        Wants = [ "graphical-session.target" ];
      };

      Service = {
        Type = "simple";
        ExecStart = "${cfg.package}/bin/wigo start";
        ExecStartPost = if cfg.notification then "${cfg.package}/bin/wigo notification" else null;
        Restart = "on-failure";
        RestartSec = 2;
      };

      Install = {
        WantedBy = [ "graphical-session.target" ];
      };
    };
  };
}
