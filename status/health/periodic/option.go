package periodic

type Option interface {
	apply(source *healthCheckSource)
}

type optionFn func(source *healthCheckSource)

func (fn optionFn) apply(source *healthCheckSource) {
	fn(source)
}

func WithInitialPoll() Option {
	return optionFn(func(source *healthCheckSource) {
		source.initialPoll = true
	})
}
