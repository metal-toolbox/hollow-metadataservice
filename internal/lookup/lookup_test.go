package lookup_test

import (
	"context"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"go.hollow.sh/metadataservice/internal/dbtools"
	"go.hollow.sh/metadataservice/internal/lookup"
)

type mockLookupClient struct {
	MetadataResponse lookup.MetadataLookupResponse
	UserdataResponse lookup.UserdataLookupResponse
	Error            error
}

func (m *mockLookupClient) GetMetadataByID(_ context.Context, _ string) (*lookup.MetadataLookupResponse, error) {
	return &m.MetadataResponse, m.Error
}

func (m *mockLookupClient) GetMetadataByIP(_ context.Context, _ string) (*lookup.MetadataLookupResponse, error) {
	return &m.MetadataResponse, m.Error
}

func (m *mockLookupClient) GetUserdataByID(_ context.Context, _ string) (*lookup.UserdataLookupResponse, error) {
	return &m.UserdataResponse, m.Error
}

func (m *mockLookupClient) GetUserdataByIP(_ context.Context, _ string) (*lookup.UserdataLookupResponse, error) {
	return &m.UserdataResponse, m.Error
}

type testInstance struct {
	ID          string
	IPAddresses []string
	Metadata    string
	Userdata    []byte
}

func (ti *testInstance) MetadataResponse() lookup.MetadataLookupResponse {
	return lookup.MetadataLookupResponse{
		ID:          ti.ID,
		IPAddresses: ti.IPAddresses,
		Metadata:    ti.Metadata,
	}
}

func (ti *testInstance) UserdataResponse() lookup.UserdataLookupResponse {
	return lookup.UserdataLookupResponse{
		ID:          ti.ID,
		IPAddresses: ti.IPAddresses,
		Userdata:    ti.Userdata,
	}
}

var testInstances = []testInstance{
	{
		ID:          "8dcf8e72-48a4-415c-afe2-6ba67bd2c956",
		IPAddresses: []string{"1.2.3.4"},
		Metadata:    `{"some":"metadata for instance 0"}`,
		Userdata:    []byte("some userdata for instance 0"),
	},
	{
		ID:          "7db4d541-a669-4860-9b32-b7f17fbe4136",
		IPAddresses: []string{"9.8.7.6"},
		Metadata:    `{"some":"metadata for instance 1"}`,
		Userdata:    []byte("some userdata for instance 1"),
	},
}

func TestFetchMetadataByIDAndStoreNilClient(t *testing.T) {
	testDB := dbtools.DatabaseTest(t)
	metadata, err := lookup.MetadataSyncByID(context.TODO(), testDB, zap.NewNop(), nil, "abc123")
	assert.NotNil(t, err)
	assert.Equal(t, "client can't be nil", err.Error())
	assert.Nil(t, metadata)
}

func TestFetchMetadataByIDAndStore(t *testing.T) {
	type testCase struct {
		ID               string
		ResponseError    error
		MetadataResponse lookup.MetadataLookupResponse
	}

	viper.SetDefault("crdb.max_retries", 5)
	viper.SetDefault("crdb.retry_interval", 1*time.Second)
	viper.SetDefault("crdb.tx_timeout", 15*time.Second)

	var testCases = []testCase{
		{
			ID:            "abc123",
			ResponseError: lookup.ErrNotFound,
		},
		{
			ID:            "def456",
			ResponseError: lookup.ErrUnexpectedStatus,
		},
		{
			ID:               testInstances[0].ID,
			MetadataResponse: testInstances[0].MetadataResponse(),
		},
		{
			ID:               testInstances[1].ID,
			MetadataResponse: testInstances[1].MetadataResponse(),
		},
	}

	testDB := dbtools.DatabaseTest(t)

	for _, tc := range testCases {
		mockClient := mockLookupClient{
			MetadataResponse: tc.MetadataResponse,
			Error:            tc.ResponseError,
		}

		metadata, err := lookup.MetadataSyncByID(context.TODO(), testDB, zap.NewNop(), &mockClient, tc.ID)
		if tc.ResponseError != nil {
			assert.NotNil(t, err)
			assert.ErrorIs(t, err, tc.ResponseError)
		} else {
			assert.Nil(t, err)
			assert.NotNil(t, metadata)
			assert.Equal(t, metadata.ID, tc.ID)
		}
	}
}

func TestFetchMetadataByIPAndStoreNilClient(t *testing.T) {
	testDB := dbtools.DatabaseTest(t)
	metadata, err := lookup.MetadataSyncByIP(context.TODO(), testDB, zap.NewNop(), nil, "1.2.3.4")
	assert.NotNil(t, err)
	assert.Equal(t, "client can't be nil", err.Error())
	assert.Nil(t, metadata)
}

func TestFetchMetadataByIPAndStore(t *testing.T) {
	type testCase struct {
		ID               string
		IPAddress        string
		ResponseError    error
		MetadataResponse lookup.MetadataLookupResponse
	}

	var testCases = []testCase{
		{
			IPAddress:     "1.2.3.4",
			ResponseError: lookup.ErrNotFound,
		},
		{
			ID:            "def456",
			ResponseError: lookup.ErrUnexpectedStatus,
		},
	}

	// Add tests for each testInstance IP address
	for _, testInstance := range testInstances {
		for _, ip := range testInstance.IPAddresses {
			tc := testCase{
				ID:               testInstance.ID,
				IPAddress:        ip,
				MetadataResponse: testInstance.MetadataResponse(),
			}

			testCases = append(testCases, tc)
		}
	}

	testDB := dbtools.DatabaseTest(t)

	for _, tc := range testCases {
		mockClient := mockLookupClient{
			MetadataResponse: tc.MetadataResponse,
			Error:            tc.ResponseError,
		}

		metadata, err := lookup.MetadataSyncByIP(context.TODO(), testDB, zap.NewNop(), &mockClient, tc.IPAddress)
		if tc.ResponseError != nil {
			assert.NotNil(t, err)
			assert.ErrorIs(t, err, tc.ResponseError)
		} else {
			assert.Nil(t, err)
			assert.NotNil(t, metadata)
			assert.Equal(t, metadata.ID, tc.ID)
		}
	}
}

func TestFetchUserdataByIDAndStoreNilClient(t *testing.T) {
	testDB := dbtools.DatabaseTest(t)
	userdata, err := lookup.UserdataSyncByID(context.TODO(), testDB, zap.NewNop(), nil, "abc123")
	assert.NotNil(t, err)
	assert.Equal(t, "client can't be nil", err.Error())
	assert.Nil(t, userdata)
}

func TestFetchUserdataByIDAndStore(t *testing.T) {
	type testCase struct {
		ID               string
		ResponseError    error
		UserdataResponse lookup.UserdataLookupResponse
	}

	var testCases = []testCase{
		{
			ID:            "abc123",
			ResponseError: lookup.ErrNotFound,
		},
		{
			ID:            "def456",
			ResponseError: lookup.ErrUnexpectedStatus,
		},
		{
			ID:               testInstances[0].ID,
			UserdataResponse: testInstances[0].UserdataResponse(),
		},
		{
			ID:               testInstances[1].ID,
			UserdataResponse: testInstances[1].UserdataResponse(),
		},
	}

	testDB := dbtools.DatabaseTest(t)

	for _, tc := range testCases {
		mockClient := mockLookupClient{
			UserdataResponse: tc.UserdataResponse,
			Error:            tc.ResponseError,
		}

		userdata, err := lookup.UserdataSyncByID(context.TODO(), testDB, zap.NewNop(), &mockClient, tc.ID)
		if tc.ResponseError != nil {
			assert.NotNil(t, err)
			assert.ErrorIs(t, err, tc.ResponseError)
		} else {
			assert.Nil(t, err)
			assert.NotNil(t, userdata)
			assert.Equal(t, userdata.ID, tc.ID)
		}
	}
}

func TestFetchUserdataByIPAndStoreNilClient(t *testing.T) {
	testDB := dbtools.DatabaseTest(t)
	userdata, err := lookup.UserdataSyncByIP(context.TODO(), testDB, zap.NewNop(), nil, "1.2.3.4")
	assert.NotNil(t, err)
	assert.Equal(t, "client can't be nil", err.Error())
	assert.Nil(t, userdata)
}

func TestFetchUserdataByIPAndStore(t *testing.T) {
	type testCase struct {
		ID               string
		IPAddress        string
		ResponseError    error
		UserdataResponse lookup.UserdataLookupResponse
	}

	var testCases = []testCase{
		{
			IPAddress:     "1.2.3.4",
			ResponseError: lookup.ErrNotFound,
		},
		{
			ID:            "def456",
			ResponseError: lookup.ErrUnexpectedStatus,
		},
	}

	// Add tests for each testInstance IP address
	for _, testInstance := range testInstances {
		for _, ip := range testInstance.IPAddresses {
			tc := testCase{
				ID:               testInstance.ID,
				IPAddress:        ip,
				UserdataResponse: testInstance.UserdataResponse(),
			}

			testCases = append(testCases, tc)
		}
	}

	testDB := dbtools.DatabaseTest(t)

	for _, tc := range testCases {
		mockClient := mockLookupClient{
			UserdataResponse: tc.UserdataResponse,
			Error:            tc.ResponseError,
		}

		userdata, err := lookup.UserdataSyncByIP(context.TODO(), testDB, zap.NewNop(), &mockClient, tc.IPAddress)
		if tc.ResponseError != nil {
			assert.NotNil(t, err)
			assert.ErrorIs(t, err, tc.ResponseError)
		} else {
			assert.Nil(t, err)
			assert.NotNil(t, userdata)
			assert.Equal(t, userdata.ID, tc.ID)
		}
	}
}
