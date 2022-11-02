package metadataservice

import (
	"fmt"
	"path"
	"strings"

	"github.com/gin-gonic/gin"

	"go.hollow.sh/metadataservice/internal/middleware"
)

const (
	// V20090404URI is the path prefix for the ec2-style (v2009-04-04) format
	V20090404URI = "/2009-04-04"

	// Ec2MetadataURI is the path to the ec2-style metadata endpoint for listing
	// available metadata items for the instance.
	Ec2MetadataURI = "/meta-data"

	// Ec2MetadataItemURI is the path to the ec2-style metadata endpoint for
	// retrieving a specified metadata item value.
	Ec2MetadataItemURI = "/meta-data/*subpath"

	// Ec2UserdataURI is the path to the ec2-style userdata endpoint
	Ec2UserdataURI = "/user-data"
)

// Ec2Routes will add the routes for the EC2-style API to a router group
func (r *Router) Ec2Routes(rg *gin.RouterGroup) {
	// GET /2009-04-04/meta-data/:item-name
	// GET /2009-04-04/user-data
	rg.GET(Ec2MetadataURI, middleware.IdentifyInstanceByIP(r.Logger, r.DB), r.instanceEc2MetadataGet)
	rg.GET(Ec2MetadataItemURI, middleware.IdentifyInstanceByIP(r.Logger, r.DB), r.instanceEc2MetadataItemGet)
	rg.GET(Ec2UserdataURI, middleware.IdentifyInstanceByIP(r.Logger, r.DB), r.instanceEc2UserdataGet)
}

// GetEc2MetadataPath returns the path used to fetch a list of the ec2-style
// metadata item fields for the instance
func GetEc2MetadataPath() string {
	return path.Join(V20090404URI, Ec2MetadataURI)
}

// GetEc2MetadataItemPath returns the path used to fetch a specific metadata
// item.
// Ex: GetEx2MetadataItemPath("foo/bar/baz") returns:
// "/2009-04-04/meta-data/foo/bar/baz"
func GetEc2MetadataItemPath(itemPath string) string {
	trimmed := strings.Trim(itemPath, "/")
	return path.Join(V20090404URI, fmt.Sprintf("%s/%s", Ec2MetadataURI, trimmed))
}

// GetEc2UserdataPath returns the path used to fetch ec2-style userdata
func GetEc2UserdataPath() string {
	return path.Join(V20090404URI, Ec2UserdataURI)
}
