package metadataservice

import (
	"database/sql"
	"encoding/json"
	"errors"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/types"

	"go.hollow.sh/metadataservice/internal/middleware"
	"go.hollow.sh/metadataservice/internal/models"
	"go.hollow.sh/metadataservice/internal/upserter"
)

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
	metadata, err := r.getMetadata(c)

	// If we got an error trying to retrieve metadata for the caller, and the
	// error wasn't a "not found" error, we should just return a generic 500
	// error result to the caller.
	if err != nil && !errors.Is(err, errNotFound) {
		dbErrorResponse(r.Logger, c, err)
		return
	}

	if metadata != nil {
		augmentedMetadata, err := addTemplateFields(metadata.Metadata, r.TemplateFields)
		if err != nil {
			r.Logger.Sugar().Warnf("Error adding additional templated fields to metadata for instance %s", metadata.ID, "error", err)

			// Since we couldn't add the templated fields, just return the metadata as-is
			c.JSON(http.StatusOK, metadata.Metadata)
		} else {
			c.JSON(http.StatusOK, augmentedMetadata)
		}
	} else {
		notFoundResponse(c)
	}
}

// instanceMetadataGetInternal retrieves the requested instance ID from the
// path and looks to see if the database has metadata recorded for that ID.
// If so, it returns a copy of the stored metadata. If not, it will just return
// a 404. This can be used by an authenticated external system to determine
// which instances the metadata service already knows about, and which
// instances may still need their metadata pushed to the service.
func (r *Router) instanceMetadataGetInternal(c *gin.Context) {
	instanceID, err := getUUIDParam(c, "instance-id")

	if err != nil {
		invalidUUIDResponse(c, err)
		return
	}

	metadata, err := models.FindInstanceMetadatum(c.Request.Context(), r.DB, instanceID)

	if err != nil {
		// Here, we don't want to try to look up the metadata from an external
		// system, as this endpoint should only return data for instances it
		// already knows about
		dbErrorResponse(r.Logger, c, err)
		return
	}

	augmentedMetadata, err := addTemplateFields(metadata.Metadata, r.TemplateFields)
	if err != nil {
		r.Logger.Sugar().Warnf("Error adding additional templated fields to metadata for instance %s", metadata.ID, "error", err)

		// Since we couldn't add the templated fields, just return the metadata as-is
		c.JSON(http.StatusOK, metadata.Metadata)
	} else {
		c.JSON(http.StatusOK, augmentedMetadata)
	}
}

// instanceMetadataExistsInternal retrieves the requested instance ID from the
// path and looks to see if the database has metadata recorded for that ID.
// If so, it returns a 200. If not, it returns a 404. This can be used by an
// authenticated external system to determine which instances the metadata
// service already knows about with minimal network overhead.
func (r *Router) instanceMetadataExistsInternal(c *gin.Context) {
	instanceID, err := getUUIDParam(c, "instance-id")

	if err != nil {
		invalidUUIDResponse(c, err)
		return
	}

	metadata, err := models.FindInstanceMetadatum(c.Request.Context(), r.DB, instanceID)

	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	// HEAD request responses still set the Content-Length header to what it
	// would be if we were returning the metadata
	bytes, err := json.Marshal(metadata.Metadata)
	if err != nil {
		r.Logger.Warn("Error during json.Marshal() of metadata")
		c.Status(http.StatusInternalServerError)

		return
	}

	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(bytes)))
	c.Status(http.StatusOK)
}

func (r *Router) instanceUserdataGet(c *gin.Context) {
	userdata, err := r.getUserdata(c)

	// If we got an error trying to retrieve userdata for the caller, and the
	// error wasn't a "not found" error, we should just return a generic 500
	// error result to the caller.
	if err != nil && !errors.Is(err, errNotFound) {
		dbErrorResponse(r.Logger, c, err)
		return
	}

	if userdata != nil {
		c.String(http.StatusOK, string(userdata.Userdata.Bytes))
	} else {
		notFoundResponse(c)
	}
}

// instanceUserdataGetInternal retrieves the requested instance ID from the
// path and looks to see if the database has userdata recorded for that ID.
// If so, it returns a copy of the stored userdata. If not, it will just return
// a 404. This can be used by an authenticated external system to determine
// which instances the userdata service already knows about, and which
// instances may still need their userdata pushed to the service.
func (r *Router) instanceUserdataGetInternal(c *gin.Context) {
	instanceID, err := getUUIDParam(c, "instance-id")

	if err != nil {
		invalidUUIDResponse(c, err)
		return
	}

	userdata, err := models.FindInstanceUserdatum(c.Request.Context(), r.DB, instanceID)

	if err != nil {
		// Here, we don't want to try to look up the userdata from an external
		// system, as this endpoint should only return data for instances it
		// already knows about
		dbErrorResponse(r.Logger, c, err)
		return
	}

	c.String(http.StatusOK, string(userdata.Userdata.Bytes))
}

// instanceUserdataExistsInternal retrieves the requested instance ID from the
// path and looks to see if the database has userdata recorded for that ID.
// If so, it returns a 200. If not, it will just return a 404. This can be use
// by an authenticated external system to determine which instances the userdata
// service already knows about with minimal network overhead.
func (r *Router) instanceUserdataExistsInternal(c *gin.Context) {
	instanceID, err := getUUIDParam(c, "instance-id")

	if err != nil {
		invalidUUIDResponse(c, err)
		return
	}

	userdata, err := models.FindInstanceUserdatum(c.Request.Context(), r.DB, instanceID)

	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	// HEAD request responses still set the Content-Length header to what it
	// would be if we were returning the userdata
	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(userdata.Userdata.Bytes)))
	c.Status(http.StatusOK)
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

	newInstanceMetadata := &models.InstanceMetadatum{
		ID:       params.getID(),
		Metadata: types.JSON(params.Metadata),
	}

	err := upserter.UpsertMetadata(c, r.DB, r.Logger, params.ID, params.getIPAddresses(), newInstanceMetadata)
	if err != nil {
		dbErrorResponse(r.Logger, c, err)
	}

	c.Status(http.StatusOK)
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

	newInstanceUserdata := &models.InstanceUserdatum{
		ID:       params.getID(),
		Userdata: null.NewBytes(params.Userdata, true),
	}

	err := upserter.UpsertUserdata(c, r.DB, r.Logger, params.ID, params.getIPAddresses(), newInstanceUserdata)
	if err != nil {
		dbErrorResponse(r.Logger, c, err)
	}

	c.Status(http.StatusOK)
}

func (r *Router) instanceMetadataDelete(c *gin.Context) {
	// When deleting metadata for an instance, we need to check if there is
	// userdata stored for the instance. If there is not, we should go ahead and
	// also delete the associated instance_ip_addresses rows.
	instanceID, err := getUUIDParam(c, "instance-id")

	if err != nil {
		invalidUUIDResponse(c, err)

		return
	}

	metadata, err := models.FindInstanceMetadatum(c.Request.Context(), r.DB, instanceID)

	if err != nil {
		dbErrorResponse(r.Logger, c, err)
		return
	}

	handleDeleteRequest(c, r, instanceID, metadata, nil)
}

func (r *Router) instanceUserdataDelete(c *gin.Context) {
	// When deleting userdata for an instance, we need to check if there is
	// metadata stored for the instance. If there is not, we should go ahead and
	// also delete the associated instance_ip_addresses rows.
	instanceID, err := getUUIDParam(c, "instance-id")

	if err != nil {
		invalidUUIDResponse(c, err)

		return
	}

	userdata, err := models.FindInstanceUserdatum(c.Request.Context(), r.DB, instanceID)

	if err != nil {
		dbErrorResponse(r.Logger, c, err)
		return
	}

	handleDeleteRequest(c, r, instanceID, nil, userdata)
}

func handleDeleteRequest(c *gin.Context, r *Router, instanceID string, metadata *models.InstanceMetadatum, userdata *models.InstanceUserdatum) {
	var err error

	deleteMetadata := metadata != nil
	deleteUserdata := userdata != nil

	deleteInstanceIPs := false

	// Step 1
	// Attempt to load instance metadata or instance userdata, depending on if
	// they were passed in as nil
	if metadata == nil {
		metadata, err = models.FindInstanceMetadatum(c.Request.Context(), r.DB, instanceID)
		// An ErrNoRows error is expected, so disregard it.
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			dbErrorResponse(r.Logger, c, err)
			return
		}
	}

	if userdata == nil {
		userdata, err = models.FindInstanceUserdatum(c.Request.Context(), r.DB, instanceID)
		// An ErrNoRows error is expected, so disregard it.
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			dbErrorResponse(r.Logger, c, err)
			return
		}
	}

	switch {
	case deleteMetadata && deleteUserdata:
		deleteInstanceIPs = true
	case deleteMetadata:
		deleteInstanceIPs = (userdata == nil)
	case deleteUserdata:
		deleteInstanceIPs = (metadata == nil)
	}

	deleteSuccess := false
	maxDeleteRetries := viper.GetInt("crdb.max_retries")
	dbRetryInterval := viper.GetDuration("crdb.retry_interval")

	for i := 0; i <= maxDeleteRetries && !deleteSuccess; i++ {
		err := performDeletions(c, r, instanceID, metadata, userdata, deleteMetadata, deleteUserdata, deleteInstanceIPs)
		if err == nil {
			deleteSuccess = true

			if i > 0 {
				r.Logger.Sugar().Info("DB delete transaction for instance ", instanceID, " successful on retry attempt #", i)
			}
		} else {
			// Exponential backoff would be overkill here, but adding a bit of jitter
			// to sleep a short time is reasonable
			jitter := time.Duration(rand.Int63n(int64(dbRetryInterval)))
			time.Sleep(jitter)
		}
	}

	if !deleteSuccess {
		r.Logger.Sugar().Warn("Deletion operation failed for instance ", instanceID, " even after ", maxDeleteRetries, " attempts")

		dbErrorResponse(r.Logger, c, err)

		return
	}

	middleware.MetricDeletionsCount.Inc()

	c.Status(http.StatusOK)
}

func performDeletions(c *gin.Context, r *Router, instanceID string, metadata *models.InstanceMetadatum, userdata *models.InstanceUserdatum, deleteMetadata bool, deleteUserdata bool, deleteInstanceIPs bool) error {
	// Step 2
	// Now that we've determined if we should delete the corresponding
	// instance_ip_addresses rows, start a transaction, delete the passed-in
	// record, and potentially delete the associated instance_ip_addresses rows.
	txErr := false

	tx, err := r.DB.BeginTx(c, nil)
	if err != nil {
		r.Logger.Sugar().Warn("Something went wrong when running DB.BeginTX() for instance: ", instanceID)
		dbErrorResponse(r.Logger, c, err)

		return err
	}

	// If there's an error, we'll want to rollback the transaction.
	defer func() {
		if txErr {
			r.Logger.Sugar().Warn("Rolling back delete transaction for instance: ", instanceID)

			err := tx.Rollback()
			if err != nil {
				r.Logger.Sugar().Error("Could not rollback delete transaction for instance ", instanceID, "error", err)
			}
		}
	}()

	// Step 3
	// Delete the metadata and/or userdata record, depending on which one was passed in.
	if deleteMetadata {
		_, err := metadata.Delete(c, tx)
		if err != nil {
			txErr = true

			r.Logger.Sugar().Warn("Something went wrong when setting up metadata.Delete transaction for instance: ", instanceID)
			dbErrorResponse(r.Logger, c, err)

			return err
		}
	}

	if deleteUserdata {
		_, err := userdata.Delete(c, tx)
		if err != nil {
			txErr = true

			r.Logger.Sugar().Warn("Something went wrong when setting up userdata.Delete transaction for instance: ", instanceID)
			dbErrorResponse(r.Logger, c, err)

			return err
		}
	}

	// Step 4
	// Delete the instance_ip_addresses rows if we've deleted the last metadata
	// or userdata record associated to the instance ID.
	if deleteInstanceIPs {
		_, err := models.InstanceIPAddresses(models.InstanceIPAddressWhere.InstanceID.EQ(instanceID)).DeleteAll(c, tx)
		if err != nil {
			txErr = true

			r.Logger.Sugar().Warn("Something went wrong when setting up deleteInstanceIPs transaction for instance: ", instanceID)
			dbErrorResponse(r.Logger, c, err)

			return err
		}
	}

	// Step 5
	// commit our transaction
	err = tx.Commit()
	if err != nil {
		txErr = true

		r.Logger.Sugar().Warn("Unable to commit db delete transaction: ", err)

		return err
	}

	return nil
}
