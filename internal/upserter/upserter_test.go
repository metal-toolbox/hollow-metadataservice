package upserter_test

import (
	"context"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/types"
	"go.uber.org/zap"

	"go.hollow.sh/metadataservice/internal/dbtools"
	"go.hollow.sh/metadataservice/internal/models"
	"go.hollow.sh/metadataservice/internal/upserter"
)

var (
	instanceID              = "22bc79fc-3834-40b8-b734-30bef9634939"
	instanceIPs             = []string{"1.2.3.4", "1f00:1f00:1f00:1f00::9/127"}
	instanceMetadata0       = `{"some":"metadata"}`
	instanceMetadata1       = `{"some":"updated metadata"}`
	instanceMetadataWithIPs = `{"some":"metadata", "updated_at": "2023-04-12T15:30:45.123Z", "network": {"addresses": [ { "address": "111.22.113.114", "address_family": 4, "cidr": 31, "enabled": true, "id": "d5a8e6e0-137d-4a74-b735-4b9fe3b66e7f" }, { "address": "2604:1380:4631:2600::3", "address_family": 6, "cidr": 127, "id": "0c12dee-19eb-4e23-84e8-978cb375a561" } ] } }`
	instanceMetadataV1      = `{"hostname": "version1", "updated_at": "2025-01-15T15:30:30:111Z"}`
	instanceMetadataV2      = `{"hostname": "version2", "updated_at": "2025-01-15T15:30:30:222Z"}`
	instanceMetadataV3      = `{"hostname": "version3", "updated_at": "2025-01-15T15:30:30:333Z"}`
	instanceUserdata0       = "some userdata..."
	instanceUserdata1       = "some updated userdata..."
)

// Test that we can parse IP addresses from metadata
func TestExtractIPAddressesFromMetadata(t *testing.T) {
	metadata := models.InstanceMetadatum(
		models.InstanceMetadatum{
			ID:       instanceID,
			Metadata: types.JSON(instanceMetadataWithIPs),
		},
	)

	ips := upserter.ExtractIPAddressesFromMetadata(&metadata)

	assert.Equal(t, 2, len(ips))
	assert.Equal(t, "111.22.113.114", ips[0])
	assert.Equal(t, "2604:1380:4631:2600::3", ips[1])
}

// Test that we can parse updated_at from metadata
func TestExtractUpdatedAtFromMetadata(t *testing.T) {
	metadataWithoutUpdatedAt := models.InstanceMetadatum(
		models.InstanceMetadatum{
			ID:       instanceID,
			Metadata: types.JSON(instanceMetadata0),
		},
	)

	metadataWithUpdatedAt := models.InstanceMetadatum(
		models.InstanceMetadatum{
			ID:       instanceID,
			Metadata: types.JSON(instanceMetadataWithIPs),
		},
	)

	noUpdatedAt := upserter.ExtractUpdatedAtFromMetadata(&metadataWithoutUpdatedAt)

	assert.Empty(t, noUpdatedAt)

	withUpdatedAt := upserter.ExtractUpdatedAtFromMetadata(&metadataWithUpdatedAt)

	assert.NotEmpty(t, withUpdatedAt)
	assert.Equal(t, "2023-04-12T15:30:45.123Z", withUpdatedAt)
}

// Test that nothing fails when there are no IP addresses in the metadata document to parse
func TestExtractIPAddressesFromMetadataWithoutIPs(t *testing.T) {
	metadata := models.InstanceMetadatum(
		models.InstanceMetadatum{
			ID:       instanceID,
			Metadata: types.JSON(instanceMetadata0),
		},
	)

	ips := upserter.ExtractIPAddressesFromMetadata(&metadata)

	assert.Equal(t, 0, len(ips))
	assert.Nil(t, ips)
}

// Test that upsert metadata adds a new instance_metadata row to the DB
func TestUpsertMetadataAddsInstanceMetadataRow(t *testing.T) {
	testDB := dbtools.DatabaseTest(t)

	viper.SetDefault("crdb.max_retries", 5)
	viper.SetDefault("crdb.retry_interval", 1*time.Second)
	viper.SetDefault("crdb.tx_timeout", 15*time.Second)

	exists, err := models.InstanceMetadatumExists(context.TODO(), testDB, instanceID)
	if err != nil {
		t.Fatal(err)
	}

	assert.False(t, exists)

	metadata := models.InstanceMetadatum{
		ID:       instanceID,
		Metadata: types.JSON(instanceMetadata0),
	}

	err = upserter.UpsertMetadata(context.TODO(), testDB, zap.NewNop(), instanceID, instanceIPs, &metadata)
	assert.Nil(t, err)

	exists, err = models.InstanceMetadatumExists(context.TODO(), testDB, instanceID)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, exists)
}

// Test that upsert metadata adds new instance_ip_addresses rows to the DB
func TestUpsertMetadataAddsInstanceIPAddressesRows(t *testing.T) {
	testDB := dbtools.DatabaseTest(t)
	metadata := models.InstanceMetadatum{
		ID:       instanceID,
		Metadata: types.JSON(instanceMetadata0),
	}

	instanceIPAddressesCount, err := models.InstanceIPAddresses().Count(context.TODO(), testDB)
	if err != nil {
		t.Fatal(err)
	}

	err = upserter.UpsertMetadata(context.TODO(), testDB, zap.NewNop(), instanceID, instanceIPs, &metadata)
	assert.Nil(t, err)

	newInstanceIPAddressesCount, err := models.InstanceIPAddresses().Count(context.TODO(), testDB)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, instanceIPAddressesCount+2, newInstanceIPAddressesCount)
}

// Test that upsert metadata updates the instance_metadata row and removes any
// "stale" instance_ip_addresses rows. "Stale" IPs are addresses that were
// previously associated to the instance, but weren't included in a subsequent
// update.
func TestUpsertMetadataRemovesStaleInstanceIPAddressesRows(t *testing.T) {
	testDB := dbtools.DatabaseTest(t)
	metadataInsert := models.InstanceMetadatum{
		ID:       instanceID,
		Metadata: types.JSON(instanceMetadata0),
	}

	metadataUpdate := models.InstanceMetadatum{
		ID:       instanceID,
		Metadata: types.JSON(instanceMetadata1),
	}

	// Insert the metadata record
	err := upserter.UpsertMetadata(context.TODO(), testDB, zap.NewNop(), instanceID, instanceIPs, &metadataInsert)
	assert.Nil(t, err)

	// Check that 2 instance_ip_addresses rows were created
	instanceIPAddresses, err := models.InstanceIPAddresses(models.InstanceIPAddressWhere.InstanceID.EQ(instanceID)).All(context.TODO(), testDB)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 2, len(instanceIPAddresses))

	// Update the metadata record
	newIPs := instanceIPs[:1]
	err = upserter.UpsertMetadata(context.TODO(), testDB, zap.NewNop(), instanceID, newIPs, &metadataUpdate)
	assert.Nil(t, err)

	// Check that now there is just 1 instance_ip_address row associated to the instance
	instanceIPAddresses, err = models.InstanceIPAddresses(models.InstanceIPAddressWhere.InstanceID.EQ(instanceID)).All(context.TODO(), testDB)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, len(instanceIPAddresses))
}

// Test that an upsert metadata call including IP Addresses already associated
// to another instance ID causes the "old" rows to be removed in favor of new
// rows for the new instance.
func TestUpsertMetadataRemovesConflictingIPAddressesRows(t *testing.T) {
	testDB := dbtools.DatabaseTest(t)
	// Create an "old" record.
	oldID := "1f36c15b-b3ef-45da-b7e8-f434287e2f03"
	oldMetadata := models.InstanceMetadatum{
		ID:       oldID,
		Metadata: types.JSON(`{"old":"metadata"}`),
	}

	err := upserter.UpsertMetadata(context.TODO(), testDB, zap.NewNop(), oldID, instanceIPs, &oldMetadata)
	if err != nil {
		t.Fatal(err)
	}

	// Verify there's 2 instance_ip_addresses associated to the "old" instance ID
	oldInstanceIPAddresses, err := models.InstanceIPAddresses(models.InstanceIPAddressWhere.InstanceID.EQ(oldID)).All(context.TODO(), testDB)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 2, len(oldInstanceIPAddresses))

	// Now upsert a new metadata record for a new instance, but with the same 2 IP addresses
	newMetadata := models.InstanceMetadatum{
		ID:       instanceID,
		Metadata: types.JSON(instanceMetadata0),
	}

	err = upserter.UpsertMetadata(context.TODO(), testDB, zap.NewNop(), instanceID, instanceIPs, &newMetadata)
	if err != nil {
		t.Fatal(err)
	}

	// Verify there's 2 instance_ip_addresses associated to the "new" instance ID
	newInstanceIPAddresses, err := models.InstanceIPAddresses(models.InstanceIPAddressWhere.InstanceID.EQ(instanceID)).All(context.TODO(), testDB)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 2, len(newInstanceIPAddresses))

	// And verify that there's now 0 instance_ip_addresses associated to the "old" instance ID
	oldInstanceIPAddresses, err = models.InstanceIPAddresses(models.InstanceIPAddressWhere.InstanceID.EQ(oldID)).All(context.TODO(), testDB)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 0, len(oldInstanceIPAddresses))
}

// Test that stale metadata updates are ignored
func TestStaleMetadataUpdatesAreIgnored(t *testing.T) {
	testDB := dbtools.DatabaseTest(t)

	viper.SetDefault("crdb.max_retries", 5)
	viper.SetDefault("crdb.retry_interval", 1*time.Second)
	viper.SetDefault("crdb.tx_timeout", 15*time.Second)

	exists, err := models.InstanceMetadatumExists(context.TODO(), testDB, instanceID)
	if err != nil {
		t.Fatal(err)
	}

	assert.False(t, exists)

	// We'll upsert metadata in the order V2 -> V3 -> V1, and the failure should
	// happen on the V1 upsert
	metadata := models.InstanceMetadatum{
		ID:       instanceID,
		Metadata: types.JSON(instanceMetadataV2),
	}

	metadataUpdate := models.InstanceMetadatum{
		ID:       instanceID,
		Metadata: types.JSON(instanceMetadataV3),
	}

	staleMetadata := models.InstanceMetadatum{
		ID:       instanceID,
		Metadata: types.JSON(instanceMetadataV1),
	}

	// Upsert V2 metadata
	err = upserter.UpsertMetadata(context.TODO(), testDB, zap.NewNop(), instanceID, instanceIPs, &metadata)
	assert.Nil(t, err)

	exists, err = models.InstanceMetadatumExists(context.TODO(), testDB, instanceID)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, exists)

	mv2, err := models.InstanceMetadata(models.InstanceMetadatumWhere.ID.EQ(instanceID)).One(context.TODO(), testDB)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, instanceMetadataV2, mv2.Metadata.String())

	// Upsert V3 metadata
	err = upserter.UpsertMetadata(context.TODO(), testDB, zap.NewNop(), instanceID, instanceIPs, &metadataUpdate)
	assert.Nil(t, err)

	exists, err = models.InstanceMetadatumExists(context.TODO(), testDB, instanceID)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, exists)

	mv3, err := models.InstanceMetadata(models.InstanceMetadatumWhere.ID.EQ(instanceID)).One(context.TODO(), testDB)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, instanceMetadataV3, mv3.Metadata.String())

	// Attempt to update with stale metadata (V1)
	err = upserter.UpsertMetadata(context.TODO(), testDB, zap.NewNop(), instanceID, instanceIPs, &staleMetadata)
	if err != nil {
		t.Fatal(err)
	}

	// Verify that the metadata was not updated
	mv1, err := models.InstanceMetadata(models.InstanceMetadatumWhere.ID.EQ(instanceID)).One(context.TODO(), testDB)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, instanceMetadataV3, mv1.Metadata.String())
}

// Test that upsert userdata adds a new instance_userdata row to the DB
func TestUpsertUserdataAddsInstanceUserdataRow(t *testing.T) {
	testDB := dbtools.DatabaseTest(t)

	exists, err := models.InstanceUserdatumExists(context.TODO(), testDB, instanceID)
	if err != nil {
		t.Fatal(err)
	}

	assert.False(t, exists)

	userdata := models.InstanceUserdatum{
		ID:       instanceID,
		Userdata: null.NewBytes([]byte(instanceUserdata0), true),
	}

	err = upserter.UpsertUserdata(context.TODO(), testDB, zap.NewNop(), instanceID, instanceIPs, &userdata)
	assert.Nil(t, err)

	exists, err = models.InstanceUserdatumExists(context.TODO(), testDB, instanceID)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, exists)
}

// Test that upsert userdata adds new instance_ip_addresses rows to the DB
func TestUpsertUserdataAddsInstanceIPAddressesRows(t *testing.T) {
	testDB := dbtools.DatabaseTest(t)
	userdata := models.InstanceUserdatum{
		ID:       instanceID,
		Userdata: null.NewBytes([]byte(instanceUserdata0), true),
	}

	instanceIPAddressesCount, err := models.InstanceIPAddresses().Count(context.TODO(), testDB)
	if err != nil {
		t.Fatal(err)
	}

	err = upserter.UpsertUserdata(context.TODO(), testDB, zap.NewNop(), instanceID, instanceIPs, &userdata)
	assert.Nil(t, err)

	newInstanceIPAddressesCount, err := models.InstanceIPAddresses().Count(context.TODO(), testDB)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, instanceIPAddressesCount+2, newInstanceIPAddressesCount)
}

// Test that upsert userdata updates the instance_userdata row and removes any
// "stale" instance_ip_addresses rows. "Stale" IPs are addresses that were
// previously associated to the instance, but weren't included in a subsequent
// update.
func TestUpsertUserdataRemovesStaleInstanceIPAddressesRows(t *testing.T) {
	testDB := dbtools.DatabaseTest(t)
	userdataInsert := models.InstanceUserdatum{
		ID:       instanceID,
		Userdata: null.NewBytes([]byte(instanceUserdata0), true),
	}

	userdataUpdate := models.InstanceUserdatum{
		ID:       instanceID,
		Userdata: null.NewBytes([]byte(instanceUserdata1), true),
	}

	// Insert the userdata record
	err := upserter.UpsertUserdata(context.TODO(), testDB, zap.NewNop(), instanceID, instanceIPs, &userdataInsert)
	assert.Nil(t, err)

	// Check that 2 instance_ip_addresses rows were created
	instanceIPAddresses, err := models.InstanceIPAddresses(models.InstanceIPAddressWhere.InstanceID.EQ(instanceID)).All(context.TODO(), testDB)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 2, len(instanceIPAddresses))

	// Update the userdata record
	newIPs := instanceIPs[:1]
	err = upserter.UpsertUserdata(context.TODO(), testDB, zap.NewNop(), instanceID, newIPs, &userdataUpdate)
	assert.Nil(t, err)

	// Check that now there is just 1 instance_ip_address row associated to the instance
	instanceIPAddresses, err = models.InstanceIPAddresses(models.InstanceIPAddressWhere.InstanceID.EQ(instanceID)).All(context.TODO(), testDB)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, len(instanceIPAddresses))
}

// Test that updating metadata results in the updated_at field being changed, as
// our TTL cache mechanism depends on this working
func TestMetadataUpsertModifiesUpdateAtField(t *testing.T) {
	testDB := dbtools.DatabaseTest(t)
	metadataInsert := models.InstanceMetadatum{
		ID:       instanceID,
		Metadata: types.JSON(instanceMetadata0),
	}

	metadataUpdate := models.InstanceMetadatum{
		ID:       instanceID,
		Metadata: types.JSON(instanceMetadata1),
	}

	// Insert the metadata record
	err := upserter.UpsertMetadata(context.TODO(), testDB, zap.NewNop(), instanceID, instanceIPs, &metadataInsert)
	assert.Nil(t, err)

	m1, err := models.InstanceMetadata(models.InstanceMetadatumWhere.ID.EQ(instanceID)).One(context.TODO(), testDB)
	if err != nil {
		t.Fatal(err)
	}

	// Update the metadata record
	err = upserter.UpsertMetadata(context.TODO(), testDB, zap.NewNop(), instanceID, instanceIPs, &metadataUpdate)
	assert.Nil(t, err)

	m2, err := models.InstanceMetadata(models.InstanceMetadatumWhere.ID.EQ(instanceID)).One(context.TODO(), testDB)
	if err != nil {
		t.Fatal(err)
	}

	assert.NotEqual(t, m1.UpdatedAt, m2.UpdatedAt)
}

// Test that updating userdata results in the updated_at field being changed, as
// our TTL cache mechanism depends on this working
func TestUserdataUpsertModifiesUpdateAtField(t *testing.T) {
	testDB := dbtools.DatabaseTest(t)
	userdataInsert := models.InstanceUserdatum{
		ID:       instanceID,
		Userdata: null.NewBytes([]byte(instanceUserdata0), true),
	}

	userdataUpdate := models.InstanceUserdatum{
		ID:       instanceID,
		Userdata: null.NewBytes([]byte(instanceUserdata1), true),
	}

	// Insert the userdata record
	err := upserter.UpsertUserdata(context.TODO(), testDB, zap.NewNop(), instanceID, instanceIPs, &userdataInsert)
	assert.Nil(t, err)

	u1, err := models.InstanceUserdata(models.InstanceUserdatumWhere.ID.EQ(instanceID)).One(context.TODO(), testDB)
	if err != nil {
		t.Fatal(err)
	}

	// Update the userdata record
	err = upserter.UpsertUserdata(context.TODO(), testDB, zap.NewNop(), instanceID, instanceIPs, &userdataUpdate)
	assert.Nil(t, err)

	u2, err := models.InstanceUserdata(models.InstanceUserdatumWhere.ID.EQ(instanceID)).One(context.TODO(), testDB)
	if err != nil {
		t.Fatal(err)
	}

	assert.NotEqual(t, u1.UpdatedAt, u2.UpdatedAt)
}

// Test that an upsert userdata call including IP Addresses already associated
// to another instance ID causes the "old" rows to be removed in favor of new
// rows for the new instance.
func TestUpsertUserdataRemovesConflictingIPAddressesRows(t *testing.T) {
	testDB := dbtools.DatabaseTest(t)
	// Create an "old" record.
	oldID := "1f36c15b-b3ef-45da-b7e8-f434287e2f03"
	oldMetadata := models.InstanceMetadatum{
		ID:       oldID,
		Metadata: types.JSON(`{"old":"metadata"}`),
	}

	err := upserter.UpsertMetadata(context.TODO(), testDB, zap.NewNop(), oldID, instanceIPs, &oldMetadata)
	if err != nil {
		t.Fatal(err)
	}

	// Verify there's 2 instance_ip_addresses associated to the "old" instance ID
	oldInstanceIPAddresses, err := models.InstanceIPAddresses(models.InstanceIPAddressWhere.InstanceID.EQ(oldID)).All(context.TODO(), testDB)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 2, len(oldInstanceIPAddresses))

	// Now upsert a new userdata record for a new instance, but with the same 2 IP addresses
	newUserdata := models.InstanceUserdatum{
		ID:       instanceID,
		Userdata: null.NewBytes([]byte(instanceUserdata0), true),
	}

	err = upserter.UpsertUserdata(context.TODO(), testDB, zap.NewNop(), instanceID, instanceIPs, &newUserdata)
	if err != nil {
		t.Fatal(err)
	}

	// Verify there's 2 instance_ip_addresses associated to the "new" instance ID
	newInstanceIPAddresses, err := models.InstanceIPAddresses(models.InstanceIPAddressWhere.InstanceID.EQ(instanceID)).All(context.TODO(), testDB)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 2, len(newInstanceIPAddresses))

	// And verify that there's now 0 instance_ip_addresses associated to the "old" instance ID
	oldInstanceIPAddresses, err = models.InstanceIPAddresses(models.InstanceIPAddressWhere.InstanceID.EQ(oldID)).All(context.TODO(), testDB)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 0, len(oldInstanceIPAddresses))
}
