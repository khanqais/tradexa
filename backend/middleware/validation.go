package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ValidateParamInt creates a middleware that validates whether a given URL path parameter
// is a valid unsigned integer. It helps prevent issues with invalid IDs being passed to queries.
func ValidateParamInt(paramName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		val := c.Param(paramName)
		if val == "" {
			c.Next()
			return
		}

		_, err := strconv.ParseUint(val, 10, 32)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "invalid parameter: " + paramName + " must be a positive integer",
			})
			return
		}
		c.Next()
	}
}
