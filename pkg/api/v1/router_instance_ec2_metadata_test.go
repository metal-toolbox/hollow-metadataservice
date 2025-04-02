package metadataservice_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"go.hollow.sh/metadataservice/internal/dbtools"
	v1api "go.hollow.sh/metadataservice/pkg/api/v1"
)

// GetEc2MetadataItemPathWithoutTrim is used to test routing edge cases where
// the trailing '/' is kept
func getEc2MetadataItemPathWithoutTrim(itemPath string) string {
	fullpath := v1api.GetEc2MetadataItemPath(itemPath)

	// GetEc2MetadataItemPath() calls path.Join(), which strips trailing slashes.
	// So restore a trailing slash if itemPath came with one
	if itemPath != "" && itemPath[len(itemPath)-1:] == "/" {
		fullpath += "/"
	}

	return fullpath
}

func TestGetEc2MetadataByIP(t *testing.T) {
	viper.SetDefault("crdb.enabled", true)

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

	standardFields := "instance-id\nhostname\niqn\nplan\nfacility\ntags\noperating-system\npublic-keys"

	// Instance A tests
	// Instance A has all 3 ip types, but no spot market info
	for _, hostIP := range dbtools.FixtureInstanceA.HostIPs {
		caseItem := testCase{
			fmt.Sprintf("Instance A IP %s", hostIP),
			hostIP,
			http.StatusOK,
			fmt.Sprintf("%s\npublic-ipv4\npublic-ipv6\nlocal-ipv4", standardFields),
		}

		testCases = append(testCases, caseItem)
	}

	// Instance A1 tests
	// Instance A1 has an IPv6 and local IPv4 address, but no public IPv4 address
	for _, hostIP := range dbtools.FixtureInstanceA1.HostIPs {
		caseItem := testCase{
			fmt.Sprintf("Instance A1 IP %s", hostIP),
			hostIP,
			http.StatusOK,
			fmt.Sprintf("%s\npublic-ipv6\nlocal-ipv4", standardFields),
		}

		testCases = append(testCases, caseItem)
	}

	// Instance A2 tests
	// Instance A2 has a local IPv4 address, but no public IPv4 or IPv6 addresses.
	// Instance A2 additionally has spot market-related info
	for _, hostIP := range dbtools.FixtureInstanceA2.HostIPs {
		caseItem := testCase{
			fmt.Sprintf("Instance A2 IP %s", hostIP),
			hostIP,
			http.StatusOK,
			fmt.Sprintf("%s\nspot\nlocal-ipv4", standardFields),
		}

		testCases = append(testCases, caseItem)
	}

	// Instance B tests
	// Instance B has all 3 ip types, but no spot market info
	for _, hostIP := range dbtools.FixtureInstanceB.HostIPs {
		caseItem := testCase{
			fmt.Sprintf("Instance B IP %s", hostIP),
			hostIP,
			http.StatusOK,
			fmt.Sprintf("%s\npublic-ipv4\npublic-ipv6\nlocal-ipv4", standardFields),
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

			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodGet, v1api.GetEc2MetadataPath(), nil)
			req.RemoteAddr = net.JoinHostPort(testcase.instanceIP, "0")
			router.ServeHTTP(w, req)

			assert.Equal(t, testcase.expectedStatus, w.Code)

			if testcase.expectedStatus == http.StatusOK {
				assert.Equal(t, testcase.expectedBody, w.Body.String())
			}
		})
	}
}

func TestGetEc2MetadataItemByIP(t *testing.T) {
	viper.SetDefault("crdb.enabled", true)

	router := *testHTTPServer(t)

	type itemTestCase struct {
		testName       string
		itemName       string
		instanceIP     string
		expectedStatus int
		expectedBody   string
	}

	var standardItems = []string{
		"instance-id",
		"hostname",
		"iqn",
		"plan",
		"facility",
		"tags",
		"operating-system",
		"public-keys",
	}

	var testCases []itemTestCase

	for _, v := range standardItems {
		testcase := itemTestCase{
			fmt.Sprintf("unknown IPv4 address-%s", v),
			v,
			"1.2.3.4",
			http.StatusNotFound,
			"",
		}

		testCases = append(testCases, testcase)

		testcase = itemTestCase{
			fmt.Sprintf("unknown IPv6 address-%s", v),
			v,
			"fe80::aede:48ff:fe00:1122",
			http.StatusNotFound,
			"",
		}

		testCases = append(testCases, testcase)
	}

	// Instance A tests
	for _, hostIP := range dbtools.FixtureInstanceA.HostIPs {
		aCases := []itemTestCase{
			{
				fmt.Sprintf("Instance A IP %s-instance-id", hostIP),
				"instance-id",
				hostIP,
				http.StatusOK,
				"316ed337-feee-48c6-a11b-3d4738e3cd6d",
			},
			{
				fmt.Sprintf("Instance A IP %s-hostname", hostIP),
				"hostname",
				hostIP,
				http.StatusOK,
				"instance-a",
			},
			{
				fmt.Sprintf("Instance A IP %s-iqn", hostIP),
				"iqn",
				hostIP,
				http.StatusOK,
				"iqn.2022-02.net.packet:device.316ed337",
			},
			{
				fmt.Sprintf("Instance A IP %s-plan", hostIP),
				"plan",
				hostIP,
				http.StatusOK,
				"c3.medium.x86",
			},
			{
				fmt.Sprintf("Instance A IP %s-facility", hostIP),
				"facility",
				hostIP,
				http.StatusOK,
				"da11",
			},
			{
				fmt.Sprintf("Instance A IP %s-tags", hostIP),
				"tags",
				hostIP,
				http.StatusOK,
				"",
			},
			{
				fmt.Sprintf("Instance A IP %s-operating-system", hostIP),
				"operating-system",
				hostIP,
				http.StatusOK,
				"slug\ndistro\nversion\nlicense-activation\nimage-tag",
			},
			{
				fmt.Sprintf("Instance A IP %s-operating-system/slug", hostIP),
				"operating-system/slug",
				hostIP,
				http.StatusOK,
				"ubuntu_20_04",
			},
			{
				fmt.Sprintf("Instance A IP %s-operating-system/distro", hostIP),
				"operating-system/distro",
				hostIP,
				http.StatusOK,
				"ubuntu",
			},
			{
				fmt.Sprintf("Instance A IP %s-operating-system/version", hostIP),
				"operating-system/version",
				hostIP,
				http.StatusOK,
				"20.04",
			},
			{
				fmt.Sprintf("Instance A IP %s-operating-system/license-activation", hostIP),
				"operating-system/license-activation",
				hostIP,
				http.StatusOK,
				"state",
			},
			{
				fmt.Sprintf("Instance A IP %s-operating-system/license-activation/state", hostIP),
				"operating-system/license-activation/state",
				hostIP,
				http.StatusOK,
				"unlicensed",
			},
			{
				fmt.Sprintf("Instance A IP %s-operating-system/image-tag", hostIP),
				"operating-system/image-tag",
				hostIP,
				http.StatusOK,
				"31853a2b0b2fcc4ee7fd5da5e53611303b60aafa",
			},
			{
				fmt.Sprintf("Instance A IP %s-public-keys", hostIP),
				"public-keys",
				hostIP,
				http.StatusOK,
				"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCV2BCNvg7WQtMzcKHCNY6/qoFC8R6GJlKq3rQRcfJMkpmSGudHx8ojuyUaj04LjDFL5pkt2lnGT5aWo2N58Y1O/7diOUNUJrTy+ZWuliEfqE7hJwuszUjhYwhiuGk6UEw5/g+lfzTv1POEqMIg2cORI7OfmSs4tf7cXqY442rdDSv9H8LtqiBER47Et23sNrcDWbK57cc2/+nwqDWtmf7Nin4t8Kc5p2I4PFVsiXzRue7wKswJJp37ZOxlnbxAJ2BQ3PJwCf9Qe7Y/zAlqUnmDaERVZyDQSVIRE8XqRTh9UtcsGqi81WGLYnW63Nd3LkfJ2WdtfMkGjOGG4aRENvQtmWzyp1QM4A/n/25PbYB2VAogf8dIVjpUFek/tXcRPEUDT1skYFt8czimbmEMnRgjihIvS6oHybl2GnJ0zvpSA9MrZy+/9AkaW1M8QYuJdHQ9JcDpFKFkXMEVPW8uUGIc4rciBoeewbsunCV8StI1XnHpaqe1VhPhCA0JK74Tnv7MUTCN8YCY65Vp6Rq4nGlNA34bJ4A0b99atmo6vYr1rvHs6R6NC+mxLyvzBQYMzhXFBbzeyFNGDdw8eRQy5WGAfyvjTQMtOK6bDpKjc57np8qJrRhIM7+Y8ovF1GWEentBzQyWAcPilvq0fSzBNDQxr7GSSRRc5USqAk0NgZPXlQ== test@user.local\nssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDPgTv1yUmNCGUcnCuFr94SQ0YqpuMwKSC022Fp2Q3TF test@user.local",
			},
			{
				fmt.Sprintf("Instance A IP %s-spot", hostIP),
				"spot",
				hostIP,
				http.StatusNotFound,
				"",
			},
			{
				fmt.Sprintf("Instance A IP %s-spot/termination-time", hostIP),
				"spot/termination-time",
				hostIP,
				http.StatusNotFound,
				"",
			},
			{
				fmt.Sprintf("Instance A IP %s-public-ipv4", hostIP),
				"public-ipv4",
				hostIP,
				http.StatusOK,
				"139.178.82.3",
			},
			{
				fmt.Sprintf("Instance A IP %s-public-ipv6", hostIP),
				"public-ipv6",
				hostIP,
				http.StatusOK,
				"2604:1380:4641:1f00::9",
			},
			{
				fmt.Sprintf("Instance A IP %s-local-ipv4", hostIP),
				"local-ipv4",
				hostIP,
				http.StatusOK,
				"10.70.17.9",
			},
		}
		testCases = append(testCases, aCases...)
	}

	// Instance A2 tests
	// Instance A2 has just a local ipv4 address and spot market info
	for _, hostIP := range dbtools.FixtureInstanceA2.HostIPs {
		a2Cases := []itemTestCase{
			{
				fmt.Sprintf("Instance A2 IP %s-instance-id", hostIP),
				"instance-id",
				hostIP,
				http.StatusOK,
				"083637a8-b674-4d33-b199-a819441d85c0",
			},
			{
				fmt.Sprintf("Instance A2 IP %s-hostname", hostIP),
				"hostname",
				hostIP,
				http.StatusOK,
				"instance-a2",
			},
			{
				fmt.Sprintf("Instance A2 IP %s-iqn", hostIP),
				"iqn",
				hostIP,
				http.StatusOK,
				"iqn.2022-02.net.packet:device.083637a8",
			},
			{
				fmt.Sprintf("Instance A2 IP %s-spot", hostIP),
				"spot",
				hostIP,
				http.StatusOK,
				"termination-time",
			},
			{
				fmt.Sprintf("Instance A2 IP %s-spot/termination-time", hostIP),
				"spot/termination-time",
				hostIP,
				http.StatusOK,
				"20220707T13:13:13Z",
			},
			{
				fmt.Sprintf("Instance A2 IP %s-public-ipv4", hostIP),
				"public-ipv4",
				hostIP,
				http.StatusNotFound,
				"",
			},
			{
				fmt.Sprintf("Instance A2 IP %s-public-ipv6", hostIP),
				"public-ipv6",
				hostIP,
				http.StatusNotFound,
				"",
			},
			{
				fmt.Sprintf("Instance A2 IP %s-local-ipv4", hostIP),
				"local-ipv4",
				hostIP,
				http.StatusOK,
				"10.70.17.25",
			},
		}
		testCases = append(testCases, a2Cases...)
	}

	// Instance E tests
	// Instance E does not have any metadata, so *for now* we should expect it to return 404.
	// Once we've implemented the call-out-to-external-service bits, we'll need to update this test.
	for _, hostIP := range dbtools.FixtureInstanceE.HostIPs {
		testcase := itemTestCase{
			fmt.Sprintf("Instance E IP %s-instance-id", hostIP),
			"instance-id",
			hostIP,
			http.StatusNotFound,
			"",
		}

		testCases = append(testCases, testcase)
	}

	for _, testcase := range testCases {
		t.Run(testcase.testName, func(t *testing.T) {
			w := httptest.NewRecorder()

			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodGet, v1api.GetEc2MetadataItemPath(testcase.itemName), nil)
			req.RemoteAddr = net.JoinHostPort(testcase.instanceIP, "0")
			router.ServeHTTP(w, req)

			assert.Equal(t, testcase.expectedStatus, w.Code)

			if testcase.expectedStatus == http.StatusOK {
				assert.Equal(t, testcase.expectedBody, w.Body.String())
			}
		})
	}

	t.Run("check routing works with trailing slash in the url", func(t *testing.T) {
		viper.SetDefault("crdb.enabled", true)

		w := httptest.NewRecorder()

		standardFields := "instance-id\nhostname\niqn\nplan\nfacility\ntags\noperating-system\npublic-keys"

		itemName := "/"
		instanceIP := "139.178.82.3"
		expectedStatus := http.StatusOK
		expectedBody := fmt.Sprintf("%s\npublic-ipv4\npublic-ipv6\nlocal-ipv4", standardFields)

		req, _ := http.NewRequestWithContext(context.TODO(), http.MethodGet, getEc2MetadataItemPathWithoutTrim(itemName), nil)
		req.RemoteAddr = net.JoinHostPort(instanceIP, "0")
		router.ServeHTTP(w, req)

		assert.Equal(t, expectedStatus, w.Code)

		if expectedStatus == http.StatusOK {
			assert.Equal(t, expectedBody, w.Body.String())
		}
	})
}
