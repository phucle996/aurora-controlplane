package config

import "time"

// Config is the root typed config object, loaded once at startup.
type Config struct {
	App        AppCfg
	Psql       PsqlCfg
	Redis      RedisCfg
	VictoriaDB VictoriaDBCfg
	GRPC       GRPCCfg
}

// AppCfg holds application-level settings.
type AppCfg struct {
	TimeZone string
	HTTPPort string
	LogLV    string
}

// PsqlCfg holds PostgreSQL connection parameters.
type PsqlCfg struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string

	// TLS
	TLSEnabled bool
	CACertPath string
	CertPath   string
	KeyPath    string

	// Pool
	MaxConns     int
	MinConns     int
	MaxConnLife  time.Duration
	MaxConnIdle  time.Duration

	// Connection behavior
	PingTimeout   time.Duration
	MaxRetries    int
	RetryInterval time.Duration
}

// RedisCfg holds Redis connection parameters for both cache and stream usage.
type RedisCfg struct {
	Addr     string
	Password string
	DB       int

	// TLS
	TLSEnabled bool
	CACertPath string
	CertPath   string
	KeyPath    string

	// Timeouts
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	// Pool
	PoolSize     int
	MinIdleConns int

	// Connection behavior
	PingTimeout   time.Duration
	MaxRetries    int
	RetryInterval time.Duration
}

// VictoriaDBCfg holds VictoriaMetrics connection parameters.
type VictoriaDBCfg struct {
	WriteURL string // e.g. http://victoria:8428/api/v1/write
	ReadURL  string // e.g. http://victoria:8428/api/v1/query

	// Auth (optional, for enterprise/proxy)
	Username string
	Password string

	// Timeouts
	WriteTimeout time.Duration
	ReadTimeout  time.Duration

	// Connection behavior
	PingTimeout   time.Duration
	MaxRetries    int
	RetryInterval time.Duration
}

// GRPCCfg holds gRPC server and client settings.
type GRPCCfg struct {
	ServerPort string

	// Client TLS
	ClientTLSEnabled bool
	ClientCACertPath string
	ClientCertPath   string
	ClientKeyPath    string
}

// LoadConfig reads environment variables and returns the root typed config.
func LoadConfig() *Config {
	return &Config{
		App: AppCfg{
			TimeZone: getEnv("APP_TIMEZONE", "UTC"),
			HTTPPort: getEnv("APP_HTTP_PORT", "8080"),
			LogLV:    getEnv("APP_LOG_LEVEL", "info"),
		},
		Psql: PsqlCfg{
			Host:          getEnv("PSQL_HOST", "localhost"),
			Port:          getEnvAsInt("PSQL_PORT", 5432),
			User:          getEnv("PSQL_USER", "postgres"),
			Password:      getEnv("PSQL_PASSWORD", ""),
			DBName:        getEnv("PSQL_DBNAME", "controlplane"),
			SSLMode:       getEnv("PSQL_SSLMODE", "disable"),
			TLSEnabled:    getEnvAsBool("PSQL_TLS_ENABLED", false),
			CACertPath:    getEnv("PSQL_TLS_CA", ""),
			CertPath:      getEnv("PSQL_TLS_CERT", ""),
			KeyPath:       getEnv("PSQL_TLS_KEY", ""),
			MaxConns:      getEnvAsInt("PSQL_MAX_CONNS", 20),
			MinConns:      getEnvAsInt("PSQL_MIN_CONNS", 5),
			MaxConnLife:   getEnvAsDuration("PSQL_MAX_CONN_LIFE", 30*time.Minute),
			MaxConnIdle:   getEnvAsDuration("PSQL_MAX_CONN_IDLE", 5*time.Minute),
			PingTimeout:   getEnvAsDuration("PSQL_PING_TIMEOUT", 5*time.Second),
			MaxRetries:    getEnvAsInt("PSQL_MAX_RETRIES", 5),
			RetryInterval: getEnvAsDuration("PSQL_RETRY_INTERVAL", 3*time.Second),
		},
		Redis: RedisCfg{
			Addr:          getEnv("REDIS_ADDR", "localhost:6379"),
			Password:      getEnv("REDIS_PASSWORD", ""),
			DB:            getEnvAsInt("REDIS_DB", 0),
			TLSEnabled:    getEnvAsBool("REDIS_TLS_ENABLED", false),
			CACertPath:    getEnv("REDIS_TLS_CA", ""),
			CertPath:      getEnv("REDIS_TLS_CERT", ""),
			KeyPath:       getEnv("REDIS_TLS_KEY", ""),
			DialTimeout:   getEnvAsDuration("REDIS_DIAL_TIMEOUT", 5*time.Second),
			ReadTimeout:   getEnvAsDuration("REDIS_READ_TIMEOUT", 3*time.Second),
			WriteTimeout:  getEnvAsDuration("REDIS_WRITE_TIMEOUT", 3*time.Second),
			PoolSize:      getEnvAsInt("REDIS_POOL_SIZE", 20),
			MinIdleConns:  getEnvAsInt("REDIS_MIN_IDLE_CONNS", 5),
			PingTimeout:   getEnvAsDuration("REDIS_PING_TIMEOUT", 5*time.Second),
			MaxRetries:    getEnvAsInt("REDIS_MAX_RETRIES", 5),
			RetryInterval: getEnvAsDuration("REDIS_RETRY_INTERVAL", 3*time.Second),
		},
		VictoriaDB: VictoriaDBCfg{
			WriteURL:      getEnv("VICTORIA_WRITE_URL", "http://localhost:8428/api/v1/write"),
			ReadURL:       getEnv("VICTORIA_READ_URL", "http://localhost:8428/api/v1/query"),
			Username:      getEnv("VICTORIA_USERNAME", ""),
			Password:      getEnv("VICTORIA_PASSWORD", ""),
			WriteTimeout:  getEnvAsDuration("VICTORIA_WRITE_TIMEOUT", 10*time.Second),
			ReadTimeout:   getEnvAsDuration("VICTORIA_READ_TIMEOUT", 10*time.Second),
			PingTimeout:   getEnvAsDuration("VICTORIA_PING_TIMEOUT", 5*time.Second),
			MaxRetries:    getEnvAsInt("VICTORIA_MAX_RETRIES", 3),
			RetryInterval: getEnvAsDuration("VICTORIA_RETRY_INTERVAL", 2*time.Second),
		},
		GRPC: GRPCCfg{
			ServerPort:       getEnv("GRPC_SERVER_PORT", "9090"),
			ClientTLSEnabled: getEnvAsBool("GRPC_CLIENT_TLS_ENABLED", false),
			ClientCACertPath: getEnv("GRPC_CLIENT_TLS_CA", ""),
			ClientCertPath:   getEnv("GRPC_CLIENT_TLS_CERT", ""),
			ClientKeyPath:    getEnv("GRPC_CLIENT_TLS_KEY", ""),
		},
	}
}
