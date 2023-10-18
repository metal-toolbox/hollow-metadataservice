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
	instanceID        = "22bc79fc-3834-40b8-b734-30bef9634939"
	instanceIPs       = []string{"1.2.3.4", "1f00:1f00:1f00:1f00::9/127"}
	instanceMetadata0 = `{"some":"metadata"}`
	instanceMetadata1 = `{"some":"updated metadata"}`
	instanceUserdata0 = "some userdata..."
	instanceUserdata1 = "some updated userdata..."
)

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

// Test that upsert userdata adds a new instance_userdata row to the DB
func TestUpsertUserdataAddsInstanceMetadataRow(t *testing.T) {
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
