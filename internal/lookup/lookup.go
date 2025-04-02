package lookup

import (
	"context"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/spf13/viper"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/types"
	"go.uber.org/zap"

	"go.hollow.sh/metadataservice/internal/middleware"
	"go.hollow.sh/metadataservice/internal/models"
	"go.hollow.sh/metadataservice/internal/upserter"
)

var (
	// ErrUnexpectedStatus indicates to the caller that the upstream lookup service
	// returned an HTTP response status code our client wasn't able to handle. This
	// may indicate errors like the upstream service is temporarily unavailable, or
	// our authentication credentials were not valid.
	ErrUnexpectedStatus = errors.New("unexpectedStatusError")

	// ErrNotFound indicates to the caller that the upstream lookup service
	// returned an HTTP 404 status code, meaning that whatever instance ID or
	// IP address we specified was not known by the upstream service.
	ErrNotFound = errors.New("notFoundError")

	errNilClient = errors.New("client can't be nil")
)

// MetadataSyncByID calls out to the metadata lookup service and
// attempts to locate metadata for the instance with the given ID. If found,
// it will create new records in the database for the instance IP addresses
// and metadata.
func MetadataSyncByID(ctx context.Context, db *sqlx.DB, logger *zap.Logger, client Client, id string) (*models.InstanceMetadatum, error) {
	if client == nil {
		return nil, errNilClient
	}

	middleware.MetricMetadataLookupRequestCount.Inc()

	resp, err := client.GetMetadataByID(ctx, id)
	if err != nil {
		middleware.MetricLookupErrors.Inc()
		return nil, err
	}

	return storeMetadata(ctx, db, logger, resp)
}

// MetadataSyncByIP calls out to the metadata lookup service and
// attempts to locate metadata for the instance with the given IP address. If
// found, it will create new records in database for the instance IP addresses
// and metadata.
func MetadataSyncByIP(ctx context.Context, db *sqlx.DB, logger *zap.Logger, client Client, ipAddress string) (*models.InstanceMetadatum, error) {
	if client == nil {
		return nil, errNilClient
	}

	middleware.MetricMetadataLookupRequestCount.Inc()

	resp, err := client.GetMetadataByIP(ctx, ipAddress)
	if err != nil {
		middleware.MetricLookupErrors.Inc()
		return nil, err
	}

	return storeMetadata(ctx, db, logger, resp)
}

// UserdataSyncByID calls out to the metadata lookup service and
// attempts to locate userdata for the instance with the given ID. If found,
// it will create new records in the database for the instance IP addresses
// and userdata.
func UserdataSyncByID(ctx context.Context, db *sqlx.DB, logger *zap.Logger, client Client, id string) (*models.InstanceUserdatum, error) {
	if client == nil {
		return nil, errNilClient
	}

	middleware.MetricUserdataLookupRequestCount.Inc()

	resp, err := client.GetUserdataByID(ctx, id)
	if err != nil {
		middleware.MetricUserdataLookupErrors.Inc()
		return nil, err
	}

	return storeUserdata(ctx, db, logger, resp)
}

// UserdataSyncByIP calls out to the metadata lookup service and
// attempts to locate userdata for the instance with the given IP address. If
// found, it will create new records in the database for the instance IP
// addresses and userdata.
func UserdataSyncByIP(ctx context.Context, db *sqlx.DB, logger *zap.Logger, client Client, ipAddress string) (*models.InstanceUserdatum, error) {
	if client == nil {
		return nil, errNilClient
	}

	middleware.MetricUserdataLookupRequestCount.Inc()

	resp, err := client.GetUserdataByIP(ctx, ipAddress)
	if err != nil {
		middleware.MetricUserdataLookupErrors.Inc()
		return nil, err
	}

	return storeUserdata(ctx, db, logger, resp)
}

func storeMetadata(ctx context.Context, db *sqlx.DB, logger *zap.Logger, lookupResp *MetadataLookupResponse) (*models.InstanceMetadatum, error) {
	newInstanceMetadata := &models.InstanceMetadatum{
		ID:       lookupResp.ID,
		Metadata: types.JSON(lookupResp.Metadata),
	}

	if viper.GetBool("crdb.enabled") {
		err := upserter.UpsertMetadata(ctx, db, logger, lookupResp.ID, lookupResp.IPAddresses, newInstanceMetadata)
		if err != nil {
			middleware.MetricMetadataStoreErrors.Inc()
			return nil, err
		}

		middleware.MetricMetadataInsertsCount.Inc()
	} else {
		logger.Sugar().Infof("storeMetadata: DB is disabled, skipping upsert for instance %s", lookupResp.ID)
	}

	return newInstanceMetadata, nil
}

func storeUserdata(ctx context.Context, db *sqlx.DB, logger *zap.Logger, lookupResp *UserdataLookupResponse) (*models.InstanceUserdatum, error) {
	newInstanceUserdata := &models.InstanceUserdatum{
		ID:       lookupResp.ID,
		Userdata: null.NewBytes(lookupResp.Userdata, true),
	}

	if viper.GetBool("crdb.enabled") {
		err := upserter.UpsertUserdata(ctx, db, logger, lookupResp.ID, lookupResp.IPAddresses, newInstanceUserdata)
		if err != nil {
			middleware.MetricUserdataStoreErrors.Inc()
			return nil, err
		}

		middleware.MetricUserdataInsertsCount.Inc()
	} else {
		logger.Sugar().Infof("storeUserdata: DB is disabled, skipping upsert for instance %s", lookupResp.ID)
	}

	logger.Sugar().Infof("storeUserdata: returning newInstanceUserdata %s for instance %s", newInstanceUserdata.Userdata, lookupResp.ID)

	return newInstanceUserdata, nil
}
