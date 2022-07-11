package ec2

import (
	"strings"
)

// MetadataContainer is an interface defining methods used to access the list
// of available metadata items as well their individual values
type MetadataContainer interface {
	ItemNames() []string
	TopLevelItemNames() []string
	GetItem(itemPath string) ([]string, bool)
}

// Metadata represents the top-level fields of the metadata
type Metadata struct {
	ID              string           `json:"id"`
	Hostname        string           `json:"hostname"`
	IQN             string           `json:"iqn"`
	Plan            string           `json:"plan"`
	Facility        string           `json:"facility"`
	Tags            []string         `json:"tags"`
	OperatingSystem *OperatingSystem `json:"operating_system"`
	SSHKeys         []string         `json:"ssh_keys"`
	Spot            *Spot            `json:"spot"`
	Network         *Network         `json:"network"`
}

// ItemNames returns the list of top-level metadata keys that can be
// subsequently queried by a client. For a Metadata record, this is thee same
// as the list of "Top Level" item names.
func (metadata *Metadata) ItemNames() []string {
	return metadata.TopLevelItemNames()
}

// TopLevelItemNames returns the list of metadata items exposed by this record
// at the "top level" (that is, the /meta-data endpoint).
func (metadata *Metadata) TopLevelItemNames() []string {
	items := []string{
		"instance-id",
		"hostname",
		"iqn",
		"plan",
		"facility",
		"tags",
		"operating-system",
		"public-keys",
	}

	items = append(items, metadata.Spot.TopLevelItemNames()...)
	items = append(items, metadata.Network.TopLevelItemNames()...)

	return items
}

// GetItem takes a string "item path" like "/instance-id" or
// "/operating-system/slug" and returns a slice of metadata values for the
// requested item. If metadata doesn't contain a value for the requested
// item path, it will return an empty slice and false.
// While most calls will result in just a 1-element slice, Some metadata items
// might contain more than one value for the requested item
// (for example, an instance might have more than 1 public IPv4 address).
func (metadata *Metadata) GetItem(itemPath string) ([]string, bool) {
	if metadata == nil {
		return []string{}, false
	}

	// trim any leading or trailing slashes
	trimmed := strings.Trim(itemPath, "/")

	if trimmed == "" {
		return []string{}, false
	}

	switch {
	// Handle all the top-level items first
	case trimmed == "instance-id":
		return []string{metadata.ID}, true
	case trimmed == "hostname":
		return []string{metadata.Hostname}, true
	case trimmed == "iqn":
		return []string{metadata.IQN}, true
	case trimmed == "plan":
		return []string{metadata.Plan}, true
	case trimmed == "facility":
		return []string{metadata.Facility}, true
	case trimmed == "tags":
		return metadata.Tags, true
	case trimmed == "public-keys":
		return metadata.SSHKeys, true
	case trimmed == "public-ipv4" || trimmed == "public-ipv6" || trimmed == "local-ipv4":
		return metadata.Network.GetItem(trimmed)
	// Now handle the potentially-nested items
	case strings.HasPrefix(trimmed, "operating-system"):
		return metadata.OperatingSystem.GetItem(strings.TrimPrefix(trimmed, "operating-system"))
	case strings.HasPrefix(trimmed, "spot"):
		return metadata.Spot.GetItem(strings.TrimPrefix(trimmed, "spot"))
	default:
		return []string{}, false
	}
}

// Network represents the network-related fields in the metadata
type Network struct {
	Addresses  []NetworkAddress   `json:"addresses"`
	Bonding    *NetworkBonding    `json:"bonding"`
	Interfaces []NetworkInterface `json:"interfaces"`
}

// ItemNames returns the list of network-related metadata items
func (network *Network) ItemNames() []string {
	var items []string

	if publicIPv4 := network.filterNetworkAddressess(publicIPv4Filter); len(publicIPv4) > 0 {
		items = append(items, "public-ipv4")
	}

	if publicIPv6 := network.filterNetworkAddressess(publicIPv6Filter); len(publicIPv6) > 0 {
		items = append(items, "public-ipv6")
	}

	if localIPv4 := network.filterNetworkAddressess(localIPv4Filter); len(localIPv4) > 0 {
		items = append(items, "local-ipv4")
	}

	return items
}

// TopLevelItemNames returns the list of metadata items exposed by this record
// at the "top level" (that is, the /meta-data endpoint).
// The network record items are all exposed at the top-level currently, under
// the aliases "public-ipv4", "public-ipv6", and "local-ipv4".
func (network *Network) TopLevelItemNames() []string {
	if network != nil {
		return network.ItemNames()
	}

	return []string{}
}

// GetItem returns the value for an operating network-related item
func (network *Network) GetItem(itemPath string) ([]string, bool) {
	if network == nil {
		return []string{}, false
	}

	trimmed := strings.Trim(itemPath, "/")

	var (
		result     []string
		filterFunc addressFilter
	)

	switch trimmed {
	case "public-ipv4":
		filterFunc = publicIPv4Filter
	case "public-ipv6":
		filterFunc = publicIPv6Filter
	case "local-ipv4":
		filterFunc = localIPv4Filter
	}

	if filterFunc != nil {
		for _, addr := range network.filterNetworkAddressess(filterFunc) {
			result = append(result, addr.Address)
		}
	}

	return result, len(result) != 0
}

type addressFilter func(address *NetworkAddress) bool

func publicIPv4Filter(address *NetworkAddress) bool {
	return address.AddressFamily == 4 && address.Public
}

func publicIPv6Filter(address *NetworkAddress) bool {
	return address.AddressFamily == 6 && address.Public
}

func localIPv4Filter(address *NetworkAddress) bool {
	return address.AddressFamily == 4 && !address.Public
}

func (network *Network) filterNetworkAddressess(filter addressFilter) []NetworkAddress {
	var filteredAddresses []NetworkAddress

	for _, addr := range network.Addresses {
		if filter(&addr) {
			filteredAddresses = append(filteredAddresses, addr)
		}
	}

	return filteredAddresses
}

// NetworkBonding represents network bonding-related information in the
// metadata
type NetworkBonding struct {
	Mode int `json:"mode"`
}

// NetworkInterface represents fields describing a network interface
type NetworkInterface struct {
	Name string `json:"name"`
}

// NetworkAddress represents the fields describing a network address
type NetworkAddress struct {
	ID            string `json:"id"`
	AddressFamily int    `json:"address_family"`
	Netmask       string `json:"netmask"`
	Public        bool   `json:"public"`
	Address       string `json:"address" validate:"ip_addr|cidr"`
}

// OperatingSystem represents the fields describing the OS
type OperatingSystem struct {
	Slug              string             `json:"slug"`
	Distro            string             `json:"distro"`
	Version           string             `json:"version"`
	LicenseActivation *LicenseActivation `json:"license_activation"`
	ImageTag          string             `json:"image_tag"`
}

// ItemNames returns the list of operating system-related metadata items
func (os *OperatingSystem) ItemNames() []string {
	return []string{
		"slug",
		"distro",
		"version",
		"license-activation",
		"image-tag",
	}
}

// TopLevelItemNames returns the list of metadata items exposed by this record
// at the "top level" (that is, the /meta-data endpoint).
// For the OperatingSystem record, this is just "operating-system".
func (os *OperatingSystem) TopLevelItemNames() []string {
	return []string{"operating-system"}
}

// GetItem returns the value for an operating system-related item
func (os *OperatingSystem) GetItem(itemPath string) ([]string, bool) {
	if os == nil {
		return []string{}, false
	}

	trimmed := strings.Trim(itemPath, "/")

	switch {
	case trimmed == "":
		return os.ItemNames(), true
	case trimmed == "slug":
		return []string{os.Slug}, true
	case trimmed == "distro":
		return []string{os.Distro}, true
	case trimmed == "version":
		return []string{os.Version}, true
	case trimmed == "image-tag":
		return []string{os.ImageTag}, true
	case strings.HasPrefix(trimmed, "license-activation"):
		return os.LicenseActivation.GetItem(strings.TrimPrefix(trimmed, "license-activation"))
	default:
		return []string{}, false
	}
}

// LicenseActivation represents the fields relating to OS license activations
type LicenseActivation struct {
	State string `json:"state"`
}

// ItemNames returns the list of license activation-related metadata items
func (la *LicenseActivation) ItemNames() []string {
	return []string{"state"}
}

// TopLevelItemNames returns the list of metadata items exposed by this record
// at the "top level" (that is, the /meta-data endpoint).
// The License activation record does not expose any top-level items.
func (la *LicenseActivation) TopLevelItemNames() []string {
	return []string{}
}

// GetItem returns the value for a license activation-related item
func (la *LicenseActivation) GetItem(itemPath string) ([]string, bool) {
	if la == nil {
		return []string{}, false
	}

	trimmed := strings.Trim(itemPath, "/")

	switch {
	case trimmed == "":
		return la.ItemNames(), true
	case trimmed == "state":
		return []string{la.State}, true
	default:
		return []string{}, false
	}
}

// Spot represents the fields describing spot market-related fields
type Spot struct {
	TerminationTime string `json:"termination_time"`
}

// ItemNames returns the list of spot market-related metadata items
func (spot *Spot) ItemNames() []string {
	return []string{"termination-time"}
}

// TopLevelItemNames returns the list of metadata items exposed by this record
// at the "top level" (that is, the /meta-data endpoint).
// For a spot record, this is just "spot"
func (spot *Spot) TopLevelItemNames() []string {
	if spot != nil {
		return []string{"spot"}
	}

	return []string{}
}

// GetItem returns the value for a spot-related item.
func (spot *Spot) GetItem(itemPath string) ([]string, bool) {
	if spot == nil {
		return []string{}, false
	}

	trimmed := strings.Trim(itemPath, "/")

	switch trimmed {
	case "":
		return spot.ItemNames(), true
	case "termination-time":
		return []string{spot.TerminationTime}, true
	default:
		return []string{}, false
	}
}
