package http

// Config holds HTTP connection pool configuration
type Config struct {
	MaxIdleConns          int  `env:"HTTP_MAX_IDLE_CONNS" envDefault:"10"`
	MaxIdleConnsPerHost   int  `env:"HTTP_MAX_IDLE_CONNS_PER_HOST" envDefault:"2"`
	MaxConnsPerHost       int  `env:"HTTP_MAX_CONNS_PER_HOST" envDefault:"10"`
	IdleConnTimeout       int  `env:"HTTP_IDLE_CONN_TIMEOUT" envDefault:"90"`       // seconds
	ResponseHeaderTimeout int  `env:"HTTP_RESPONSE_HEADER_TIMEOUT" envDefault:"30"` // seconds
	Timeout               int  `env:"HTTP_TIMEOUT" envDefault:"30"`                 // seconds
	DisableCompression    bool `env:"HTTP_DISABLE_COMPRESSION" envDefault:"false"`
	InsecureSkipVerify    bool `env:"HTTP_INSECURE_SKIP_VERIFY" envDefault:"true"`
}

// DefaultConfig returns default HTTP connection pool configuration
func DefaultConfig() *Config {
	return &Config{
		MaxIdleConns:          10,
		MaxIdleConnsPerHost:   2,
		MaxConnsPerHost:       10,
		IdleConnTimeout:       90,
		ResponseHeaderTimeout: 30,
		Timeout:               30,
		DisableCompression:    false,
		InsecureSkipVerify:    true,
	}
}
