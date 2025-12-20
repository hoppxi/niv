package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var (
	once sync.Once
	v    *viper.Viper
)

type ConfigManager struct{}

var Config = &ConfigManager{}

func (c *ConfigManager) Load() *viper.Viper {
	once.Do(func() {
		v = viper.New()

		configDir, err := os.UserConfigDir()
		if err != nil {
			panic(err)
		}

		confPath := filepath.Join(configDir, "eww", "wigo.yaml")

		v.SetConfigFile(confPath)
		v.SetConfigType("yaml")

		if err := v.ReadInConfig(); err != nil {
			panic(fmt.Errorf("failed to read config: %w", err))
		}
	})

	return v
}

func (c *ConfigManager) Watch(onChange func()) {
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		onChange()
	})
}
