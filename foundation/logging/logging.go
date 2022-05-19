// Package logging is responsible for creating and configuring a zap
// logging with a uniform policy. The foundation package is usually
// where packages live while they are maturing to be moved to the
// organization's gokit or an opensource project. In this case since
// it is organization logging policy it would be the gokit.
package logging

import (
	"github.com/rsb/failure"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(service, version string) (*zap.SugaredLogger, error) {
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{"stdout"}
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.DisableStacktrace = true
	config.InitialFields = map[string]interface{}{
		"service":         service,
		"service-version": version,
	}

	log, err := config.Build()
	if err != nil {
		return nil, failure.ToConfig(err, "[zap.NewProductionConfig()] config.Build failed")
	}

	return log.Sugar(), nil
}
