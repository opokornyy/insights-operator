package sca

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

var testNodes = []v1.Node{
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-x86_64",
			Labels: map[string]string{
				// Node marked as control plane
				"node-role.kubernetes.io/control-plane": "",
			},
		},
		Status: v1.NodeStatus{
			NodeInfo: v1.NodeSystemInfo{
				Architecture: "amd64",
			},
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-ppc64le",
		},
		Status: v1.NodeStatus{
			NodeInfo: v1.NodeSystemInfo{
				Architecture: "ppc64le",
			},
		},
	},
}

func Test_SCAController_GatherMultipleArchitectures(t *testing.T) {
	kube := kubefake.NewSimpleClientset()
	coreClient := kube.CoreV1()

	// Create test nodes
	for _, node := range testNodes {
		_, err := coreClient.Nodes().Create(context.Background(), &node, metav1.CreateOptions{})
		assert.NoError(t, err)
	}

	expectedArchitectures := map[string]struct{}{
		"x86_64":  {},
		"ppc64le": {},
	}

	scaController := New(coreClient, nil, nil)
	clusterArchitectures, err := scaController.gatherArchitectures(context.Background())
	assert.NoError(t, err, "failed to gather architectures")

	// check the correct control plane arch was found
	assert.Equal(t, "x86_64", clusterArchitectures.ControlPlaneArch, "incorrect control plane architecture")

	assert.Len(t, clusterArchitectures.NodeArchitectures, len(testNodes), "unexpected number of architectures")
	assert.Equal(t, expectedArchitectures, clusterArchitectures.NodeArchitectures, "unexpected architectures")
}
