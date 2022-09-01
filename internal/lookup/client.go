package lookup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"go.hollow.sh/toolbox/version"
	"go.uber.org/zap"
)

var (
	errBaseURLParse = errors.New("could not parse base URL")
	errNoBaseURL    = errors.New("failed to initialize: no lookup service base URL provided")
	userAgentString = fmt.Sprintf("go-hollow-metadataservice-lookup-client (%s)", version.String())
)

// MetadataLookupResponse represents the data we expect to receive from a call
// to the lookup service for an instance's metadata.
type MetadataLookupResponse struct {
	ID          string   `json:"id"`
	IPAddresses []string `json:"ipAddresses"`
	Metadata    string   `json:"metadata"`
}

// UserdataLookupResponse represents the data we expect to receive from a call
// to the lookup service for an instance's userdata.
type UserdataLookupResponse struct {
	ID          string   `json:"id"`
	IPAddresses []string `json:"ipAddresses"`
	Userdata    []byte   `json:"userdata"`
}

// Client defines the methods and lookup service client should implement
type Client interface {
	GetMetadataByID(ctx context.Context, instanceID string) (*MetadataLookupResponse, error)
	GetMetadataByIP(ctx context.Context, instanceIP string) (*MetadataLookupResponse, error)
	GetUserdataByID(ctx context.Context, instanceID string) (*UserdataLookupResponse, error)
	GetUserdataByIP(ctx context.Context, instanceIP string) (*UserdataLookupResponse, error)
}

// ServiceClient is the client used to reach out to the lookup service.
type ServiceClient struct {
	BaseURL *url.URL
	client  *http.Client
	Logger  *zap.Logger
}

// ErrorResponse represents an error response record received from the lookup
// service.
type ErrorResponse struct {
	Errors []string `json:"errors,omitempty"`
}

// NewClient builds a new client for calling the lookup service. Pass in a
// base URL for the lookup service, and an *http.Client with oauth2 creds setup
func NewClient(logger *zap.Logger, baseURL string, httpClient *http.Client) (*ServiceClient, error) {
	if baseURL == "" {
		return nil, errNoBaseURL
	}

	parsedURL, err := url.Parse(baseURL)

	if err != nil {
		return nil, fmt.Errorf("%w: %s", errBaseURLParse, baseURL)
	}

	c := &ServiceClient{
		BaseURL: parsedURL,
		client:  httpClient,
		Logger:  logger,
	}

	return c, nil
}

// GetMetadataByID is used to look up metadata by instance ID
func (c *ServiceClient) GetMetadataByID(ctx context.Context, instanceID string) (*MetadataLookupResponse, error) {
	path := path.Join("device-metadata", instanceID)

	resp, err := c.getMetadata(ctx, path)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.Logger.Sugar().Warnf("Metadata for instance ID %s was not found in the Lookup Service", instanceID)
		}
	}

	return resp, err
}

// GetMetadataByIP is used to look up metadata by instance IP address
func (c *ServiceClient) GetMetadataByIP(ctx context.Context, instanceIP string) (*MetadataLookupResponse, error) {
	path := fmt.Sprintf("device-metadata?ip_address=%s", instanceIP)

	resp, err := c.getMetadata(ctx, path)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.Logger.Sugar().Warnf("Metadata for IP Address %s was not found in the Lookup Service", instanceIP)
		}
	}

	return resp, err
}

// GetUserdataByID is used to look up userdata by instance ID
func (c *ServiceClient) GetUserdataByID(ctx context.Context, instanceID string) (*UserdataLookupResponse, error) {
	path := path.Join("device-userdata", instanceID)

	resp, err := c.getUserdata(ctx, path)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.Logger.Sugar().Warnf("Userdata for instance ID %s was not found in the Lookup Service", instanceID)
		}
	}

	return resp, err
}

// GetUserdataByIP is used to look up userdata by instance IP address
func (c *ServiceClient) GetUserdataByIP(ctx context.Context, instanceIP string) (*UserdataLookupResponse, error) {
	path := fmt.Sprintf("device-userdata?ip_address=%s", instanceIP)

	resp, err := c.getUserdata(ctx, path)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.Logger.Sugar().Warnf("Userdata for IP Address %s was not found in the Lookup Service", instanceIP)
		}
	}

	return resp, err
}

func newGetRequest(ctx context.Context, baseURL string, path string) (*http.Request, error) {
	requestURL, err := url.Parse(fmt.Sprintf("%s/%s", baseURL, path))
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if req != nil {
		req.Header.Set("User-Agent", userAgentString)
	}

	return req, err
}

func (c *ServiceClient) getMetadata(ctx context.Context, path string) (*MetadataLookupResponse, error) {
	req, err := newGetRequest(ctx, c.BaseURL.String(), path)
	if err != nil {
		return nil, err
	}

	metadata := &MetadataLookupResponse{}

	err = c.get(req, metadata)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

func (c *ServiceClient) getUserdata(ctx context.Context, path string) (*UserdataLookupResponse, error) {
	req, err := newGetRequest(ctx, c.BaseURL.String(), path)
	if err != nil {
		return nil, err
	}

	userdata := &UserdataLookupResponse{}

	err = c.get(req, userdata)
	if err != nil {
		return nil, err
	}

	return userdata, nil
}

func (c *ServiceClient) get(req *http.Request, v interface{}) error {
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}

	if resp.StatusCode != http.StatusOK {
		errResp := map[string]string{}
		err = json.NewDecoder(resp.Body).Decode(&errResp)

		if err != nil {
			if err != nil {
				c.Logger.Sugar().Errorf("Received unexpected response status from Lookup Service: (%d), but failed to decode an error response: %v", resp.StatusCode, err)
			} else {
				c.Logger.Sugar().Errorf("Received unexpected response status from Lookup Service: (%d), with error: %v", resp.StatusCode, err)
			}
		}

		return fmt.Errorf("%w: %d", ErrUnexpectedStatus, resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(v)
}
