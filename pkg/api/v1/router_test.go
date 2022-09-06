package metadataservice_test

import (
	"context"
	"net/http"
	"testing"

	"go.hollow.sh/toolbox/ginjwt"
	"go.uber.org/zap"

	"go.hollow.sh/metadataservice/internal/dbtools"
	"go.hollow.sh/metadataservice/internal/httpsrv"
	"go.hollow.sh/metadataservice/internal/lookup"
)

func testHTTPServer(t *testing.T) *http.Handler {
	authConfig := ginjwt.AuthConfig{}

	db := dbtools.DatabaseTest(t)

	hs := httpsrv.Server{Logger: zap.NewNop(), AuthConfig: authConfig, DB: db}

	s := hs.NewServer()

	return &s.Handler
}

func testHTTPServerWithLookup(t *testing.T, lookupClient lookup.Client) *http.Handler {
	authConfig := ginjwt.AuthConfig{}

	db := dbtools.DatabaseTest(t)

	hs := httpsrv.Server{Logger: zap.NewNop(), AuthConfig: authConfig, DB: db, LookupEnabled: true, LookupClient: lookupClient}

	s := hs.NewServer()

	return &s.Handler
}

type lookupResponse struct {
	metadataResponse lookup.MetadataLookupResponse
	userdataResponse lookup.UserdataLookupResponse
	Error            error
}

type mockLookupClient struct {
	responses map[string]lookupResponse
}

func newMockLookupClient() *mockLookupClient {
	return &mockLookupClient{responses: make(map[string]lookupResponse)}
}

func (m *mockLookupClient) setResponse(key string, resp lookupResponse) {
	m.responses[key] = resp
}

func (m *mockLookupClient) getMetadataResponse(key string) (*lookup.MetadataLookupResponse, error) {
	resp, exists := m.responses[key]
	if !exists {
		return nil, lookup.ErrNotFound
	}

	return &resp.metadataResponse, resp.Error
}

func (m *mockLookupClient) getUserdataResponse(key string) (*lookup.UserdataLookupResponse, error) {
	resp, exists := m.responses[key]
	if !exists {
		return nil, lookup.ErrNotFound
	}

	return &resp.userdataResponse, resp.Error
}

func (m *mockLookupClient) GetMetadataByID(_ context.Context, id string) (*lookup.MetadataLookupResponse, error) {
	return m.getMetadataResponse(id)
}

func (m *mockLookupClient) GetMetadataByIP(_ context.Context, ip string) (*lookup.MetadataLookupResponse, error) {
	return m.getMetadataResponse(ip)
}

func (m *mockLookupClient) GetUserdataByID(_ context.Context, id string) (*lookup.UserdataLookupResponse, error) {
	return m.getUserdataResponse(id)
}

func (m *mockLookupClient) GetUserdataByIP(_ context.Context, ip string) (*lookup.UserdataLookupResponse, error) {
	return m.getUserdataResponse(ip)
}
