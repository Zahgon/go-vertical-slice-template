package problemdetail

import (
	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/http/ginweb/webcore"
	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/http/httperrors/problemdetails"
)

type config struct {
	Skipper       webcore.Skipper
	ProblemParser problemDetails.ErrorParserFunc
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

func WithErrorParser(parser problemDetails.ErrorParserFunc) Option {
	return optionFunc(func(c *config) {
		c.ProblemParser = parser
	})
}
