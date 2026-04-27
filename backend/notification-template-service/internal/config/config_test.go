package config

import (
	"testing"
)

func TestConfig_DatabaseDSN(t *testing.T) {
	t.Parallel()

	// Arrange
	cfg := &Config{
		DatabaseHost:     "localhost",
		DatabasePort:     5432,
		DatabaseUser:     "user",
		DatabasePassword: "pass",
		DatabaseName:     "dbname",
		DatabaseSSLMode:  "disable",
	}

	// Act
	dsn := cfg.DatabaseDSN()

	// Assert
	expected := "host=localhost port=5432 user=user password=pass dbname=dbname sslmode=disable"
	if dsn != expected {
		t.Errorf("DatabaseDSN() = %q, want %q", dsn, expected)
	}
}

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				ServerAddress: ":50051",
				DatabaseHost:  "localhost",
				DatabaseName:  "notifications",
			},
			wantErr: false,
		},
		{
			name: "empty server address",
			cfg: Config{
				ServerAddress: "",
				DatabaseHost:  "localhost",
				DatabaseName:  "notifications",
			},
			wantErr: true,
		},
		{
			name: "empty database host",
			cfg: Config{
				ServerAddress: ":50051",
				DatabaseHost:  "",
				DatabaseName:  "notifications",
			},
			wantErr: true,
		},
		{
			name: "empty database name",
			cfg: Config{
				ServerAddress: ":50051",
				DatabaseHost:  "localhost",
				DatabaseName:  "",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Act
			err := tc.cfg.Validate()

			// Assert
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
