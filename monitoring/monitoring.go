package monitoring

import (
	"log/slog"

	"github.com/pkg/errors"
	"github.com/pure-golang/adapters/logger"
	"github.com/pure-golang/adapters/metrics"
	"github.com/pure-golang/adapters/tracing"
	"github.com/pure-golang/adapters/tracing/jaeger"
)

type Config struct {
	Logger  logger.Config
	Tracing jaeger.Config
	Metrics metrics.Config
}

func InitDefault(c Config) func() error {
	logger.InitDefault(c.Logger)

	// logger
	l := slog.Default()

	// tracing
	tp, tracingErr := tracing.Init(jaeger.NewProviderBuilder(c.Tracing))
	if tracingErr != nil {
		l.Warn("tracing init error", "error", tracingErr)
	}

	metricClose, metricErr := metrics.InitDefault(c.Metrics)
	if metricErr != nil {
		l.Warn("metrics init error", "error", metricErr)
	}

	return func() error {
		if tracingErr == nil {
			if err := tp.Close(); err != nil {
				return errors.Wrap(err, "failed to close tracing provider")
			}
		}

		if metricErr == nil {
			if err := metricClose.Close(); err != nil {
				return errors.Wrap(err, "failed to close metrics provider")
			}
		}

		return nil
	}
}
