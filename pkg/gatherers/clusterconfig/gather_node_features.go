package clusterconfig

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
	nfdv1alpha1 "sigs.k8s.io/node-feature-discovery/api/nfd/v1alpha1"

	"github.com/openshift/insights-operator/pkg/record"
)

// List of attribute fields that is used to filter the NodeFeatureSpec
// fields to remove not required fields
var allowedAttributesFields []string = []string{
	"cpu.model",
	"cpu.topology",
	"memory.numa",
	"memory.hugepages",
}

// List of instance fields that is used to filter the NodeFeatureSpec
// fields to remove not required fields
var allowedInstacesFields []string = []string{
	"storage.block",
	"pci.device",
}

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

	records, err := getNodeFeaturesData(ctx, dynamicClient)
	if err != nil {
		klog.V(2).Infof("GatherNodeFeatures: Failed to get NodeFeatures data: %v", err)
		return nil, []error{err}
	}

	klog.V(2).Infof("GatherNodeFeatures: Successfully created %d records", len(records))
	return records, nil
}

func filterNodeFeatureSpec(nodeFeature *nfdv1alpha1.NodeFeatureSpec) *nfdv1alpha1.NodeFeatureSpec {
	filteredNofeFeatureSpec := nfdv1alpha1.NodeFeatureSpec{
		Features: nfdv1alpha1.Features{
			Attributes: make(map[string]nfdv1alpha1.AttributeFeatureSet),
			Instances:  make(map[string]nfdv1alpha1.InstanceFeatureSet),
		},
	}

	// Filter attribute keys
	for _, key := range allowedAttributesFields {
		if value, exists := nodeFeature.Features.Attributes[key]; exists {
			filteredNofeFeatureSpec.Features.Attributes[key] = value
		}
	}

	// Filter instance keys
	for _, key := range allowedInstacesFields {
		if value, exists := nodeFeature.Features.Instances[key]; exists {
			filteredNofeFeatureSpec.Features.Instances[key] = value
		}
	}

	return &filteredNofeFeatureSpec
}

func getNodeFeaturesData(ctx context.Context, dynamicClient dynamic.Interface) ([]record.Record, error) {
	nodeFeaturesList, err := dynamicClient.Resource(nodeFeatureResource).Namespace("openshift-nfd").List(ctx, metav1.ListOptions{})
	if errors.IsNotFound(err) {
		klog.V(2).Infof("GatherNodeFeatures: NodeFeatures resource not found in openshift-nfd namespace (may not be installed)")
		return nil, nil
	}
	if err != nil {
		klog.V(2).Infof("GatherNodeFeatures: Failed to list NodeFeatures: %v", err)
		return nil, err
	}

	var records []record.Record

	for _, nodeFeature := range nodeFeaturesList.Items {
		// Marshal the unstructured object to JSON
		data, err := json.Marshal(nodeFeature.Object)
		if err != nil {
			klog.V(2).Infof("GatherNodeFeatures: Failed to marshal NodeFeature %s: %v", nodeFeature.GetName(), err)
			continue
		}

		// Unmarshal into our NodeFeature struct
		var nodeFeature nfdv1alpha1.NodeFeature
		if err := json.Unmarshal(data, &nodeFeature); err != nil {
			klog.V(2).Infof("GatherNodeFeatures: Failed to unmarshal NodeFeature %s: %v", nodeFeature.GetName(), err)
			continue
		}

		// Filter only allowed spec fields
		filteredNodeFeatureSpec := filterNodeFeatureSpec(&nodeFeature.Spec)

		recordName := fmt.Sprintf("config/nodefeatures/%s",
			nodeFeature.Name,
		)
		klog.V(4).Infof("GatherNodeFeatures: Adding record: %s", recordName)

		records = append(records, record.Record{
			Name: recordName,
			Item: record.JSONMarshaller{Object: *filteredNodeFeatureSpec},
		})
	}

	return records, nil
}
