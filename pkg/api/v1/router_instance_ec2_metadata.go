package metadataservice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"go.hollow.sh/metadataservice/internal/middleware"
	"go.hollow.sh/metadataservice/internal/models"
)

// var (
// 	metadataKeyFieldMap map[string]string = map[string]string{
// 		"instance-id": "id",
// 		"hostname": "hostname",
// 		"iqn": "iqn",
// 		"plan": "plan",
// 		"facility": "facility",
// 		"tags": "tags",
// 		"operating-system": "operating_system",
// 		"public-keys": "ssh_keys",
// 		"spot": "spot",
// 	}

// 	spotKeyFieldMap map[string]string = map[string]string{
// 		"termination-time": "termination_time",
// 	}

// 	networkKeyFieldMap map[string]string = map[string]string{
// 		"public-ipv4"
// 	}
// )

func (r *Router) instanceEc2MetadataGet(c *gin.Context) {
	instanceID := c.GetString(middleware.ContextKeyInstanceID)
	if instanceID == "" {
		// TODO: Try to fetch the metadata from an external source of truth.
		// Return 404 for now...
		notFoundResponse(c)
		return
	}

	metadata, err := models.FindInstanceMetadatum(c.Request.Context(), r.DB, instanceID)

	// Loading the metadata json into a generic map[string]interface loses the
	// ordering of the keys in the json document. We can use Decoder.Token to
	// step through the json and extract the keys, allowing us to preserve the
	// key order from the original metadata document.

	decoder := json.NewDecoder(bytes.NewReader(metadata.Metadata))

	for {
		t, err := decoder.Token()
		if err == io.EOF {
			break
		}

		if err != nil {
			fmt.Println("Error getting token: ", err)
		}

		fmt.Printf("%T: %v", t, t)
		if decoder.More() {
			fmt.Printf(" (more) ")
		}
		fmt.Printf("\n")
	}

	if err != nil {
		dbErrorResponse(c, err)
		return
	}

	// TODO: Extract requested key from metadata, this response is just a placeholder for now...
	c.JSON(http.StatusOK, metadata.Metadata)
}

func (r *Router) instanceEc2MetadataItemGet(c *gin.Context) {
	instanceID := c.GetString(middleware.ContextKeyInstanceID)
	if instanceID == "" {
		// TODO: Try to fetch the metadata from an external source of truth.
		// Return 404 for now...
		notFoundResponse(c)
		return
	}

	metadata, err := models.FindInstanceMetadatum(c.Request.Context(), r.DB, instanceID)

	if err != nil {
		dbErrorResponse(c, err)
		return
	}

	// TODO: Extract requested key from metadata, this response is just a placeholder for now...
	c.JSON(http.StatusOK, metadata.Metadata)
}

func (r *Router) instanceEc2UserdataGet(c *gin.Context) {
	instanceID := c.GetString(middleware.ContextKeyInstanceID)
	if instanceID == "" {
		// TODO: Try to fetch the userdata from an external source of truth.
		// Return 404 for now...
		notFoundResponse(c)
		return
	}

	userdata, err := models.FindInstanceUserdatum(c.Request.Context(), r.DB, instanceID)

	if err != nil {
		dbErrorResponse(c, err)
		return
	}

	c.String(http.StatusOK, string(userdata.Userdata.Bytes))
}

func spotKeys(metadata map[string]interface{}) []string {
	if _, hasSpot := metadata["spot"]; hasSpot {
		return []string{"spot"}
	}

	return []string{}
}

func networkKeys(metadata map[string]interface{}) []string {
	if networkData, hasNetwork := metadata["network"]; hasNetwork {
		if networkDataMap, ok := networkData.(map[string]interface{}); ok {
			if addresses, hasAddresses := networkDataMap["addresses"]; hasAddresses {
				if addressesSlice, ok := addresses.([]map[string]interface{}); ok {
					var ipKeys []string
					for _, address := range addressesSlice {
						ipKeys = append(ipKeys, addressKey(address))
					}
				}
			}
		}
	}

	return []string{}
}

func addressKey(address map[string]interface{}) string {

}
