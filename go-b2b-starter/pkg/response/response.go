package response

import (
	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/pkg/httperr"
)

// Success sends a successful response
func Success(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, gin.H{
		"success": true,
		"data":    data,
	})
}

// Paginated sends a successful paginated response carrying the total count of
// rows matching the request's filters (ignoring limit/offset) alongside the
// page's data array.
func Paginated(c *gin.Context, statusCode int, data interface{}, total int32) {
	c.JSON(statusCode, gin.H{
		"success": true,
		"data":    data,
		"total":   total,
	})
}

// Error sends an error response
func Error(c *gin.Context, statusCode int, message string, err error) {
	c.JSON(statusCode, httperr.NewHTTPError(
		statusCode,
		"error",
		message,
	))
}