//go:build testtools
// +build testtools

package dbtools

import (
	"context"
	_ "embed"
	"net"

	"github.com/jmoiron/sqlx"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/types"

	"go.hollow.sh/metadataservice/internal/models"
)

// InstanceFixture represents the metadata, userdata, and IP addresses
// for a particular instance
type InstanceFixture struct {
	InstanceID          string
	InstanceMetadata    *models.InstanceMetadatum
	InstanceUserdata    *models.InstanceUserdatum
	InstanceIPAddresses []models.InstanceIPAddress
	HostIPs             []string
}

const (
	instanceAUUID string = "316ed337-feee-48c6-a11b-3d4738e3cd6d"
	instanceBUUID string = "37066580-de45-44ea-8cbb-eff3e932e3b1"
	instanceCUUID string = "a830ee39-c037-4b27-8d4d-79da9e360568"
	instanceDUUID string = "beb5a9eb-5703-44ff-9091-d41130747b8d"
	instanceEUUID string = "93a9ffad-f636-49aa-96d9-fb894684978b"
	instanceFUUID string = "d5377460-4eb3-454c-aa85-233f18f4ee28"
)

var (
	// FixtureInstanceA represents an instance with Metadata, Userdata, and known IPs
	FixtureInstanceA *InstanceFixture
	// FixtureInstanceB represents an instance with Metadata, no Userdata, and known IPS
	FixtureInstanceB *InstanceFixture
	// FixtureInstanceC represents an instance with Metadata, Userdata, but no IPs
	FixtureInstanceC *InstanceFixture
	// FixtureInstanceD represents an instance with Metadata, no userdata, and no IPs
	FixtureInstanceD *InstanceFixture
	// FixtureInstanceE represents an instance with no Metadata, but with Userdata, and known IPs
	FixtureInstanceE *InstanceFixture
	// FixtureInstanceF represents an instance with no Metadata, but with Userdata, but no IPs
	FixtureInstanceF *InstanceFixture

	//go:embed instance-data/instance-a-metadata.json
	instanceAMetadata []byte

	//go:embed instance-data/instance-b-metadata.json
	instanceBMetadata []byte

	//go:embed instance-data/instance-c-metadata.json
	instanceCMetadata []byte

	//go:embed instance-data/instance-d-metadata.json
	instanceDMetadata []byte

	//go:embed instance-data/userdata-example-1.txt
	userdataExample1 []byte

	//go:embed instance-data/userdata-example-2.txt
	userdataExample2 []byte

	instanceAIPs = []string{"139.178.82.3", "2604:1380:4641:1f00::9/127", "10.70.17.8/31"}
	instanceBIPs = []string{"145.40.77.21", "2604:1380:4641:1f00::1/127", "10.1.2.8/29"}
	instanceEIPs = []string{"172.16.1.12"}
)

func addFixtures() error {
	ctx := context.TODO()

	if err := setupInstanceA(ctx, testDB); err != nil {
		return err
	}

	if err := setupInstanceB(ctx, testDB); err != nil {
		return err
	}

	if err := setupInstanceC(ctx, testDB); err != nil {
		return err
	}

	if err := setupInstanceD(ctx, testDB); err != nil {
		return err
	}

	if err := setupInstanceE(ctx, testDB); err != nil {
		return err
	}

	if err := setupInstanceF(ctx, testDB); err != nil {
		return err
	}

	return nil
}

func setupInstanceA(ctx context.Context, db *sqlx.DB) error {
	FixtureInstanceA = &InstanceFixture{
		InstanceID: instanceAUUID,
		HostIPs:    getIPs(instanceAIPs),
		InstanceMetadata: &models.InstanceMetadatum{
			ID:       instanceAUUID,
			Metadata: types.JSON(instanceAMetadata),
		},
		InstanceUserdata: &models.InstanceUserdatum{
			ID:       instanceAUUID,
			Userdata: null.NewBytes([]byte(userdataExample1), true),
		},
	}

	if err := FixtureInstanceA.InstanceMetadata.Insert(ctx, db, boil.Infer()); err != nil {
		return err
	}

	if err := FixtureInstanceA.InstanceUserdata.Insert(ctx, db, boil.Infer()); err != nil {
		return err
	}

	for _, address := range instanceAIPs {
		instanceIPAddress := models.InstanceIPAddress{
			InstanceID: instanceAUUID,
			Address:    address,
		}

		if err := instanceIPAddress.Insert(ctx, db, boil.Infer()); err != nil {
			return err
		}

		FixtureInstanceA.InstanceIPAddresses = append(FixtureInstanceA.InstanceIPAddresses, instanceIPAddress)
	}

	return nil
}

func setupInstanceB(ctx context.Context, db *sqlx.DB) error {
	FixtureInstanceB = &InstanceFixture{
		InstanceID: instanceBUUID,
		HostIPs:    getIPs(instanceBIPs),
		InstanceMetadata: &models.InstanceMetadatum{
			ID:       instanceBUUID,
			Metadata: types.JSON(instanceBMetadata),
		},
	}

	if err := FixtureInstanceB.InstanceMetadata.Insert(ctx, db, boil.Infer()); err != nil {
		return err
	}

	for _, address := range instanceBIPs {
		instanceIPAddress := models.InstanceIPAddress{
			InstanceID: instanceBUUID,
			Address:    address,
		}

		if err := instanceIPAddress.Insert(ctx, db, boil.Infer()); err != nil {
			return err
		}

		FixtureInstanceB.InstanceIPAddresses = append(FixtureInstanceB.InstanceIPAddresses, instanceIPAddress)
	}

	return nil
}

func setupInstanceC(ctx context.Context, db *sqlx.DB) error {
	FixtureInstanceC = &InstanceFixture{
		InstanceID: instanceCUUID,
		InstanceMetadata: &models.InstanceMetadatum{
			ID:       instanceCUUID,
			Metadata: types.JSON(instanceCMetadata),
		},
		InstanceUserdata: &models.InstanceUserdatum{
			ID:       instanceCUUID,
			Userdata: null.NewBytes([]byte(userdataExample2), true),
		},
	}

	if err := FixtureInstanceC.InstanceMetadata.Insert(ctx, db, boil.Infer()); err != nil {
		return err
	}

	if err := FixtureInstanceC.InstanceUserdata.Insert(ctx, db, boil.Infer()); err != nil {
		return err
	}

	return nil
}

func setupInstanceD(ctx context.Context, db *sqlx.DB) error {
	FixtureInstanceD = &InstanceFixture{
		InstanceID: instanceDUUID,
		InstanceMetadata: &models.InstanceMetadatum{
			ID:       instanceDUUID,
			Metadata: types.JSON(instanceDMetadata),
		},
	}

	if err := FixtureInstanceD.InstanceMetadata.Insert(ctx, db, boil.Infer()); err != nil {
		return err
	}

	return nil
}

func setupInstanceE(ctx context.Context, db *sqlx.DB) error {
	FixtureInstanceE = &InstanceFixture{
		InstanceID: instanceEUUID,
		HostIPs:    getIPs(instanceEIPs),
		InstanceUserdata: &models.InstanceUserdatum{
			ID:       instanceEUUID,
			Userdata: null.NewBytes([]byte(userdataExample2), true),
		},
	}

	if err := FixtureInstanceE.InstanceUserdata.Insert(ctx, db, boil.Infer()); err != nil {
		return err
	}

	for _, address := range instanceEIPs {
		instanceIPAddress := models.InstanceIPAddress{
			InstanceID: instanceEUUID,
			Address:    address,
		}

		if err := instanceIPAddress.Insert(ctx, db, boil.Infer()); err != nil {
			return err
		}

		FixtureInstanceE.InstanceIPAddresses = append(FixtureInstanceE.InstanceIPAddresses, instanceIPAddress)
	}

	return nil
}

func setupInstanceF(ctx context.Context, db *sqlx.DB) error {
	FixtureInstanceF = &InstanceFixture{
		InstanceID: instanceFUUID,
		InstanceUserdata: &models.InstanceUserdatum{
			ID:       instanceFUUID,
			Userdata: null.NewBytes([]byte(userdataExample2), true),
		},
	}

	if err := FixtureInstanceF.InstanceUserdata.Insert(ctx, db, boil.Infer()); err != nil {
		return err
	}

	return nil
}

func getIPs(addresses []string) []string {
	var ips []string

	for _, address := range addresses {
		ip, ipNet, err := net.ParseCIDR(address)
		if err == nil {
			// It was a CIDR address
			ones, bits := ipNet.Mask.Size()

			count := bits - ones

			networkIP := ipNet.IP

			if count == 0 {
				// If it's /32 or /128, just return the IP
				ips = append(ips, ip.String())
			} else {
				networkIP[len(networkIP)-1]++
				ips = append(ips, networkIP.String())

				if count > 1 {
					// Go ahead and add one more IP to test against
					networkIP[len(networkIP)-1]++
					ips = append(ips, networkIP.String())
				}
			}
		} else {
			// It was (probably) an IP without subnet specified in CIDR format. And
			// if it wasn't, well, we've added something bad in our fixture.
			ip = net.ParseIP(address)

			if ip != nil {
				ips = append(ips, ip.String())
			}
		}
	}

	return ips
}
