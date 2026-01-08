package common_mysql

// MysqlConfig defines MySQL connection configuration.
// This is a local copy of the fields needed from server/config.MysqlConfig
// to avoid pulling in heavy dependencies (AWS SDK, viper, etc.).
type MysqlConfig struct {
	Protocol        string
	Address         string
	Username        string
	Password        string
	PasswordPath    string
	Database        string
	TLSCert         string
	TLSKey          string
	TLSCA           string
	TLSServerName   string
	TLSConfig       string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime int
	SQLMode         string
	Region          string
}

// LoggingConfig defines logging configuration.
// This is a local copy of the fields needed from server/config.LoggingConfig.
type LoggingConfig struct {
	TracingEnabled bool
	TracingType    string
}
