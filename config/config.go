package config

import (
	"flag"
	"os"
)

type Config struct {
	RunAddress           string
	AccrualSystemAddress string
	DBConnStr            string
	LogLevel             string
}

// Configure priority: 1. Environment. 2. Flags
func (c *Config) Configure() {
	//env
	runAddr, okRunAddr := os.LookupEnv("RUN_ADDRESS")
	dbStr, okdbStr := os.LookupEnv("DATABASE_URI")
	accrSysAddr, okAccrSysAddr := os.LookupEnv("ACCRUAL_SYSTEM_ADDRESS")
	logLevel, okLogLevel := os.LookupEnv("LOG_LEVEL")

	//flags
	if !okRunAddr {
		flag.StringVar(&c.RunAddress, "a", "", "Server will run on this address and port")
	} else {
		c.RunAddress = runAddr
	}

	if !okdbStr {
		flag.StringVar(&c.DBConnStr, "d", "", "Db conn str")
	} else {
		c.DBConnStr = dbStr
	}

	if !okAccrSysAddr {
		flag.StringVar(&c.AccrualSystemAddress, "r", "", "Accrual system address")
	} else {
		c.AccrualSystemAddress = accrSysAddr
	}

	if !okLogLevel {
		flag.StringVar(&c.LogLevel, "l", "", "Log level")
	} else {
		c.LogLevel = logLevel
	}
}
