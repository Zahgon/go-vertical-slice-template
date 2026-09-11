package params

import (
	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"
)

type ProductRouteParams struct {
	Logger        logger.Logger
	ProductsGroup *gin.RouterGroup
	Validator     *validator.Validate
}
