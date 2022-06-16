package metadataservice

import (
	"path"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"

	"go.hollow.sh/toolbox/ginjwt"

	"go.hollow.sh/metadataservice/internal/middleware"
)

const (
	// V1URI is the path prefix for all v1 endpoints
	V1URI = "/api/v1"
	// MetadataURI is the path to the regular metadata endpoint
	MetadataURI = "/metadata"
	// UserdataURI is the path to the regular userdata endpoint
	UserdataURI = "/userdata"
)

// Router provides a router for the v1 API
type Router struct {
	AuthMW *ginjwt.Middleware
	DB     *sqlx.DB
}

// Routes will add the routes for this API version to a router group
func (r *Router) Routes(rg *gin.RouterGroup) {
	rg.GET(MetadataURI, middleware.IdentifyInstanceByIP(r.DB), r.instanceMetadataGet)
	rg.GET(UserdataURI, middleware.IdentifyInstanceByIP(r.DB), r.instanceUserdataGet)
}

// GetMetadataPath returns the path used to fetch Metadata
func GetMetadataPath() string {
	return path.Join(V1URI, MetadataURI)
}

// GetUserdataPath returns the path used to fetch Userdata
func GetUserdataPath() string {
	return path.Join(V1URI, UserdataURI)
}
