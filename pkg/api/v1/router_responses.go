package metadataservice

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ErrorResponse represents an error response record
type ErrorResponse struct {
	Message string   `json:"message,omitempty"`
	Errors  []string `json:"errors,omitempty"`
}

func dbErrorResponse(c *gin.Context, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		notFoundResponse(c)
	} else {
		c.JSON(http.StatusInternalServerError, &ErrorResponse{Errors: []string{"internal server error"}})
	}
}

func notFoundResponse(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusNotFound, &ErrorResponse{Message: "resource not found"})
}

func badRequestResponse(c *gin.Context, message string, err error) {
	var errMsgs []string
	if err != nil {
		errMsgs = getErrorMessagesFromError(err)
	}

	_ = c.Error(err)

	c.AbortWithStatusJSON(http.StatusBadRequest, &ErrorResponse{Message: message, Errors: errMsgs})
}

func getErrorMessagesFromError(err error) []string {
	if err == nil {
		return []string{}
	}

	var errMsgs []string

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			errMsgs = append(errMsgs, getErrorMessageFromError(e))
		}
	}

	return errMsgs
}

func getErrorMessageFromError(err error) string {
	if err == nil {
		return ""
	}

	var errMsg string
	if fieldError, ok := err.(validator.FieldError); ok {
		errMsg = fmt.Sprintf("validation failed on %s, condition: %s", fieldError.Field(), fieldError.Tag())
	} else {
		errMsg = ""
	}

	return errMsg
}
