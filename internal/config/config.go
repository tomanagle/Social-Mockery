package config

import (
	"fmt"
	"net"
)

type Config struct {
	Addr, Database, SeedPath, AdminToken string
	MaxRecords                           int
}

func Default() Config { return Config{Addr: "127.0.0.1:3000", Database: ":memory:", MaxRecords: 10000} }
func (c Config) Validate() error {
	if _, _, err := net.SplitHostPort(c.Addr); err != nil {
		return err
	}
	if c.Database == "" {
		return fmt.Errorf("database must not be empty")
	}
	if c.MaxRecords < 2 || c.MaxRecords > 1000000 {
		return fmt.Errorf("max-records must be 2..1000000")
	}
	return nil
}

// Must validates required startup configuration and panics on invalid values.
func Must(c Config) Config {
	if err := c.Validate(); err != nil {
		panic("config: initialization failed")
	}
	return c
}
