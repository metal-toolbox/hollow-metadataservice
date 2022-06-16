package metadataservice

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"go.hollow.sh/metadataservice/internal/middleware"
	"go.hollow.sh/metadataservice/internal/models"
)

func (r *Router) instanceUserdataGet(c *gin.Context) {
	instanceID := c.GetString(middleware.ContextKeyInstanceID)
	if instanceID == "" {
		// TODO: Try to fetch the userdata from an external source of truth.
		// Return 404 for now...
		notFoundResponse(c)
		return
	}

	userdata, err := models.FindInstanceUserdatum(c.Request.Context(), r.DB, instanceID)

	if err != nil {
		dbErrorResponse(c, err)
		return
	}

	c.String(http.StatusOK, string(userdata.Userdata.Bytes))
}
