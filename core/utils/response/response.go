package response

import (
	"net/http"

	commonDto "alerthub/core/dto/common"

	"github.com/gin-gonic/gin"
)

func Success(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, commonDto.APIResponse{
		Status:  true,
		Message: message,
		Data:    data,
	})
}

func Paginated(c *gin.Context, message string, data interface{}, pagination commonDto.PaginationMeta) {
	c.JSON(http.StatusOK, commonDto.PaginatedResponse{
		Status:     true,
		Message:    message,
		Data:       data,
		Pagination: pagination,
	})
}

func Error(c *gin.Context, statusCode int, code string, message string, details interface{}) {
	c.JSON(statusCode, commonDto.ErrorResponse{
		Status:  false,
		Message: message,
		Error: commonDto.ErrorBody{
			Code:    code,
			Details: details,
		},
	})
}
