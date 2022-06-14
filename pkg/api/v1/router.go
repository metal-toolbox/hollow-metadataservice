package metadataservice

import (
	"path"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"

	"go.hollow.sh/metadataservice/internal/middleware"
	"go.hollow.sh/toolbox/ginjwt"
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

// // Routes will add the routes for this API version to a router group

func (r *Router) Routes(rg *gin.RouterGroup) {
	// amw := r.AuthMW

	rg.GET(MetadataURI, middleware.IdentifyInstanceByIP(r.DB), r.instanceMetadataGet)
	rg.GET(UserdataURI, middleware.IdentifyInstanceByIP(r.DB), r.instanceUserdataGet)
}

func GetMetadataPath() string {
	return path.Join(V1URI, MetadataURI)
}

func GetUserdataPath() string {
	return path.Join(V1URI, UserdataURI)
}
