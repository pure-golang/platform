package migration

type Option func(m *Migration)

func WithPrefix(prefix string) Option {
	return func(m *Migration) {
		m.commandNamePrefix = prefix
	}
}
