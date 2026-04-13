# Monitoring

Модуль для настройки мониторинга с поддержкой OpenTelemetry и Prometheus метрик.

## Использование

```go
package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"github.com/pure-golang/platform/monitoring"
)

var version string     // -ldflags "-X main.version=${VERSION}"
var serviceName string // -ldflags "-X main.serviceName=${SERVICE_NAME}"

func main() {
	c := new(monitoring.Config)
	closeMonitoring := monitoring.InitDefault(*c, serviceName, version)
	defer func() {
		if err := closeMonitoring(); err != nil {
			slog.Default().Error("failed to close monitoring: ", err.Error())
		}
	}()

	// Ваш код приложения

	// Graceful shutdown
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
}
```

