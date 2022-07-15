package metadataservice

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/types"

	"go.hollow.sh/metadataservice/internal/middleware"
	"go.hollow.sh/metadataservice/internal/models"
)

// upsertDataRequest is an interface defining the common traits shared between
// the requests for upserting metadata and userdata. Namely, an instance ID and
// a set of (optional) IP addresses.
type upsertDataRequest interface {
	getID() string
	getIPAddresses() []string
}

// upsertRecord is a function defined in by each metadata or userdata upsert
// handler function and passed into the general handleUpsertRequest function.
// This lets us share the common functionality shared between both, like
// handling conflicting IPs, adding new instance_ip_address rows, and
// removing stale instance_ip_address rows can be handled generically while
// delegating the specific implementation for handling upserting metadata
// or userdata records back to the calling method.
type upsertRecord func(c *gin.Context, exec boil.ContextExecutor) error

// UpsertMetadataRequest contains the fields for inserting or updating an
// instances metadata.
type UpsertMetadataRequest struct {
	ID          string   `json:"id" validate:"required,uuid"`
	Metadata    string   `json:"metadata" validate:"required,json"`
	IPAddresses []string `json:"ipAddresses" validate:"dive,ip_addr|cidr"`
}

func (upsertRequest *UpsertMetadataRequest) validate() error {
	return validate.Struct(upsertRequest)
}

func (upsertRequest UpsertMetadataRequest) getID() string {
	return upsertRequest.ID
}

func (upsertRequest UpsertMetadataRequest) getIPAddresses() []string {
	return upsertRequest.IPAddresses
}

// UpsertUserdataRequest contains the fields for inserting or updating an
// instances userdata.
type UpsertUserdataRequest struct {
	ID          string   `json:"id" validate:"required,uuid"`
	Userdata    []byte   `json:"userdata"`
	IPAddresses []string `json:"ipAddresses" validate:"dive,ip_addr|cidr"`
}

func (upsertRequest *UpsertUserdataRequest) validate() error {
	return validate.Struct(upsertRequest)
}

func (upsertRequest UpsertUserdataRequest) getID() string {
	return upsertRequest.ID
}

func (upsertRequest UpsertUserdataRequest) getIPAddresses() []string {
	return upsertRequest.IPAddresses
}

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

// instanceMetadataGetInternal retrieves the requested instance ID from the
// path and looks to see if the database has metadata recorded for that ID.
// If so, it returns a copy of the stored metadata. If not, it will just return
// a 404. This can be used by an authenticated external system to determine
// which instances the metadata service already knows about, and which
// instances may still need their metadata pushed to the service.
func (r *Router) instanceMetadataGetInternal(c *gin.Context) {
	instanceID, ok := c.Params.Get("instance-id")

	if !ok || instanceID == "" {
		notFoundResponse(c)
		return
	}

	metadata, err := models.FindInstanceMetadatum(c.Request.Context(), r.DB, instanceID)

	if err != nil {
		// Here, we don't want to try to look up the metadata from an external
		// system, as this endpoint should only return data for instances it
		// already knows about
		dbErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusOK, metadata.Metadata)
}

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

// instanceUserdataGetInternal retrieves the requested instance ID from the
// path and looks to see if the database has userdata recorded for that ID.
// If so, it returns a copy of the stored userdata. If not, it will just return
// a 404. This can be used by an authenticated external system to determine
// which instances the userdata service already knows about, and which
// instances may still need their userdata pushed to the service.
func (r *Router) instanceUserdataGetInternal(c *gin.Context) {
	instanceID, ok := c.Params.Get("instance-id")

	if !ok || instanceID == "" {
		notFoundResponse(c)
		return
	}

	userdata, err := models.FindInstanceUserdatum(c.Request.Context(), r.DB, instanceID)

	if err != nil {
		// Here, we don't want to try to look up the userdata from an external
		// system, as this endpoint should only return data for instances it
		// already knows about
		dbErrorResponse(c, err)
		return
	}

	c.String(http.StatusOK, string(userdata.Userdata.Bytes))
}

// There's a few steps we need to perform when upserting both instance_metadata
// and instance_userdata:
// 0. Validate the request body
// 1. Look at the list of IP addresses -- see if there are any existing instance_ip_addresses
// rows using the IP but not the same instance ID.
// 2. Look for any instance_ip_address rows for the Instance ID specified in this request.
// 3. Start a DB transaction
// 4. If we identified rows in instance_ip_addresses that match on the IP address but not the instance ID,
//    it means we're likely out-of-sync -- maybe the external system forgot to inform us that an instance
//    has been deprovisioned or that the IP address is no longer being used for that instance.
//    - 3a. We need to *at least* delete the old instance_ip_addresses row(s).
//    - 3b. We may also want to go ahead and delete the instance_metadata / instance_userdata records
//          associated to the IP as well. Or we might do this just when removing *the last*
//          instance_ip_address record for the other instance ID (but only when the removal is
//          happening due to conflict)
// 5. Remove any "stale" instance_ip_address rows for the instance ID from the request. A row would
//    be "stale" if it exists in the DB, but the associated IP address wasn't included in the request.
// 6. Add any new instance_ip_address rows for the instance ID and IP addresses in the request
// 7. Upsert the instance_metadata or instance_userdata record for this instance ID.
// 8. Finish the transaction

func (r *Router) instanceMetadataSet(c *gin.Context) {
	params := UpsertMetadataRequest{}

	// Step 0
	// Validate the request body
	if err := c.BindJSON(&params); err != nil {
		badRequestResponse(c, "invalid request body", err)
		return
	}

	if err := params.validate(); err != nil {
		badRequestResponse(c, "Invalid request", err)
		return
	}

	upsertInstanceMetadata := func(c *gin.Context, exec boil.ContextExecutor) error {
		newInstanceMetadata := &models.InstanceMetadatum{
			ID:       params.ID,
			Metadata: types.JSON(params.Metadata),
		}

		return newInstanceMetadata.Upsert(c, exec, true, []string{"id"}, boil.Whitelist("metadata"), boil.Infer())
	}

	handleUpsertRequest(c, r, params, upsertInstanceMetadata)
}

func (r *Router) instanceUserdataSet(c *gin.Context) {
	params := UpsertUserdataRequest{}

	// Validate the request
	if err := c.BindJSON(&params); err != nil {
		badRequestResponse(c, "invalid request body", err)
		return
	}

	if err := params.validate(); err != nil {
		badRequestResponse(c, "invalid request", err)
		return
	}

	upsertInstanceUserdata := func(c *gin.Context, exec boil.ContextExecutor) error {
		newInstanceUserdata := &models.InstanceUserdatum{
			ID:       params.ID,
			Userdata: null.NewBytes(params.Userdata, true),
		}

		return newInstanceUserdata.Upsert(c, exec, true, []string{"id"}, boil.Whitelist("userdata"), boil.Infer())
	}

	handleUpsertRequest(c, r, params, upsertInstanceUserdata)
}

func handleUpsertRequest(c *gin.Context, r *Router, params upsertDataRequest, upsertRecordFunc upsertRecord) {
	// Step 1
	// Look for any conflicting IP addresses (IPs already present and associated
	// with a *different* Instance ID)
	conflictIPs, err := models.InstanceIPAddresses(models.InstanceIPAddressWhere.Address.IN(params.getIPAddresses()), models.InstanceIPAddressWhere.InstanceID.NEQ(params.getID())).All(c, r.DB)

	if err != nil {
		dbErrorResponse(c, err)
		return
	}

	// Step 2
	// Look up any existing instance_ip_addresses rows for the provided instance_id
	instanceIPAddresses, err := models.InstanceIPAddresses(models.InstanceIPAddressWhere.InstanceID.EQ(params.getID())).All(c, r.DB)

	if err != nil {
		dbErrorResponse(c, err)
		return
	}

	// Step 2.5.a
	// Find "stale" InstanceIPAddress rows for this instance. That is, select
	// rows from the instanceIPAddresses result which don't have a corresponding
	// entry in the list of IP Addresses supplied in the request.
	var staleInstanceIPAddreses models.InstanceIPAddressSlice

	for _, instanceIP := range instanceIPAddresses {
		found := false

		for _, paramIP := range params.getIPAddresses() {
			if strings.EqualFold(instanceIP.Address, paramIP) {
				found = true
				break
			}
		}

		if !found {
			staleInstanceIPAddreses = append(staleInstanceIPAddreses, instanceIP)
		}
	}

	// Step 2.5.b
	// Find new IP Addresses that were specified in the request that aren't
	// currently associated to the instance.
	var newInstanceIPAddresses models.InstanceIPAddressSlice

	for _, paramIP := range params.getIPAddresses() {
		found := false

		for _, instanceIP := range instanceIPAddresses {
			if strings.EqualFold(paramIP, instanceIP.Address) {
				found = true
				break
			}
		}

		if !found {
			newRecord := &models.InstanceIPAddress{
				InstanceID: params.getID(),
				Address:    paramIP,
			}
			newInstanceIPAddresses = append(newInstanceIPAddresses, newRecord)
		}
	}

	// Step 3
	// Kick off the DB transaction
	txErr := false

	tx, err := r.DB.BeginTx(c, nil)
	if err != nil {
		dbErrorResponse(c, err)
		return
	}

	// // If there's an error, we'll want to rollback the transaction.
	defer func() {
		if txErr {
			err := tx.Rollback()
			if err != nil {
				r.Logger.Sugar().Error("Could not rollback transaction", "error", err)
			}
		}
	}()

	// Step 4
	// Remove any instance_ip_address rows for the specified IP addresses that
	// are currently associated to a *different* instance ID
	for _, conflictingIP := range conflictIPs {
		// TODO: Maybe remove instance_metadata and instance_userdata records for the "old" instance ID(s)?
		// Potentially after checking to see if this IP was the *last* IP address associated to the
		// "old" Instance ID?
		_, err := conflictingIP.Delete(c, tx)
		if err != nil {
			txErr = true

			dbErrorResponse(c, err)

			return
		}
	}

	// Step 5
	// Remove any "stale" instance_ip_addresses rows associated to the provided
	// instance_id but were not specified in this request.
	for _, staleIP := range staleInstanceIPAddreses {
		_, err := staleIP.Delete(c, tx)
		if err != nil {
			txErr = true

			dbErrorResponse(c, err)

			return
		}
	}

	// Step 6
	// Create instance_ip_addresses rows for any IP addresses specified in the
	// request that aren't already associated to the provided instance_id
	for _, newInstanceIP := range newInstanceIPAddresses {
		err := newInstanceIP.Insert(c, tx, boil.Infer())
		if err != nil {
			txErr = true

			dbErrorResponse(c, err)

			return
		}
	}

	// Step 7
	// Upsert the instance_metadata or instance_userdata table. This will create
	// a new row with the provided instance ID and metadata or userdata if there
	// is no current row for instance_id. If there is an existing row matching on
	// instance_id, instead this will just update the metadata or userdata column value.
	if err := upsertRecordFunc(c, tx); err != nil {
		txErr = true

		dbErrorResponse(c, err)

		return
	}

	// Step 8
	// Commit our transaction
	// if err := c.BindJSON(&params); err != nil {
	if err := tx.Commit(); err != nil {
		txErr = true

		dbErrorResponse(c, err)

		return
	}

	c.Status(http.StatusOK)
}
