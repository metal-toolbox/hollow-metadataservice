package metadataservice

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"go.hollow.sh/metadataservice/internal/middleware"
	"go.hollow.sh/metadataservice/internal/models"
	"go.hollow.sh/metadataservice/pkg/api/v1/ec2"
)

// Current top-level items available:
// instance-id
// hostname
// iqn
// plan
// facility
// tags
// operating-system
// public-keys
// spot
// public-ipv4
// public-ipv6
// local-ipv4

// operating-system items:
// slug
// distro
// version
// license_activation
//   - state
// image_tag

// spot items:
// termination-time

// instanceEc2MetadataGet returns the list of top-level metadata item names
// which can be subsequently queried by the caller.
func (r *Router) instanceEc2MetadataGet(c *gin.Context) {
	instanceID := c.GetString(middleware.ContextKeyInstanceID)
	if instanceID == "" {
		// TODO: Try to fetch the metadata from an external source of truth.
		// Return 404 for now...
		notFoundResponse(c)
		return
	}

	instanceMetadata, err := models.FindInstanceMetadatum(c.Request.Context(), r.DB, instanceID)

	if err != nil {
		dbErrorResponse(c, err)
		return
	}

	var metadata = ec2.Metadata{}

	err = json.Unmarshal([]byte(instanceMetadata.Metadata), &metadata)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, &ErrorResponse{Errors: []string{"Invalid metadata for instance"}})
		return
	}

	c.String(http.StatusOK, strings.Join(metadata.ItemNames(), "\n"))
}

func (r *Router) instanceEc2MetadataItemGet(c *gin.Context) {
	instanceID := c.GetString(middleware.ContextKeyInstanceID)
	if instanceID == "" {
		// TODO: Try to fetch the metadata from an external source of truth.
		// Return 404 for now...
		notFoundResponse(c)
		return
	}

	instanceMetadata, err := models.FindInstanceMetadatum(c.Request.Context(), r.DB, instanceID)

	if err != nil {
		dbErrorResponse(c, err)
		return
	}

	var metadata = ec2.Metadata{}

	err = json.Unmarshal([]byte(instanceMetadata.Metadata), &metadata)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, &ErrorResponse{Errors: []string{"Invalid metadata for instance"}})
		return
	}

	if subPath, ok := c.Params.Get("subpath"); ok {
		if result, ok := metadata.GetItem(subPath); ok {
			c.String(http.StatusOK, strings.Join(result, "\n"))
			return
		}
	}

	// If we're here, that means that either there wasn't a subpath item, or we
	// couldn't find the item in the metadata for the instance. In that case,
	// just return a 404.
	notFoundResponse(c)
}

func (r *Router) instanceEc2UserdataGet(c *gin.Context) {
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
