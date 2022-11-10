package config

import (
	"go.infratographer.com/x/crdbx"
	"go.infratographer.com/x/loggingx"
	"go.infratographer.com/x/otelx"
)

// AppConfig represents application-wide config options
var AppConfig struct {
	CRDB    crdbx.Config
	Logging loggingx.Config
	Tracing otelx.Config
}
