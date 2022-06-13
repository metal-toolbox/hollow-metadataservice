// Code generated by SQLBoiler 4.11.0 (https://github.com/volatiletech/sqlboiler). DO NOT EDIT.
// This file is meant to be re-generated in place and/or deleted at any time.

package models

import "testing"

// This test suite runs each operation test in parallel.
// Example, if your database has 3 tables, the suite will run:
// table1, table2 and table3 Delete in parallel
// table1, table2 and table3 Insert in parallel, and so forth.
// It does NOT run each operation group in parallel.
// Separating the tests thusly grants avoidance of Postgres deadlocks.
func TestParent(t *testing.T) {
	t.Run("InstanceIPAddresses", testInstanceIPAddresses)
	t.Run("InstanceMetadata", testInstanceMetadata)
	t.Run("InstanceUserdata", testInstanceUserdata)
}

func TestDelete(t *testing.T) {
	t.Run("InstanceIPAddresses", testInstanceIPAddressesDelete)
	t.Run("InstanceMetadata", testInstanceMetadataDelete)
	t.Run("InstanceUserdata", testInstanceUserdataDelete)
}

func TestQueryDeleteAll(t *testing.T) {
	t.Run("InstanceIPAddresses", testInstanceIPAddressesQueryDeleteAll)
	t.Run("InstanceMetadata", testInstanceMetadataQueryDeleteAll)
	t.Run("InstanceUserdata", testInstanceUserdataQueryDeleteAll)
}

func TestSliceDeleteAll(t *testing.T) {
	t.Run("InstanceIPAddresses", testInstanceIPAddressesSliceDeleteAll)
	t.Run("InstanceMetadata", testInstanceMetadataSliceDeleteAll)
	t.Run("InstanceUserdata", testInstanceUserdataSliceDeleteAll)
}

func TestExists(t *testing.T) {
	t.Run("InstanceIPAddresses", testInstanceIPAddressesExists)
	t.Run("InstanceMetadata", testInstanceMetadataExists)
	t.Run("InstanceUserdata", testInstanceUserdataExists)
}

func TestFind(t *testing.T) {
	t.Run("InstanceIPAddresses", testInstanceIPAddressesFind)
	t.Run("InstanceMetadata", testInstanceMetadataFind)
	t.Run("InstanceUserdata", testInstanceUserdataFind)
}

func TestBind(t *testing.T) {
	t.Run("InstanceIPAddresses", testInstanceIPAddressesBind)
	t.Run("InstanceMetadata", testInstanceMetadataBind)
	t.Run("InstanceUserdata", testInstanceUserdataBind)
}

func TestOne(t *testing.T) {
	t.Run("InstanceIPAddresses", testInstanceIPAddressesOne)
	t.Run("InstanceMetadata", testInstanceMetadataOne)
	t.Run("InstanceUserdata", testInstanceUserdataOne)
}

func TestAll(t *testing.T) {
	t.Run("InstanceIPAddresses", testInstanceIPAddressesAll)
	t.Run("InstanceMetadata", testInstanceMetadataAll)
	t.Run("InstanceUserdata", testInstanceUserdataAll)
}

func TestCount(t *testing.T) {
	t.Run("InstanceIPAddresses", testInstanceIPAddressesCount)
	t.Run("InstanceMetadata", testInstanceMetadataCount)
	t.Run("InstanceUserdata", testInstanceUserdataCount)
}

func TestHooks(t *testing.T) {
	t.Run("InstanceIPAddresses", testInstanceIPAddressesHooks)
	t.Run("InstanceMetadata", testInstanceMetadataHooks)
	t.Run("InstanceUserdata", testInstanceUserdataHooks)
}

func TestInsert(t *testing.T) {
	t.Run("InstanceIPAddresses", testInstanceIPAddressesInsert)
	t.Run("InstanceIPAddresses", testInstanceIPAddressesInsertWhitelist)
	t.Run("InstanceMetadata", testInstanceMetadataInsert)
	t.Run("InstanceMetadata", testInstanceMetadataInsertWhitelist)
	t.Run("InstanceUserdata", testInstanceUserdataInsert)
	t.Run("InstanceUserdata", testInstanceUserdataInsertWhitelist)
}

// TestToOne tests cannot be run in parallel
// or deadlocks can occur.
func TestToOne(t *testing.T) {}

// TestOneToOne tests cannot be run in parallel
// or deadlocks can occur.
func TestOneToOne(t *testing.T) {}

// TestToMany tests cannot be run in parallel
// or deadlocks can occur.
func TestToMany(t *testing.T) {}

// TestToOneSet tests cannot be run in parallel
// or deadlocks can occur.
func TestToOneSet(t *testing.T) {}

// TestToOneRemove tests cannot be run in parallel
// or deadlocks can occur.
func TestToOneRemove(t *testing.T) {}

// TestOneToOneSet tests cannot be run in parallel
// or deadlocks can occur.
func TestOneToOneSet(t *testing.T) {}

// TestOneToOneRemove tests cannot be run in parallel
// or deadlocks can occur.
func TestOneToOneRemove(t *testing.T) {}

// TestToManyAdd tests cannot be run in parallel
// or deadlocks can occur.
func TestToManyAdd(t *testing.T) {}

// TestToManySet tests cannot be run in parallel
// or deadlocks can occur.
func TestToManySet(t *testing.T) {}

// TestToManyRemove tests cannot be run in parallel
// or deadlocks can occur.
func TestToManyRemove(t *testing.T) {}

func TestReload(t *testing.T) {
	t.Run("InstanceIPAddresses", testInstanceIPAddressesReload)
	t.Run("InstanceMetadata", testInstanceMetadataReload)
	t.Run("InstanceUserdata", testInstanceUserdataReload)
}

func TestReloadAll(t *testing.T) {
	t.Run("InstanceIPAddresses", testInstanceIPAddressesReloadAll)
	t.Run("InstanceMetadata", testInstanceMetadataReloadAll)
	t.Run("InstanceUserdata", testInstanceUserdataReloadAll)
}

func TestSelect(t *testing.T) {
	t.Run("InstanceIPAddresses", testInstanceIPAddressesSelect)
	t.Run("InstanceMetadata", testInstanceMetadataSelect)
	t.Run("InstanceUserdata", testInstanceUserdataSelect)
}

func TestUpdate(t *testing.T) {
	t.Run("InstanceIPAddresses", testInstanceIPAddressesUpdate)
	t.Run("InstanceMetadata", testInstanceMetadataUpdate)
	t.Run("InstanceUserdata", testInstanceUserdataUpdate)
}

func TestSliceUpdateAll(t *testing.T) {
	t.Run("InstanceIPAddresses", testInstanceIPAddressesSliceUpdateAll)
	t.Run("InstanceMetadata", testInstanceMetadataSliceUpdateAll)
	t.Run("InstanceUserdata", testInstanceUserdataSliceUpdateAll)
}