package middleware

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"go.uber.org/zap"

	"go.hollow.sh/metadataservice/internal/models"
)

// ContextKeyInstanceID is the magic string set in the gin.Context key/value
// store used for storing the ID of the instance making the request, if the
// instance has been identified.
const ContextKeyInstanceID = "instance-id"

// ContextKeyRequestorIP is the magic string set in the gin.Context key/value
// store used for storing the IP address of a caller making a request for
// metadata or userdata.
const ContextKeyRequestorIP = "requestor-ip-address"

// When a request comes in to the /metadata or /userdata endpoints (or the 2009-04-04/* variants)
// we need to identify the instance making the request.
// There's 2 ways to do this:
// a) (pending) if the request was made with the special auth header that tells
// us the request is being proxied for the instance through another system
// (like a switch), use the auth header info to get the instance ID.
// OR
// b) via the request ip from the instance making the request.
//
// For case (a), we'll know the instance ID, and can check if we have metadata
// or userdata stored for that ID. If not, we need to fetch it from an external
// system.
// For case (b), we'll look up the instance ID from our instance_ip_addresses
// table. If there's no rows matching the request IP, we'll know we need to
// fetch it from an external system.

// IdentifyInstanceByIP is used to determine the ID of the instance making the
// request by looking at the request IP.
// If a row in the instance_ip_addresses table is found with a matching IP
// address, we set the instance ID in the context.
func IdentifyInstanceByIP(logger *zap.Logger, db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			address           string
			instanceIPAddress *models.InstanceIPAddress
			err               error
		)

		// When trusted proxies are configured in gin, ClientIP() will use the
		// X-Forwarded-For or X-Real-Ip headers (if present) to report the remote
		// IP. If trusted proxies are not configured, these headers will be ignored
		// to prevent spoofing by clients, and instead the request's RemoteAddr
		// will be returned.
		// But if a proxy is sitting in front of this service, RemoteAddr will be
		// the IP of the proxy, and not the requestor.
		// Use the `gin-trusted-proxies` flag
		// (or METADATASERVICE_GIN_TRUSTED_PROXIES envvar) when starting the server
		// to provide the list of trusted proxy IP's to use.
		address = c.ClientIP()

		c.Set(ContextKeyRequestorIP, address)

		instanceIPAddress, err = models.InstanceIPAddresses(qm.Where("address >>= ?::inet", address)).One(c, db)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			logger.Error("error looking up instance address", zap.Error(err))

			c.AbortWithStatus(http.StatusInternalServerError)
		}

		if instanceIPAddress != nil {
			// We found the row, set the instance ID into the gin context.
			c.Set(ContextKeyInstanceID, instanceIPAddress.InstanceID)
		}
	}
}
