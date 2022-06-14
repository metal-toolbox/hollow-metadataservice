package metadataservice_test

import (
	"net/http"
	"testing"

	"go.hollow.sh/toolbox/ginjwt"
	"go.uber.org/zap"

	"go.hollow.sh/metadataservice/internal/dbtools"
	"go.hollow.sh/metadataservice/internal/httpsrv"
)

func testHTTPServer(t *testing.T) *http.Handler {
	authConfig := ginjwt.AuthConfig{}

	db := dbtools.DatabaseTest(t)

	hs := httpsrv.Server{Logger: zap.NewNop(), AuthConfig: authConfig, DB: db}

	s := hs.NewServer()

	return &s.Handler
}
