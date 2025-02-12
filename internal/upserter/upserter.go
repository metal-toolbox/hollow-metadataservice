package upserter

import (
	"context"
	"database/sql"
	"encoding/json"
	"math/rand/v2"
	"strings"

	"github.com/cockroachdb/cockroach-go/v2/crdb"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"go.uber.org/zap"

	"go.hollow.sh/metadataservice/internal/models"
)

// RecordUpserter is a function defined in by each metadata or userdata upsert
// handler function and passed into the general handleUpsertRequest function.
// This lets us share the common functionality shared between both, like
// handling conflicting IPs, adding new instance_ip_address rows, and
// removing stale instance_ip_address rows can be handled generically while
// delegating the specific implementation for handling upserting metadata
// or userdata records back to the calling method.
type RecordUpserter func(c context.Context, exec boil.ContextExecutor) error

// The following types are used to unmarshal the metadata JSON body so we can
// extract the IP addresses from it for logging.

// NetworkAddress is a struct used to unmarshal the "network.addresses" JSON array
type NetworkAddress struct {
	Address string `json:"address"`
}

// Network is a struct used to unmarshal the "network" JSON object
type Network struct {
	Addresses []NetworkAddress `json:"addresses"`
}

// MetadataContent is a struct used to unmarshal the metadata JSON body
type MetadataContent struct {
	Network Network `json:"network"`
}

// ExtractIPAddressesFromMetadata is a helper function used to extract IP addresses
// from the metadata JSON. We only use this for logging purposes, so it can fail silently.
func ExtractIPAddressesFromMetadata(metadata *models.InstanceMetadatum) []string {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(metadata.Metadata), &raw); err != nil {
		return nil
	}

	network, ok := raw["network"].(map[string]interface{})
	if !ok {
		return nil
	}

	addresses, ok := network["addresses"].([]interface{})
	if !ok {
		return nil
	}

	result := make([]string, 0, len(addresses))

	for _, addr := range addresses {
		if addrMap, ok := addr.(map[string]interface{}); ok {
			if ipAddr, ok := addrMap["address"].(string); ok {
				result = append(result, ipAddr)
			}
		}
	}

	return result
}

// UpsertMetadata is used to upsert (update or insert) an instance_metadata
// record, along with managing inserting new instance_ip_addresses rows and
// removing conflicting or stale instance_ip_addresses rows.
func UpsertMetadata(ctx context.Context, db *sqlx.DB, logger *zap.Logger, id string, ipAddresses []string, metadata *models.InstanceMetadatum) error {
	metadataUpserter := func(c context.Context, exec boil.ContextExecutor) error {
		return metadata.Upsert(c, exec, true, []string{"id"}, boil.Whitelist("metadata", "updated_at"), boil.Infer())
	}

	// Extract all IP addresses from the metadata body - note that this is different from
	// the ipAddresses list, which doesn't include IPv6 addresses, as it only includes
	// addresses that the metadata service would conceivably perform lookups based on.
	allIPs := ExtractIPAddressesFromMetadata(metadata)
	logger.Sugar().Info("Starting metadata upsert for instance uuid: ", id, " where metadata contains IPs: ", allIPs)

	return doUpsert(ctx, db, logger, id, ipAddresses, metadataUpserter)
}

// UpsertUserdata is used to upsert (update or insert) an instance_userdata
// record, along with managing inserting new instance_ip_addresses rows and
// removing conflicting or stale instance_ip_addresses rows.
func UpsertUserdata(ctx context.Context, db *sqlx.DB, logger *zap.Logger, id string, ipAddresses []string, userdata *models.InstanceUserdatum) error {
	userdataUpserter := func(c context.Context, exec boil.ContextExecutor) error {
		return userdata.Upsert(c, exec, true, []string{"id"}, boil.Whitelist("userdata", "updated_at"), boil.Infer())
	}

	logger.Sugar().Info("Starting userdata upsert for instance uuid: ", id)

	return doUpsert(ctx, db, logger, id, ipAddresses, userdataUpserter)
}

// doUpsert performs an upsert operation using explicit row locks to ensure upserts are atomic
// doUpsert performs an upsert operation using explicit row locks to ensure upserts are atomic
func doUpsert(ctx context.Context, db *sqlx.DB, logger *zap.Logger, id string, ipAddresses []string, upsertRecordFunc RecordUpserter) error {
	// Generate a 5-digit random ID between 10000 and 99999 for this upsert operation for logging purposes
	lowerLimit, upperLimit := 10000, 9000
	upsertID := lowerLimit + rand.IntN(upperLimit)

	logger.Sugar().Info("doUpsert ", upsertID, " starting for instance uuid: ", id, " - upserting using lookupable IPs ", ipAddresses)

	maxRetries := viper.GetInt("crdb.max_retries")
	retryCount := 0

	for {
		// Create a new context with a timeout for the DB transaction
		ctxWithTimeout, cancel := context.WithTimeout(ctx, viper.GetDuration("crdb.tx_timeout"))

		// Start a DB transaction using crdb.ExecuteTx, which has built-in support for retrying
		// the transaction with exponential backoff if it fails for transient errors
		err := crdb.ExecuteTx(ctxWithTimeout, db.DB, nil, func(tx *sql.Tx) error {
			// Step 1
			// Select and lock the ip address rows that may be updated or deleted by this operation
			// to prevent race conditions. This includes:
			// * ip addresses that already exist for this instance id (instanceIPAddresses)
			// * ip addresses included in this update request, but are associated with a different instance id (conflictIPs)
			var queryMods []qm.QueryMod
			queryMods = append(queryMods, qm.For("UPDATE"))

			if len(ipAddresses) > 0 {
				// If we have IP addresses to look up, we'll want to lock rows for these conflictIPs as well
				queryMods = append(queryMods,
					qm.Where("instance_id = ? OR address = ANY(?::inet[])",
						id, pq.Array(ipAddresses),
					),
				)
			} else {
				// If we don't have any IP addresses to look up, we'll just lock rows for this instance ID
				queryMods = append(queryMods,
					qm.Where("instance_id = ?", id),
				)
			}

			// Perform the select and lock query
			_, err := models.InstanceIPAddresses(queryMods...).All(ctxWithTimeout, db)
			if err != nil {
				logger.Sugar().Error("doUpsert ", upsertID, " DB error when selecting and locking instance_ip_address rows for update. Instance uuid: ", id, " Error: ", err)
				return err
			}

			// Now save the two segments of that query as separate vars
			instanceIPAddresses, err := models.InstanceIPAddresses(models.InstanceIPAddressWhere.InstanceID.EQ(id)).All(ctxWithTimeout, tx)
			if err != nil {
				logger.Sugar().Error("doUpsert ", upsertID, " DB error when selecting instanceIPAddresses for update (post-lock). Instance uuid: ", id, " Error: ", err)
				return err
			}

			conflictIPs, err := models.InstanceIPAddresses(models.InstanceIPAddressWhere.Address.IN(ipAddresses), models.InstanceIPAddressWhere.InstanceID.NEQ(id)).All(ctxWithTimeout, tx)
			if err != nil {
				logger.Sugar().Error("doUpsert ", upsertID, " DB error when selecting conflictIPs for update (post-lock). Instance uuid: ", id, " IP Addresses: ", ipAddresses, " Error: ", err)
				return err
			}

			// Step 2.a
			// Find "stale" InstanceIPAddress rows for this instance. That is, select
			// rows from the instanceIPAddresses result which don't have a corresponding
			// entry in the list of IP Addresses supplied in the call.
			var staleInstanceIPAddresses models.InstanceIPAddressSlice

			for _, instanceIP := range instanceIPAddresses {
				found := false

				for _, IP := range ipAddresses {
					if strings.EqualFold(instanceIP.Address, IP) {
						found = true
						break
					}
				}

				if !found {
					logger.Sugar().Info("doUpsert ", upsertID, " found stale instanceIP row for instance uuid: ", id, " IP: ", instanceIP.Address)
					staleInstanceIPAddresses = append(staleInstanceIPAddresses, instanceIP)
				}
			}

			// Step 2.b
			// Find new IP Addresses that were specified in the call that aren't
			// currently associated to the instance.
			var newInstanceIPAddresses models.InstanceIPAddressSlice

			for _, IP := range ipAddresses {
				found := false

				for _, instanceIP := range instanceIPAddresses {
					if strings.EqualFold(IP, instanceIP.Address) {
						found = true
						break
					}
				}

				if !found {
					logger.Sugar().Info("doUpsert ", upsertID, " found new instanceIP for instance uuid: ", id, " IP: ", IP)
					newRecord := &models.InstanceIPAddress{
						InstanceID: id,
						Address:    IP,
					}
					newInstanceIPAddresses = append(newInstanceIPAddresses, newRecord)
				}
			}

			// Step 3
			// Remove any instance_ip_address rows for the specified IP addresses that
			// are currently associated to a *different* instance ID
			for _, conflictingIP := range conflictIPs {
				logger.Sugar().Info("doUpsert ", upsertID, " deleting conflictIP row for instance uuid: ", id, " IP: ", conflictingIP.Address)

				// TODO: Maybe remove instance_metadata and instance_userdata records for the "old" instance ID(s)?
				_, err := conflictingIP.Delete(ctxWithTimeout, tx)
				if err != nil {
					logger.Sugar().Error("doUpsert ", upsertID, " DB error when deleting conflictIPs. Instance uuid: ", id, " conflicting IP: ", conflictingIP, " Error: ", err)

					return err
				}
			}

			// Step 4
			// Remove any "stale" instance_ip_addresses rows associated to the provided
			// instnace_id but were not specified in the call.
			for _, staleIP := range staleInstanceIPAddresses {
				logger.Sugar().Info("doUpsert ", upsertID, " deleting stale instanceIP row for instance uuid: ", id, " IP: ", staleIP.Address)

				_, err := staleIP.Delete(ctxWithTimeout, tx)
				if err != nil {
					logger.Sugar().Error("doUpsert ", upsertID, " DB error when deleting staleIPs. Instance uuid: ", id, " staleIP: ", staleIP, " Error: ", err)

					return err
				}
			}

			// Step 5
			// Create instance_ip_addresses rows for any IP addresses specified in the
			// call that aren't already associated to the provided instance_id
			for _, newInstanceIP := range newInstanceIPAddresses {
				err := newInstanceIP.Insert(ctxWithTimeout, tx, boil.Infer())
				if err != nil {
					logger.Sugar().Error("doUpsert ", upsertID, " DB error when inserting newInstanceIPs. Instance uuid: ", id, " newInstanceIP: ", " Error: ", err)

					return err
				}
			}

			// Step 6
			// Upsert the instance_metadata or instance_userdata table. This will create
			// a new row with the provided instance ID and metadata or userdata if there
			// is no current row for instance_id. If there is an existing row matching on
			// instance_id, instead this will just update the metadata or userdata column
			// value.
			if err := upsertRecordFunc(ctxWithTimeout, tx); err != nil {
				logger.Sugar().Error("doUpsert ", upsertID, " DB error when upserting the instance_metadata or instance_userdata table for instance uuid: ", id, " Error: ", err)

				return err
			}

			return nil
		})

		// Cancel the context to ensure the transaction is cleaned up
		cancel()

		if err == nil {
			logger.Sugar().Info("doUpsert ", upsertID, " successful on retry ", retryCount, " for instance uuid: ", id)
			return nil
		}

		logger.Sugar().Error("doUpsert ", upsertID, " DB error on retry ", retryCount, " when executing the transaction. Instance uuid: ", id, " Error: ", err)

		if retryCount >= maxRetries {
			logger.Sugar().Error("doUpsert ", upsertID, " failed for instance uuid: ", id, " even after ", maxRetries, " attempts")
			return err
		}

		// Check if the parent context has been cancelled before retrying
		if ctx.Err() != nil {
			logger.Sugar().Error("doUpsert ", upsertID, " parent context cancelled for instance uuid: ", id, " - aborting retry")
			return ctx.Err()
		}

		retryCount++
		logger.Sugar().Warn("doUpsert ", upsertID, " retrying upsert for instance uuid: ", id, " on attempt ", retryCount)
	}
}
