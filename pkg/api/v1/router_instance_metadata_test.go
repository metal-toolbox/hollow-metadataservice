package metadataservice_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.hollow.sh/metadataservice/internal/dbtools"
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
