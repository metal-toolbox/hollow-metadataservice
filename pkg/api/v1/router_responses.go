package metadataservice

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse represents an error response record
type ErrorResponse struct {
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func dbErrorResponse(c *gin.Context, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, &ErrorResponse{Message: "resource not found"})
	} else {
		c.JSON(http.StatusInternalServerError, &ErrorResponse{Error: "internal server error"})
	}
}

func notFoundResponse(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusNotFound, &ErrorResponse{Message: "resource not found"})
}
