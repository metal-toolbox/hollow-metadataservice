package metadataservice

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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

	instanceMetadata, err := models.FindInstanceMetadatum(c.Request.Context(), r.DB, instanceID)

	if err != nil {
		dbErrorResponse(c, err)
		return
	}

	var metadata = make(map[string]interface{})
	err = json.Unmarshal([]byte(instanceMetadata.Metadata), &metadata)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, &ErrorResponse{Errors: []string{"Invalid metadata for instance"}})
		return
	}

	standardItems := []string{
		"instance-id", "hostname", "iqn", "plan", "facility", "tags",
		"operating-system", "public-keys",
	}
	spotItems := getSpotItems(metadata)
	networkItems := getNetworkItems(metadata)

	var metadataItems []string = make([]string, len(standardItems)+len(spotItems)+len(networkItems))

	metadataItems = append(metadataItems, standardItems...)
	metadataItems = append(metadataItems, spotItems...)
	metadataItems = append(metadataItems, networkItems...)

	var sb strings.Builder
	for _, item := range metadataItems {
		sb.WriteString(fmt.Sprintf("%s\n", item))
	}

	c.String(http.StatusOK, sb.String())
}

func getSpotItems(metadata map[string]interface{}) []string {
	var spotItems []string

	if _, ok := metadata["spot"]; ok {
		spotItems = []string{"spot"}
	}

	return spotItems
}

func getNetworkItems(metadata map[string]interface{}) []string {
	var networkItems []string

	network, ok := metadata["network"]
	if !ok {
		return networkItems
	}

	networkMap, ok := network.(map[string]interface{})
	if !ok {
		return networkItems
	}

	addresses, ok := networkMap["addresses"]
	if !ok {
		return networkItems
	}

	addressSlice, ok := addresses.([]map[string]interface{})
	if !ok {
		return networkItems
	}

	return networkItems
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
