package metadataservice_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.hollow.sh/metadataservice/internal/dbtools"
	v1api "go.hollow.sh/metadataservice/pkg/api/v1"
)

func TestGetUserDataByIP(t *testing.T) {
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
			string(dbtools.FixtureInstanceA.InstanceUserdata.Userdata.Bytes),
		}

		testCases = append(testCases, caseItem)
	}

	// Instance B tests
	// Instance B does not have any userdata, so *for now* we should expect it to return 404.
	// Once we've implemented the call-out-to-external-service bits, we'll need to update this test.
	for _, hostIP := range dbtools.FixtureInstanceB.HostIPs {
		caseItem := testCase{
			fmt.Sprintf("Instance B IP %s", hostIP),
			hostIP,
			http.StatusNotFound,
			"",
		}

		testCases = append(testCases, caseItem)
	}

	// Instance E tests
	for _, hostIP := range dbtools.FixtureInstanceE.HostIPs {
		caseItem := testCase{
			fmt.Sprintf("Instance E IP %s", hostIP),
			hostIP,
			http.StatusOK,
			string(dbtools.FixtureInstanceE.InstanceUserdata.Userdata.Bytes),
		}

		testCases = append(testCases, caseItem)
	}

	for _, testcase := range testCases {
		t.Run(testcase.testName, func(t *testing.T) {
			w := httptest.NewRecorder()

			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodGet, v1api.GetUserdataPath(), nil)
			req.RemoteAddr = net.JoinHostPort(testcase.instanceIP, "0")
			router.ServeHTTP(w, req)

			assert.Equal(t, testcase.expectedStatus, w.Code)

			if testcase.expectedStatus == http.StatusOK {
				assert.Equal(t, testcase.expectedBody, w.Body.String())
			}
		})
	}
}
