package log

// WithName sets logger name
func WithName(name string) Option {
	return optionFunc(func(l *logger) {
		l.name = name
	})
}

// WithLevel sets level for the logger
func WithLevel(level Level) Option {
	return optionFunc(func(l *logger) {
		l.level = level
	})
}

// WithFormat sets log Formatter
func WithFormat(format Format) Option {
	return optionFunc(func(l *logger) {
		l.format = format
	})
}

// WithTags sets logger's tags
func WithTags(tags map[string]string) Option {
	return optionFunc(func(l *logger) {
		l.tags = tags
	})
}

// WithRollbar enables critical logging to rollbar
func WithRollbar(token string, minLevel Level) Option {
	return optionFunc(func(l *logger) {
		l.rollbarToken = token
		l.rollbarMinLevel = minLevel
	})
}
