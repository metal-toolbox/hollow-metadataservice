package metadataservice_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.hollow.sh/metadataservice/internal/dbtools"
	"go.hollow.sh/metadataservice/internal/models"
	v1api "go.hollow.sh/metadataservice/pkg/api/v1"
)

func TestGetMetadataByIP(t *testing.T) {
	router := *testHTTPServer(t)

	type testCase struct {
		testName       string
		instanceIP     string
		expectedStatus int
		expectedBody   string
	}

	testCases := []testCase{
		{
			"unknown IPv4 address",
			"1.2.3.4",
			http.StatusNotFound,
			"",
		},
		{
			"unknown IPv6 address",
			"fe80::aede:48ff:fe00:1122",
			http.StatusNotFound,
			"",
		},
	}

	// Instance A tests
	for _, hostIP := range dbtools.FixtureInstanceA.HostIPs {
		caseItem := testCase{
			fmt.Sprintf("Instance A IP %s", hostIP),
			hostIP,
			http.StatusOK,
			dbtools.FixtureInstanceA.InstanceMetadata.Metadata.String(),
		}

		testCases = append(testCases, caseItem)
	}

	// Instance B tests
	for _, hostIP := range dbtools.FixtureInstanceB.HostIPs {
		caseItem := testCase{
			fmt.Sprintf("Instance B IP %s", hostIP),
			hostIP,
			http.StatusOK,
			dbtools.FixtureInstanceB.InstanceMetadata.Metadata.String(),
		}

		testCases = append(testCases, caseItem)
	}

	// Instance E tests
	// Instance E does not have any metadata, so *for now* we should expect it to return 404.
	// Once we've implemented the call-out-to-external-service bits, we'll need to update this test.
	for _, hostIP := range dbtools.FixtureInstanceE.HostIPs {
		caseItem := testCase{
			fmt.Sprintf("Instance E IP %s", hostIP),
			hostIP,
			http.StatusNotFound,
			"",
		}

		testCases = append(testCases, caseItem)
	}

	for _, testcase := range testCases {
		t.Run(testcase.testName, func(t *testing.T) {
			w := httptest.NewRecorder()

			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodGet, v1api.GetMetadataPath(), nil)
			req.RemoteAddr = net.JoinHostPort(testcase.instanceIP, "0")
			router.ServeHTTP(w, req)

			assert.Equal(t, testcase.expectedStatus, w.Code)

			if testcase.expectedStatus == http.StatusOK {
				var (
					err         error
					expectedMap map[string]interface{}
					resultMap   map[string]interface{}
				)

				err = json.Unmarshal([]byte(testcase.expectedBody), &expectedMap)
				if err != nil {
					t.Fatal(err)
				}

				err = json.Unmarshal(w.Body.Bytes(), &resultMap)
				if err != nil {
					t.Fatal(err)
				}

				assert.Equal(t, expectedMap, resultMap)
			}
		})
	}
}

// TestSetMetadataRequestValidations tests the different validations performed
// on the request body
func TestSetMetadataRequestValidations(t *testing.T) {
	router := *testHTTPServer(t)

	type testCase struct {
		testName       string
		requestBody    *v1api.UpsertMetadataRequest
		expectedStatus int
		expectedBody   *regexp.Regexp
	}

	testCases := []testCase{
		{
			"empty instance ID",
			&v1api.UpsertMetadataRequest{
				ID:          "",
				Metadata:    `{"some": "json"}`,
				IPAddresses: []string{"1.2.3.4", "10.1.0.0/25", "fe80:aede:48ff:fe00::1122"},
			},
			http.StatusBadRequest,
			regexp.MustCompile(`.*id.*required`),
		},
		{
			"non-uuid instance ID",
			&v1api.UpsertMetadataRequest{
				ID:          "abc123",
				Metadata:    `{"some": "json"}`,
				IPAddresses: []string{"1.2.3.4", "10.1.0.0/25", "fe80:aede:48ff:fe00::1122"},
			},
			http.StatusBadRequest,
			regexp.MustCompile(`.*id.*uuid`),
		},
		{
			"empty instance ID and empty metadata",
			&v1api.UpsertMetadataRequest{
				ID:          "",
				Metadata:    "",
				IPAddresses: []string{"1.2.3.4", "10.1.0.0/25", "fe80:aede:48ff:fe00::1122"},
			},
			http.StatusBadRequest,
			regexp.MustCompile(`.*id.*required.*metadata.*required`),
		},
		{
			"invalid IPv4 address",
			&v1api.UpsertMetadataRequest{
				ID:          "b9b24320-304e-4bfb-b46a-db75901c2f46",
				Metadata:    `{"some": "json"}`,
				IPAddresses: []string{"a.b.c.d"},
			},
			http.StatusBadRequest,
			regexp.MustCompile(`.*ipAddresses\[0\].*ip_addr|cidr`),
		},
		{
			"invalid IPv6 address",
			&v1api.UpsertMetadataRequest{
				ID:          "02d91622-b1e8-41b4-9add-ce77ac619b89",
				Metadata:    `{"some": "json"}`,
				IPAddresses: []string{"a:b:c:d:e:f:g:h"},
			},
			http.StatusBadRequest,
			regexp.MustCompile(`.*ipAddresses\[0\].*ip_addr|cidr`),
		},
	}

	for _, testcase := range testCases {
		t.Run(testcase.testName, func(t *testing.T) {
			reqBody, err := json.Marshal(testcase.requestBody)
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()

			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPost, v1api.GetMetadataPath(), bytes.NewReader(reqBody))
			router.ServeHTTP(w, req)

			assert.Equal(t, testcase.expectedStatus, w.Code)
			assert.Regexp(t, testcase.expectedBody, w.Body.String())
		})
	}
}

// TestSetMetadataIPAddressConflict tests the actions performed when the
// incoming request specifies an IP address (or multiple IP addresses) that are
// currently associated to another instance.
func TestSetMetadataIPAddressConflict(t *testing.T) {
	router := *testHTTPServer(t)
	testDB := dbtools.TestDB()

	type testCase struct {
		testName                string
		conflictInstanceIDToIPs map[string][]string
		requestBody             *v1api.UpsertMetadataRequest
	}

	testCases := []testCase{
		{
			"single IPv4 conflict",
			map[string][]string{dbtools.FixtureInstanceA.InstanceID: {dbtools.FixtureInstanceA.HostIPs[0]}},
			&v1api.UpsertMetadataRequest{
				ID:          "59e1fac8-adc5-4955-9cc3-2fa3e5f5370e",
				Metadata:    `{"some": "json"}`,
				IPAddresses: []string{dbtools.FixtureInstanceA.HostIPs[0]},
			},
		},
		{
			"single IPv6 conflict",
			map[string][]string{dbtools.FixtureInstanceA.InstanceID: {dbtools.FixtureInstanceA.HostIPs[1]}},
			&v1api.UpsertMetadataRequest{
				ID:          "b5b851a7-ea59-498d-b5c2-9ba10201ac28",
				Metadata:    `{"some": "json"}`,
				IPAddresses: []string{dbtools.FixtureInstanceA.HostIPs[1]},
			},
		},
		{
			"ipv4 and ipv6 conflict from same 'old' instance",
			map[string][]string{dbtools.FixtureInstanceA.InstanceID: {dbtools.FixtureInstanceA.HostIPs[0], dbtools.FixtureInstanceA.HostIPs[1]}},
			&v1api.UpsertMetadataRequest{
				ID:          "12256023-e708-4620-b6f0-57d39541994a",
				Metadata:    `{"some": "json"}`,
				IPAddresses: []string{dbtools.FixtureInstanceA.HostIPs[0], dbtools.FixtureInstanceA.HostIPs[1]},
			},
		},
		{
			"ipv4 conflicts from two different 'old' instances",
			map[string][]string{
				dbtools.FixtureInstanceA.InstanceID: {dbtools.FixtureInstanceA.HostIPs[0]},
				dbtools.FixtureInstanceB.InstanceID: {dbtools.FixtureInstanceB.HostIPs[0]}},
			&v1api.UpsertMetadataRequest{
				ID:          "6bd001dd-0523-4002-93e9-36a98607638a",
				Metadata:    `{"some": "json"}`,
				IPAddresses: []string{dbtools.FixtureInstanceA.HostIPs[0], dbtools.FixtureInstanceB.HostIPs[0]},
			},
		},
		{
			"ipv6 conflicts from two different 'old' instances",
			map[string][]string{
				dbtools.FixtureInstanceA.InstanceID: {dbtools.FixtureInstanceA.HostIPs[1]},
				dbtools.FixtureInstanceB.InstanceID: {dbtools.FixtureInstanceB.HostIPs[1]},
			},
			&v1api.UpsertMetadataRequest{
				ID:          "8c18b684-efb4-476b-87c3-a1dfd70a2024",
				Metadata:    `{"some": "json"}`,
				IPAddresses: []string{dbtools.FixtureInstanceA.HostIPs[1], dbtools.FixtureInstanceB.HostIPs[1]},
			},
		},
		{
			"ipv4 and ipv6 conflicts from two different 'old' instances",
			map[string][]string{
				dbtools.FixtureInstanceA.InstanceID: {dbtools.FixtureInstanceA.HostIPs[0]},
				dbtools.FixtureInstanceB.InstanceID: {dbtools.FixtureInstanceB.HostIPs[1]},
			},
			&v1api.UpsertMetadataRequest{
				ID:          "f92d1d4a-a408-42d7-b541-3bc3296c9c7d",
				Metadata:    `{"some": "json"}`,
				IPAddresses: []string{dbtools.FixtureInstanceA.HostIPs[0], dbtools.FixtureInstanceB.HostIPs[1]},
			},
		},
	}

	for _, testcase := range testCases {
		t.Run(testcase.testName, func(t *testing.T) {
			reqBody, err := json.Marshal(testcase.requestBody)
			if err != nil {
				t.Fatal(err)
			}

			w := httptest.NewRecorder()

			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPost, v1api.GetMetadataPath(), bytes.NewReader(reqBody))
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			// Check that the conflicting InstanceIPAddress row has been deleted
			for id, conflictIPs := range testcase.conflictInstanceIDToIPs {
				for _, conflictIP := range conflictIPs {
					count, err := models.InstanceIPAddresses(models.InstanceIPAddressWhere.InstanceID.EQ(id), models.InstanceIPAddressWhere.Address.EQ(conflictIP)).Count(context.TODO(), testDB)
					if err != nil {
						t.Fatal(err)
					}

					assert.Equal(t, int64(0), count)
				}
			}
		})
	}
}

// TestSetMetadataCreateMetadata tests the actions we perform when we receive a
// request that should insert the metadata for an instance ID we haven't seen
// before.
func TestSetMetadataCreateMetadata(t *testing.T) {
	router := *testHTTPServer(t)
	testDB := dbtools.TestDB()

	requestBody := &v1api.UpsertMetadataRequest{
		ID:          "b94fa75b-1fee-45eb-9925-83011c4834b9",
		Metadata:    `{"some": "json for instance 'b94fa75b-1fee-45eb-9925-83011c4834b9'"}`,
		IPAddresses: []string{"192.168.0.1/25"},
	}

	// Assert that we don't have an existing record for InstanceID
	exists, err := models.InstanceIPAddressExists(context.TODO(), testDB, requestBody.ID)
	if err != nil {
		t.Fatal(err)
	}

	assert.False(t, exists)

	reqBody, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()

	req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPost, v1api.GetMetadataPath(), bytes.NewReader(reqBody))
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	instanceMetadata, _ := models.InstanceMetadata(models.InstanceMetadatumWhere.ID.EQ(requestBody.ID)).One(context.TODO(), testDB)
	assert.NotNil(t, instanceMetadata)
	assert.Equal(t, requestBody.ID, instanceMetadata.ID)
	assert.Equal(t, requestBody.Metadata, instanceMetadata.Metadata.String())

	instanceIPAddresses, _ := models.InstanceIPAddresses(models.InstanceIPAddressWhere.ID.EQ(requestBody.ID)).All(context.TODO(), testDB)
	for _, instanceIPAddress := range instanceIPAddresses {
		assert.Equal(t, requestBody.ID, instanceIPAddress.InstanceID)

		found := false

		for _, ipAddress := range requestBody.IPAddresses {
			if ipAddress == instanceIPAddress.Address {
				found = true
			}
		}

		assert.True(t, found)
	}
}

// TestSetMetadataUpsertMetadata tests the actions we perform when we receive a
// request that should update the metadata for an existing instance record.
func TestSetMetadataUpsertMetadata(t *testing.T) {
	router := *testHTTPServer(t)
	testDB := dbtools.TestDB()

	// Assert that we have an existing record for InstanceID
	exists, err := models.InstanceMetadatumExists(context.TODO(), testDB, dbtools.FixtureInstanceA.InstanceID)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, exists)

	requestBody := &v1api.UpsertMetadataRequest{
		ID:          dbtools.FixtureInstanceA.InstanceID,
		Metadata:    `{"some": "json"}`,
		IPAddresses: dbtools.FixtureInstanceA.HostIPs,
	}

	reqBody, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()

	req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPost, v1api.GetMetadataPath(), bytes.NewReader(reqBody))
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	instanceMetadata, _ := models.InstanceMetadata(models.InstanceMetadatumWhere.ID.EQ(dbtools.FixtureInstanceA.InstanceID)).One(context.TODO(), testDB)
	assert.Equal(t, requestBody.Metadata, instanceMetadata.Metadata.String())
}
