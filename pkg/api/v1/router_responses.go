package metadataservice

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"text/template"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/volatiletech/sqlboiler/v4/types"
	"go.uber.org/zap"
)

// ErrorResponse represents an error response record
type ErrorResponse struct {
	Message string   `json:"message,omitempty"`
	Errors  []string `json:"errors,omitempty"`
}

func dbErrorResponse(logger *zap.Logger, c *gin.Context, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		notFoundResponse(c)
	} else {
		logger.Error("database error", zap.Error(err))

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

func invalidUUIDResponse(c *gin.Context, err error) {
	if err != nil {
		if errors.Is(err, ErrInvalidUUID) {
			c.Error(err) //nolint:errcheck // error response is not needed
		}

		notFoundResponse(c)
	}

	c.AbortWithStatusJSON(http.StatusInternalServerError, &ErrorResponse{Errors: []string{"internal server error"}})
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

// addTemplateFields will unmarshal the raw JSON and attempt to augment it with
// the configured template fields.
// If an error occurs unmarshalling the json, or an error occurs while
// executing a template, we'll just return nil, err.
func addTemplateFields(metadata types.JSON, templateFields map[string]template.Template) (map[string]interface{}, error) {
	// Attempt to unmarshal the stored json for the instance.
	resp := make(map[string]interface{})
	err := json.Unmarshal(metadata, &resp)

	if err != nil {
		return nil, err
	}

	// Now that we've unmarshaled the raw json message, augment it with the templated fields
	for k, v := range templateFields {
		// If the metadata already has a field with a matching name, just use what was provided.
		if _, ok := resp[k]; ok {
			continue
		}

		templateBuf := new(bytes.Buffer)

		err = v.Execute(templateBuf, resp)
		if err != nil {
			return nil, err
		}

		resp[k] = templateBuf.String()
	}

	return resp, nil
}
