package metadataservice

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"go.hollow.sh/metadataservice/internal/middleware"
	"go.hollow.sh/metadataservice/internal/models"
)

func (r *Router) instanceMetadataGet(c *gin.Context) {
	instanceID := c.GetString(middleware.ContextKeyInstanceID)
	if instanceID == "" {
		// TODO: Try to fetch the metadata from an external source of truth.
		// Return 404 for now...
		notFoundResponse(c)
		return
	}

	metadata, err := models.FindInstanceMetadatum(c.Request.Context(), r.DB, instanceID)

	if err != nil {
		dbErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusOK, metadata.Metadata)
}
