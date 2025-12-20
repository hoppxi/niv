package watchers

import (
	"github.com/spf13/viper"
)

func ConfigUpdate(v *viper.Viper) {
	updateEww("APPS_CONFIG", v.Get("apps"))
	updateEww("WIDGETS_CONFIG", v.Get("widgets"))
	updateEww("NOTIFICATION_CONFIG", v.Get("notifications"))
	updateEww("DISABLED_NOTIFICATION_CONFIG", v.Get("notifications_disabled"))
	updateEww("GENERAL_CONFIG", v.Get("general"))
}
