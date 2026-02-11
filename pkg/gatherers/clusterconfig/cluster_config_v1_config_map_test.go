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

func Test_anonymizeFencing_EmptyCredentials(t *testing.T) {
	fencing := &installertypes.Fencing{
		Credentials: []*installertypes.Credential{},
	}

	anonymizeFencing(fencing)

	// Verify empty credentials slice is handled
	assert.NotNil(t, fencing)
	assert.Len(t, fencing.Credentials, 0)
}

func Test_anonymizeFencing_WithCredentials(t *testing.T) {
	type testCase struct {
		name      string
		fencing   *installertypes.Fencing
		checkData bool
	}

	testCases := []testCase{
		{
			name: "Anonymize fencing credentials",
			fencing: &installertypes.Fencing{
				Credentials: []*installertypes.Credential{
					{
						HostName: "bmc1.example.com",
						Username: "admin",
						Password: "secretPassword123",
						Address:  "192.168.1.100",
					},
				},
			},
			checkData: true,
		},
		{
			name: "Empty credentials does not panic",
			fencing: &installertypes.Fencing{
				Credentials: []*installertypes.Credential{},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			anonymizeFencing(tt.fencing)

			if tt.checkData {
				// Verify first credential is anonymized
				assert.Len(t, tt.fencing.Credentials, 1)
				assert.Equal(t, "xxxxxxxxxxxxxxxx", tt.fencing.Credentials[0].HostName)
				assert.Equal(t, "xxxxx", tt.fencing.Credentials[0].Username)
				assert.Equal(t, "xxxxxxxxxxxxxxxxx", tt.fencing.Credentials[0].Password)
				assert.Equal(t, "xxxxxxxxxxxxx", tt.fencing.Credentials[0].Address)
			}
		})
	}
}

func Test_anonymizeInstallConfig_ControlPlaneFencing(t *testing.T) {
	installConfig := &installertypes.InstallConfig{
		ControlPlane: &installertypes.MachinePool{
			Name: "master",
			Fencing: &installertypes.Fencing{
				Credentials: []*installertypes.Credential{
					{
						HostName: "master-bmc.example.com",
						Username: "admin",
						Password: "masterPassword",
						Address:  "10.0.0.1",
					},
				},
			},
		},
	}

	result := anonymizeInstallConfig(installConfig)

	// Verify ControlPlane fencing credentials are anonymized
	assert.NotNil(t, result.ControlPlane)
	assert.NotNil(t, result.ControlPlane.Fencing)
	assert.Len(t, result.ControlPlane.Fencing.Credentials, 1)
	assert.Equal(t, "xxxxxxxxxxxxxxxxxxxxxx", result.ControlPlane.Fencing.Credentials[0].HostName)
	assert.Equal(t, "xxxxx", result.ControlPlane.Fencing.Credentials[0].Username)
	assert.Equal(t, "xxxxxxxxxxxxxx", result.ControlPlane.Fencing.Credentials[0].Password)
	assert.Equal(t, "xxxxxxxx", result.ControlPlane.Fencing.Credentials[0].Address)
}

func Test_anonymizeInstallConfig_ArbiterFencing(t *testing.T) {
	installConfig := &installertypes.InstallConfig{
		Arbiter: &installertypes.MachinePool{
			Name: "arbiter",
			Fencing: &installertypes.Fencing{
				Credentials: []*installertypes.Credential{
					{
						HostName: "arbiter-bmc.example.com",
						Username: "arbiter-admin",
						Password: "arbiterPassword",
						Address:  "10.0.0.2",
					},
				},
			},
		},
	}

	result := anonymizeInstallConfig(installConfig)

	// Verify Arbiter fencing credentials are anonymized
	assert.NotNil(t, result.Arbiter)
	assert.NotNil(t, result.Arbiter.Fencing)
	assert.Len(t, result.Arbiter.Fencing.Credentials, 1)
	assert.Equal(t, "xxxxxxxxxxxxxxxxxxxxxxx", result.Arbiter.Fencing.Credentials[0].HostName)
	assert.Equal(t, "xxxxxxxxxxxxx", result.Arbiter.Fencing.Credentials[0].Username)
	assert.Equal(t, "xxxxxxxxxxxxxxx", result.Arbiter.Fencing.Credentials[0].Password)
	assert.Equal(t, "xxxxxxxx", result.Arbiter.Fencing.Credentials[0].Address)
}

func Test_anonymizeInstallConfig_ComputeFencing(t *testing.T) {
	installConfig := &installertypes.InstallConfig{
		Compute: []installertypes.MachinePool{
			{
				Name: "worker",
				Fencing: &installertypes.Fencing{
					Credentials: []*installertypes.Credential{
						{
							HostName: "worker1-bmc.example.com",
							Username: "worker-admin",
							Password: "workerPassword1",
							Address:  "10.0.0.10",
						},
					},
				},
			},
			{
				Name: "worker",
				Fencing: &installertypes.Fencing{
					Credentials: []*installertypes.Credential{},
				},
			},
		},
	}

	result := anonymizeInstallConfig(installConfig)

	// Verify first compute node fencing credentials are anonymized
	assert.Len(t, result.Compute, 2)
	assert.NotNil(t, result.Compute[0].Fencing)
	assert.Len(t, result.Compute[0].Fencing.Credentials, 1)
	assert.Equal(t, "xxxxxxxxxxxxxxxxxxxxxxx", result.Compute[0].Fencing.Credentials[0].HostName)
	assert.Equal(t, "xxxxxxxxxxxx", result.Compute[0].Fencing.Credentials[0].Username)
	assert.Equal(t, "xxxxxxxxxxxxxxx", result.Compute[0].Fencing.Credentials[0].Password)
	assert.Equal(t, "xxxxxxxxx", result.Compute[0].Fencing.Credentials[0].Address)
}

func Test_anonymizeInstallConfig_MultipleMachinePoolsWithFencing(t *testing.T) {
	installConfig := &installertypes.InstallConfig{
		ControlPlane: &installertypes.MachinePool{
			Name: "master",
			Fencing: &installertypes.Fencing{
				Credentials: []*installertypes.Credential{
					{
						HostName: "master-bmc.example.com",
						Username: "master-admin",
						Password: "masterPass",
						Address:  "10.0.0.1",
					},
				},
			},
		},
		Arbiter: &installertypes.MachinePool{
			Name: "arbiter",
			Fencing: &installertypes.Fencing{
				Credentials: []*installertypes.Credential{
					{
						HostName: "arbiter-bmc.example.com",
						Username: "arbiter-admin",
						Password: "arbiterPass",
						Address:  "10.0.0.2",
					},
				},
			},
		},
		Compute: []installertypes.MachinePool{
			{
				Name: "worker",
				Fencing: &installertypes.Fencing{
					Credentials: []*installertypes.Credential{
						{
							HostName: "worker-bmc.example.com",
							Username: "worker-admin",
							Password: "workerPass",
							Address:  "10.0.0.10",
						},
					},
				},
			},
		},
	}

	result := anonymizeInstallConfig(installConfig)

	// Verify all machine pools have anonymized fencing credentials
	assert.NotNil(t, result.ControlPlane)
	assert.NotNil(t, result.ControlPlane.Fencing)
	assert.Equal(t, "xxxxxxxxxxxxxxxxxxxxxx", result.ControlPlane.Fencing.Credentials[0].HostName)

	assert.NotNil(t, result.Arbiter)
	assert.NotNil(t, result.Arbiter.Fencing)
	assert.Equal(t, "xxxxxxxxxxxxxxxxxxxxxxx", result.Arbiter.Fencing.Credentials[0].HostName)

	assert.Len(t, result.Compute, 1)
	assert.NotNil(t, result.Compute[0].Fencing)
	assert.Equal(t, "xxxxxxxxxxxxxxxxxxxxxx", result.Compute[0].Fencing.Credentials[0].HostName)
}
