package endpoints

import (
	"net/http"

	"github.com/mehdihadeli/go-vertical-slice-template/internal/catalogs/products/contracts"
	"github.com/mehdihadeli/go-vertical-slice-template/internal/catalogs/products/contracts/params"
	"github.com/mehdihadeli/go-vertical-slice-template/internal/catalogs/products/features/gettingproductbyid/dtos"
	"github.com/mehdihadeli/go-vertical-slice-template/internal/catalogs/products/features/gettingproductbyid/queries"
	"github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/http/ginweb"
	customErrors "github.com/mehdihadeli/go-vertical-slice-template/internal/pkg/http/httperrors/customerrors"

	"emperror.dev/errors"
	"github.com/gin-gonic/gin"
	"github.com/mehdihadeli/go-mediatr"
)

type getProductByIdEndpoint struct {
	*params.ProductRouteParams
}

func NewGetProductByIdEndpoint(params *params.ProductRouteParams) contracts.Endpoint {
	return &getProductByIdEndpoint{ProductRouteParams: params}
}

func (ep *getProductByIdEndpoint) MapEndpoint() {
	ep.ProductsGroup.GET("/:id", ep.handler())
	ep.ProductsGroup.GET("/:id/*action", ep.handler())
}

// GetProductByID
// @Tags Products
// @Summary Get product
// @Description Get product by id
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Success 200 {object} gettingProductByIdDtos.GetProductByIdResponseDto
// @Router /api/v1/products/{id} [get]
func (ep *getProductByIdEndpoint) handler() gin.HandlerFunc {
	return ginweb.Handler(func(ctx *gin.Context) error {
		ginweb.JoinCatchAll(ctx, "id", "action")

		request := &dtos.GetProductByIdRequestDto{}
		if err := ginweb.BindPathParams(ctx, request); err != nil {
			return customErrors.NewBadRequestErrorWrap(
				err,
				"error in the binding request",
			)
		}

		query := queries.NewGetProductByIdQuery(request.ProductId)

		if err := ep.Validator.StructCtx(ctx.Request.Context(), query); err != nil {
			return customErrors.NewValidationErrorWrap(err, "validation error")
		}

		queryResult, err := mediatr.Send[*queries.GetProductByIdQuery, *dtos.GetProductByIdQueryResponse](
			ctx.Request.Context(),
			query,
		)
		if err != nil {
			return errors.WithMessage(
				err,
				"error in sending GetProductByIdQuery",
			)
		}

		return ginweb.JSON(ctx, http.StatusOK, queryResult)
	})
}
