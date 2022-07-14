package metadataservice

import (
	"fmt"
	"path"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"go.hollow.sh/toolbox/ginjwt"

	"go.hollow.sh/metadataservice/internal/middleware"
)

const (
	// V1URI is the path prefix for all v1 endpoints
	V1URI = "/api/v1"

	// MetadataURI is the path to the regular metadata endpoint, called by the
	// instances themselves to retrieve their metadata.
	MetadataURI = "/metadata"

	// UserdataURI is the path to the regular userdata endpoint, called by the
	// instances themselves to retrieve their userdata.
	UserdataURI = "/userdata"

	// InternalMetadataURI is the path to the internal (authenticated) endpoint
	// used for updating & retrieving metadata for any instance
	InternalMetadataURI = "/device-metadata"

	// InternalUserdataURI is the path to the internal (authenticated) endpoint
	// used for updating & retrieving metadata for any instance
	InternalUserdataURI = "/device-userdata"

	// InternalMetadataWithIDURI is the path to the internal (authenticated)
	// endpoint used for retrieving the stored metadata for an instance
	InternalMetadataWithIDURI = "/device-metadata/:instance-id"

	// InternalUserdataWithIDURI is the path to the internal (authenticated)
	// endpoint used for retrieving the stored metadata for an instance
	InternalUserdataWithIDURI = "/device-userdata/:instance-id"

	scopePrefix = "metadata"
)

var (
	validate *validator.Validate
)

// Router provides a router for the v1 API
type Router struct {
	AuthMW *ginjwt.Middleware
	DB     *sqlx.DB
	Logger *zap.Logger
}

// Routes will add the routes for this API version to a router group
func (r *Router) Routes(rg *gin.RouterGroup) {
	setupValidator()

	rg.GET(MetadataURI, middleware.IdentifyInstanceByIP(r.DB), r.instanceMetadataGet)
	rg.GET(UserdataURI, middleware.IdentifyInstanceByIP(r.DB), r.instanceUserdataGet)

	authMw := r.AuthMW
	rg.POST(InternalMetadataURI, authMw.AuthRequired(), authMw.RequiredScopes(upsertScopes("metadata")), r.instanceMetadataSet)
	rg.POST(InternalUserdataURI, authMw.AuthRequired(), authMw.RequiredScopes(upsertScopes("userdata")), r.instanceUserdataSet)

	rg.GET(InternalMetadataWithIDURI, authMw.AuthRequired(), authMw.RequiredScopes(readScopes("metadata")), r.instanceMetadataGetInternal)
	rg.GET(InternalUserdataWithIDURI, authMw.AuthRequired(), authMw.RequiredScopes(readScopes("metadata")), r.instanceUserdataGetInternal)
}

// GetMetadataPath returns the path used by an instance to fetch Metadata
func GetMetadataPath() string {
	return path.Join(V1URI, MetadataURI)
}

// GetUserdataPath returns the path used by an instance to fetch Userdata
func GetUserdataPath() string {
	return path.Join(V1URI, UserdataURI)
}

// GetInternalMetadataPath returns the path used by an internal, authenticated
// system or used to update or retrieve metadata.
func GetInternalMetadataPath() string {
	return path.Join(V1URI, InternalMetadataURI)
}

// GetInternalMetadataByIDPath returns the path used by an internal,
// authenticated system or user to retrieve the metadata for a specific
// instance.
func GetInternalMetadataByIDPath(id string) string {
	return path.Join(V1URI, InternalMetadataURI, id)
}

// GetInternalUserdataPath returns the patch used by an internal, authenticated
// system or used to update or retrieve userdata.
func GetInternalUserdataPath() string {
	return path.Join(V1URI, InternalUserdataURI)
}

// GetInternalUserdataByIDPath returns the path used by an internal,
// authenticated system or user to retrieve the metadata for a specific
// instance.
func GetInternalUserdataByIDPath(id string) string {
	return path.Join(V1URI, InternalUserdataURI, id)
}

func upsertScopes(items ...string) []string {
	s := []string{"write", "create", "update"}
	for _, i := range items {
		s = append(s, fmt.Sprintf("%s:create:%s", scopePrefix, i))
	}

	for _, i := range items {
		s = append(s, fmt.Sprintf("%s:update:%s", scopePrefix, i))
	}

	return s
}

func readScopes(items ...string) []string {
	s := []string{"read"}
	for _, i := range items {
		s = append(s, fmt.Sprintf("%s:read:%s", scopePrefix, i))
	}

	return s
}

func setupValidator() {
	validate = validator.New()

	splitSliceNum := 2

	// Set up a function to grab the json tag from a struct (if set)
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", splitSliceNum)[0]
		if name == "-" {
			return ""
		}
		return name
	})
}
