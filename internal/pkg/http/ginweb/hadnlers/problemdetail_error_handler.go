package handlers

import (
	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/http/httperrors/problemdetails"
	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/logger"

	"emperror.dev/errors"
	"github.com/gin-gonic/gin"
)

func ProblemDetailErrorHandlerFunc(
	err error,
	c *gin.Context,
	logger logger.Logger,
) {
	var problem problemDetails.ProblemDetailErr

	// if error was not problem detail we will convert the error to a problem detail
	if ok := errors.As(err, &problem); !ok {
		problem = problemDetails.ParseError(err)
	}

	if !c.Writer.Written() && problem != nil {
		// `WriteTo` will set `Response status code` to our problem details status
		if _, err := problemDetails.WriteTo(problem, c.Writer); err != nil {
			logger.Error(err)
		}
	}
}
