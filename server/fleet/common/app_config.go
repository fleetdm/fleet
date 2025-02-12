package common

type AppConfig interface {
	AndroidEnabledAndConfigured() bool
	ServerURL() string
}
