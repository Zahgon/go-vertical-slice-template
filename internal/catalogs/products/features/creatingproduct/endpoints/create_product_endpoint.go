package endpoints

import (
	"net/http"

	"github.com/mehdihadeli/go-vertical-slice-template/internal/catalogs/products/contracts"
	"github.com/mehdihadeli/go-vertical-slice-template/internal/catalogs/products/contracts/params"
	"github.com/mehdihadeli/go-vertical-slice-template/internal/catalogs/products/features/creatingproduct/commands"
	"github.com/mehdihadeli/go-vertical-slice-template/internal/catalogs/products/features/creatingproduct/dtos"
	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/http/ginweb"
	customErrors "github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/http/httperrors/customerrors"

	"emperror.dev/errors"
	"github.com/gin-gonic/gin"
	"github.com/mehdihadeli/go-mediatr"
)

type createProductEndpoint struct {
	*params.ProductRouteParams
}

func NewCreteProductEndpoint(params *params.ProductRouteParams) contracts.Endpoint {
	return &createProductEndpoint{ProductRouteParams: params}
}

func (ep *createProductEndpoint) MapEndpoint() {
	ep.ProductsGroup.POST("", ep.handler())
}

// CreateProduct
// @Tags Products
// @Summary Create product
// @Description Create new product item
// @Accept json
// @Produce json
// @Param CreateProductRequestDto body creatingProductsDtos.CreateProductRequestDto true "Product data"
// @Success 201 {object} creatingProductsDtos.CreateProductResponseDto
// @Router /api/v1/products [post]
func (ep *createProductEndpoint) handler() gin.HandlerFunc {
	return ginweb.Handler(func(ctx *gin.Context) error {
		request := &dtos.CreateProductRequestDto{}
		if err := ginweb.BindBody(ctx, request); err != nil {
			return customErrors.NewBadRequestErrorWrap(
				err,
				"error in the binding request",
			)
		}

		if err := ep.Validator.StructCtx(ctx.Request.Context(), request); err != nil {
			return customErrors.NewValidationErrorWrap(err, "validation error")
		}

		command := commands.NewCreateProductCommand(request.Name, request.Description, request.Price)
		result, err := mediatr.Send[*commands.CreateProductCommand, *dtos.CreateProductCommandResponse](
			ctx.Request.Context(),
			command,
		)
		if err != nil {
			return errors.WithMessage(
				err,
				"error in sending CreateProductCommand",
			)
		}

		return ginweb.JSON(ctx, http.StatusCreated, result)
	})
}
