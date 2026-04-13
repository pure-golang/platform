# Usage in project
```go
package main

import (
	"embed"
	"github.com/pure-golang/platform/migration"
	"log"
)

//go:embed migrations/*.sql
var fs embed.FS

func main() {
	if err := migration.DefaultPGMigrate(fs);err != nil {
		log.Fatal("failed to migrate: " + err.Error())
	}
}
```