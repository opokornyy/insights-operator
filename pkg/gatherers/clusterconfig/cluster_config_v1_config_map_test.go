package clusterconfig

import (
	"context"
	"testing"

	installertypes "github.com/openshift/installer/pkg/types"
	"github.com/openshift/installer/pkg/types/baremetal"
	"github.com/openshift/installer/pkg/types/gcp"
	"github.com/openshift/installer/pkg/types/vsphere"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func Test_gatherClusterConfigV1(t *testing.T) {
	coreClient := kubefake.NewSimpleClientset()

	_, err := coreClient.CoreV1().ConfigMaps("kube-system").Create(context.Background(), &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster-config-v1",
		},
		Immutable: nil,
		Data: map[string]string{
			"install-config": "{}",
		},
		BinaryData: nil,
	}, metav1.CreateOptions{})
	assert.NoError(t, err)

	records, errs := gatherClusterConfigV1(context.Background(), coreClient.CoreV1())
	assert.Empty(t, errs)

	assert.Len(t, records, 1)
	assert.Equal(t, "config/configmaps/kube-system/cluster-config-v1/install-config", records[0].Name)

	data, err := records[0].Item.(ConfigMapAnonymizer).Marshal()
	assert.NoError(t, err)

	installConfig := `baseDomain: ""
metadata: {}
platform: {}
pullSecret: ""
`

	assert.Equal(t, installConfig, string(data))
}

func Test_anonymizeVSphere(t *testing.T) {
	platform := &vsphere.Platform{
		FailureDomains: []vsphere.FailureDomain{
			{
				Topology: vsphere.Topology{
					Datacenter: "datacenter-1",
				},
			},
		},
		VCenters: []vsphere.VCenter{
			{
				Username: "admin@vsphere.local",
				Password: "SuperSecretPassword123",
			},
		},
	}

	anonymizeVSphere(platform)

	// Verify failure domains are anonymized
	assert.Len(t, platform.FailureDomains, 1)
	assert.NotEqual(t, "datacenter-1", platform.FailureDomains[0].Topology.Datacenter)
	assert.NotEmpty(t, platform.FailureDomains[0].Topology.Datacenter)

	// Verify vCenter credentials are anonymized
	assert.Len(t, platform.VCenters, 1)
	assert.NotEqual(t, "admin@vsphere.local", platform.VCenters[0].Username)
	assert.NotEqual(t, "SuperSecretPassword123", platform.VCenters[0].Password)
	assert.NotEmpty(t, platform.VCenters[0].Username)
	assert.NotEmpty(t, platform.VCenters[0].Password)
}

func Test_anonymizeGCPConfig(t *testing.T) {
	platform := &gcp.Platform{
		Region:    "us-central1",
		ProjectID: "my-project-12345",
		DNS: &gcp.DNS{
			PrivateZone: &gcp.DNSZone{
				ProjectID: "dns-project-xyz",
			},
		},
	}

	anonymizeGCPConfig(platform)

	// Verify main fields are anonymized
	assert.NotEqual(t, "us-central1", platform.Region)
	assert.NotEqual(t, "my-project-12345", platform.ProjectID)
	assert.NotEmpty(t, platform.Region)
	assert.NotEmpty(t, platform.ProjectID)

	// Verify DNS private zone is anonymized
	assert.NotNil(t, platform.DNS)
	assert.NotNil(t, platform.DNS.PrivateZone)
	assert.NotEqual(t, "dns-project-xyz", platform.DNS.PrivateZone.ProjectID)
	assert.NotEmpty(t, platform.DNS.PrivateZone.ProjectID)
}

func Test_anonymizeInstallConfig_BareMetal(t *testing.T) {
	installConfig := &installertypes.InstallConfig{
		Platform: installertypes.Platform{
			BareMetal: &baremetal.Platform{
				Hosts: []*baremetal.Host{
					{
						BMC: baremetal.BMC{
							Username: "admin",
							Password: "password123",
						},
					},
					{
						BMC: baremetal.BMC{
							Username: "root",
							Password: "secret456",
						},
					},
				},
			},
		},
	}
	result := anonymizeInstallConfig(installConfig)

	// Verify BMC credentials are anonymized
	assert.NotNil(t, result.BareMetal)
	assert.Len(t, result.BareMetal.Hosts, 2)
	assert.NotEqual(t, "admin", result.BareMetal.Hosts[0].BMC.Username)
	assert.NotEqual(t, "password123", result.BareMetal.Hosts[0].BMC.Password)
	assert.NotEqual(t, "root", result.BareMetal.Hosts[1].BMC.Username)
	assert.NotEqual(t, "secret456", result.BareMetal.Hosts[1].BMC.Password)
	assert.NotEmpty(t, result.BareMetal.Hosts[0].BMC.Username)
	assert.NotEmpty(t, result.BareMetal.Hosts[0].BMC.Password)
}
