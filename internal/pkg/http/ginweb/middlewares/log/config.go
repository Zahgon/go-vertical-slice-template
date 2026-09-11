package log

import (
	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/http/ginweb/webcore"
)

type config struct {
	Skipper webcore.Skipper
}

type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

func WithSkipper(skipper webcore.Skipper) Option {
	return optionFunc(func(c *config) {
		c.Skipper = skipper
	})
}
