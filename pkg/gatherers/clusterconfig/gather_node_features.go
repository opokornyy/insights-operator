package clusterconfig

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"

	"github.com/openshift/insights-operator/pkg/record"
)

// GatherNodeFeatures Collects `nodefeatures.nfd.k8s-sigs.io` custom resources
// from the openshift-nfd namespace.
//
// ### API Reference
// None
//
// ### Sample data
// - docs/insights-archive-sample/customresources/nfd.k8s-sigs.io/nodefeatures/openshift-nfd/{name}.json
//
// ### Location in archive
// - `customresources/nfd.k8s-sigs.io/nodefeatures/{namespace}/{name}.json`
//
// ### Config ID
// `clusterconfig/node_features`
//
// ### Released version
// - TBD
//
// ### Backported versions
// None
//
// ### Changes
// None
func (g *Gatherer) GatherNodeFeatures(ctx context.Context) ([]record.Record, []error) {
	gatherDynamicClient, err := dynamic.NewForConfig(g.gatherKubeConfig)
	if err != nil {
		return nil, []error{err}
	}

	return gatherNodeFeatures(ctx, gatherDynamicClient)
}

func gatherNodeFeatures(ctx context.Context, dynamicClient dynamic.Interface) ([]record.Record, []error) {
	klog.V(2).Infof("GatherNodeFeatures: Starting to gather NodeFeatures from openshift-nfd namespace")
	klog.V(4).Infof("GatherNodeFeatures: Using resource GVR: %s/%s/%s",
		nodeFeatureResource.Group, nodeFeatureResource.Version, nodeFeatureResource.Resource)

	nodeFeaturesList, err := dynamicClient.Resource(nodeFeatureResource).Namespace("openshift-nfd").List(ctx, metav1.ListOptions{})
	if errors.IsNotFound(err) {
		klog.V(2).Infof("GatherNodeFeatures: NodeFeatures resource not found in openshift-nfd namespace (may not be installed)")
		return nil, nil
	}
	if err != nil {
		klog.V(2).Infof("GatherNodeFeatures: Failed to list NodeFeatures: %v", err)
		return nil, []error{err}
	}

	klog.V(2).Infof("GatherNodeFeatures: Found %d NodeFeatures resources", len(nodeFeaturesList.Items))

	var records []record.Record

	for i, nodeFeature := range nodeFeaturesList.Items {
		recordName := fmt.Sprintf("config/nodefeatures/%s/%s/%s/%s",
			nodeFeatureResource.Group,
			nodeFeatureResource.Resource,
			nodeFeature.GetNamespace(),
			nodeFeature.GetName(),
		)
		klog.V(4).Infof("GatherNodeFeatures: Adding record %d: %s", i+1, recordName)

		records = append(records, record.Record{
			Name: recordName,
			Item: record.ResourceMarshaller{Resource: &nodeFeaturesList.Items[i]},
		})
	}

	klog.V(2).Infof("GatherNodeFeatures: Successfully created %d records", len(records))
	return records, nil
}
