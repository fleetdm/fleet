package app

import "github.com/spf13/viper"

func init() {
	setDefaultConfigValue := func(key string, value interface{}) {
		if viper.Get(key) == nil {
			viper.Set(key, value)
		}
	}
	setDefaultConfigValue("auth.bcrypt_cost", 12)
	setDefaultConfigValue("auth.salt_key_size", 24)

	setDefaultConfigValue("session.key_size", 64)
	setDefaultConfigValue("session.expiration_seconds", 60*60*24*90)

	setDefaultConfigValue("osquery.node_key_size", 24)
	setDefaultConfigValue("osquery.enroll_secret", "super secret")
}
