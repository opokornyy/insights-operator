package configobserver

import (
	"context"
	"testing"

	"github.com/openshift/api/config/v1alpha2"
	fakeConfigCli "github.com/openshift/client-go/config/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInsightsDataGatherSync(t *testing.T) {
	tests := []struct {
		name                        string
		insightsDatagatherToUpdated *v1alpha2.InsightsDataGather
		expectedGatherConfig        *v1alpha2.GatherConfig
		expectedDisable             bool
	}{
		{
			name: "Obfuscation configured and some disabled gatherers",
			insightsDatagatherToUpdated: &v1alpha2.InsightsDataGather{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: v1alpha2.InsightsDataGatherSpec{
					GatherConfig: v1alpha2.GatherConfig{
						DataPolicy: []v1alpha2.DataPolicyOption{
							v1alpha2.DataPolicyOptionObfuscateNetworking,
						},
						Gatherers: v1alpha2.Gatherers{
							Mode: v1alpha2.GatheringModeCustom,
							Custom: &v1alpha2.Custom{
								Configs: []v1alpha2.GathererConfig{
									{
										Name:  "fooBar",
										State: v1alpha2.GathererStateDisabled,
									},
									{
										Name:  "barrGather",
										State: v1alpha2.GathererStateDisabled,
									},
								},
							},
						},
					},
				},
			},
			expectedGatherConfig: &v1alpha2.GatherConfig{
				DataPolicy: []v1alpha2.DataPolicyOption{
					v1alpha2.DataPolicyOptionObfuscateNetworking,
				},
				Gatherers: v1alpha2.Gatherers{
					Mode: v1alpha2.GatheringModeCustom,
					Custom: &v1alpha2.Custom{
						Configs: []v1alpha2.GathererConfig{
							{
								Name:  "fooBar",
								State: v1alpha2.GathererStateDisabled,
							},
							{
								Name:  "barrGather",
								State: v1alpha2.GathererStateDisabled,
							},
						},
					},
				},
			},
			expectedDisable: false,
		},
		{
			name: "Gathering disabled and no obfuscation",
			insightsDatagatherToUpdated: &v1alpha2.InsightsDataGather{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: v1alpha2.InsightsDataGatherSpec{
					GatherConfig: v1alpha2.GatherConfig{
						DataPolicy: []v1alpha2.DataPolicyOption{
							v1alpha2.DataPolicyOptionObfuscateNetworking,
						},
						Gatherers: v1alpha2.Gatherers{
							Mode: v1alpha2.GatheringModeNone,
						},
					},
				},
			},
			expectedGatherConfig: &v1alpha2.GatherConfig{
				DataPolicy: []v1alpha2.DataPolicyOption{
					v1alpha2.DataPolicyOptionObfuscateNetworking,
				},
				Gatherers: v1alpha2.Gatherers{
					Mode: v1alpha2.GatheringModeNone,
				},
			},
			expectedDisable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			insightDefaultConfig := &v1alpha2.InsightsDataGather{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
			}

			client := fakeConfigCli.NewSimpleClientset(insightDefaultConfig)
			idgObserver := insightsDataGatherController{
				gatherConfig: &insightDefaultConfig.Spec.GatherConfig,
				cli:          client.ConfigV1alpha2(),
			}
			err := idgObserver.sync(context.Background(), nil)
			assert.NoError(t, err)

			assert.Equal(t, &insightDefaultConfig.Spec.GatherConfig, idgObserver.GatherConfig())
			_, err = idgObserver.cli.InsightsDataGathers().Update(context.Background(), tt.insightsDatagatherToUpdated, metav1.UpdateOptions{})
			assert.NoError(t, err)
			err = idgObserver.sync(context.Background(), nil)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedGatherConfig, idgObserver.GatherConfig())
			assert.Equal(t, tt.expectedDisable, idgObserver.GatherDisabled())
		})
	}
}
