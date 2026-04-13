package migration

import (
	"embed"
	"os"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pkg/errors"
	"github.com/pure-golang/adapters/db/pg/pgx"
	"github.com/pure-golang/adapters/env"
	"github.com/urfave/cli/v2"
)

const migrationDir = "migrations"

func DefaultPGMigrate(fs embed.FS) error {
	conf := new(pgx.Config)
	err := env.InitConfig(conf)
	if err != nil {
		return errors.Wrapf(err, "failed to init env")
	}

	pool, err := pgx.NewDefault(*conf)
	if err != nil {
		return errors.Wrapf(err, "failed to init db")
	}

	migrate, err := New(stdlib.OpenDBFromPool(pool.Pool), fs, migrationDir)
	if err != nil {
		return errors.Wrapf(err, "failed create migrator")
	}

	app := &cli.App{
		Usage: "Default pg migrator",
		Commands: cli.Commands{
			migrate.Up(),
			migrate.UpTo(),
			migrate.Down(),
			migrate.DownTo(),
			migrate.To(),
			migrate.DownToLastSequentVersion(),
		},
	}

	return errors.Wrap(app.Run(os.Args), "failed to run app")
}
