package migration

import (
	"database/sql"
	"embed"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/pressly/goose/v3"
	"github.com/urfave/cli/v2"
)

type Migration struct {
	migrationDir      string
	db                *sql.DB
	commandNamePrefix string
}

func New(db *sql.DB, fs embed.FS, migrationDir string, opts ...Option) (*Migration, error) {
	if db == nil {
		return nil, errors.New("db is nil")
	}

	goose.SetBaseFS(fs)

	m := &Migration{
		migrationDir: migrationDir,
		db:           db,
	}

	for _, opt := range opts {
		opt(m)
	}

	return m, nil
}

func (m *Migration) Up() *cli.Command {
	return &cli.Command{
		Name:  m.name("migrate-up"),
		Usage: "run database migrations",
		Action: func(ctx *cli.Context) error {
			err := goose.Up(m.db, m.migrationDir)
			return errors.Wrap(err, "failed to up migrations")
		},
	}
}

func (m *Migration) UpTo() *cli.Command {
	return &cli.Command{
		Name:  m.name("migrate-up-to"),
		Usage: "database migrations up to version",
		Action: func(ctx *cli.Context) error {
			version, err := strconv.ParseInt(ctx.Args().Get(0), 10, 64)
			if err != nil {
				return errors.Wrap(err, "version is not int")
			}
			err = goose.UpTo(m.db, m.migrationDir, version)
			return errors.Wrap(err, "failed to 'up to' migrations")
		},
	}
}

func (m *Migration) Down() *cli.Command {
	return &cli.Command{
		Name:  m.name("migrate-down"),
		Usage: "rollback database migration",
		Action: func(ctx *cli.Context) error {
			err := goose.Down(m.db, m.migrationDir)
			return errors.Wrap(err, "failed to down migrations")
		},
	}
}
func (m *Migration) DownTo() *cli.Command {
	return &cli.Command{
		Name:  m.name("migrate-down-to"),
		Usage: "rollback database migrations down to version",
		Action: func(ctx *cli.Context) error {
			version, err := strconv.ParseInt(ctx.Args().Get(0), 10, 64)
			if err != nil {
				return errors.Wrap(err, "version is not int")
			}
			err = goose.DownTo(m.db, m.migrationDir, version)
			return errors.Wrap(err, "failed to 'down to' migrations")
		},
	}
}

func (m *Migration) To() *cli.Command {
	return &cli.Command{
		Name:  m.name("migrate-to"),
		Usage: "rollback database migrations down to version",
		Action: func(ctx *cli.Context) error {
			version, err := strconv.ParseInt(ctx.Args().Get(0), 10, 64)
			if err != nil {
				return errors.Wrap(err, "version is not int")
			}
			err = goose.UpTo(m.db, m.migrationDir, version)
			if err != nil {
				return errors.Wrap(err, "failed to 'down to' migrations")
			}
			err = goose.DownTo(m.db, m.migrationDir, version)
			return errors.Wrap(err, "failed to 'down to' migrations")
		},
	}
}

func (m *Migration) DownToLastSequentVersion() *cli.Command {
	return &cli.Command{
		Name:  m.name("migrate-down-to-sequent"),
		Usage: "rollback database migrations down to last sequent version",
		Action: func(ctx *cli.Context) error {
			version, err := m.getLastSeqVersion()
			if err != nil {
				return errors.Wrap(err, "failed to fetch last version")
			}
			err = goose.DownTo(m.db, m.migrationDir, version)
			return errors.Wrap(err, "failed to 'down to' migrations")
		},
	}
}

func (m *Migration) name(s string) string {
	if m.commandNamePrefix == "" {
		return s
	}
	return fmt.Sprintf("%s-%s", m.commandNamePrefix, s)
}

func (m *Migration) getLastSeqVersion() (int64, error) {
	migrations, err := goose.CollectMigrations(m.migrationDir, 0, goose.MaxVersion)
	if err != nil {
		return -1, errors.Wrap(err, "failed to fetch migrations")
	}

	var migration *goose.Migration
	migration, err = migrations.Last()
	if err != nil {
		return -1, errors.Wrap(err, "failed to fetch last migration")
	}

	const timestampMask = 20000000000000
	for {
		version := migration.Version
		if version < timestampMask {
			return version, nil
		}

		migration, err = migrations.Previous(version)
		if err != nil {
			return -1, errors.Wrap(err, "failed to fetch previous migration")
		}
	}
}
