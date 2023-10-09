package logwhale

import "fmt"

// WithOptions applies the given options to the LogManager
func (lm *LogManager) withOptions(opts ...Option) error {
	for _, opt := range opts {
		err := opt(lm)
		if err != nil {
			return fmt.Errorf("cannot apply Option: %w", err)
		}
	}
	return nil
}

func WithBufferSize(bs int) Option {
	return func(lm *LogManager) error {
		if bs <= 0 {
			return fmt.Errorf("buffer size must be greater than 0")
		}
		lm.bufferSize = bs
		return nil
	}
}
