package grpc

// Config holds gRPC connection pool configuration
type Config struct {
	Address            string `env:"GRPC_ADDRESS" envDefault:"localhost:5001"`
	MaxConns           int    `env:"GRPC_MAX_CONNS" envDefault:"10"`
	MaxIdleConns       int    `env:"GRPC_MAX_IDLE_CONNS" envDefault:"2"`
	MaxConnAge         int    `env:"GRPC_MAX_CONN_AGE" envDefault:"300"`          // seconds
	MaxConnIdleTime    int    `env:"GRPC_MAX_CONN_IDLE_TIME" envDefault:"120"`    // seconds
	ConnectTimeout     int    `env:"GRPC_CONNECT_TIMEOUT" envDefault:"10"`        // seconds
	Timeout            int    `env:"GRPC_TIMEOUT" envDefault:"30"`                // seconds
	MaxRecvMsgSize     int    `env:"GRPC_MAX_RECV_MSG_SIZE" envDefault:"4194304"` // 4MB
	MaxSendMsgSize     int    `env:"GRPC_MAX_SEND_MSG_SIZE" envDefault:"4194304"` // 4MB
	EnableTLS          bool   `env:"GRPC_ENABLE_TLS" envDefault:"false"`
	InsecureSkipVerify bool   `env:"GRPC_INSECURE_SKIP_VERIFY" envDefault:"true"`
}

// DefaultConfig returns default gRPC connection pool configuration
func DefaultConfig() *Config {
	return &Config{
		Address:            "localhost:5001",
		MaxConns:           10,
		MaxIdleConns:       2,
		MaxConnAge:         300,
		MaxConnIdleTime:    120,
		ConnectTimeout:     10,
		Timeout:            30,
		MaxRecvMsgSize:     4194304,
		MaxSendMsgSize:     4194304,
		EnableTLS:          false,
		InsecureSkipVerify: true,
	}
}
