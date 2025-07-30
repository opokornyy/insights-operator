package anonymization

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/openshift/insights-operator/pkg/record"
	"k8s.io/klog/v2"
)

const workloadRegex = `(?m)(,?\"%s\":\"[^\"]*\")`

type WorkloadAnonymizer struct {
	filterProps []*regexp.Regexp
}

func NewWorkloadAnonymizer() *WorkloadAnonymizer {
	return &WorkloadAnonymizer{
		filterProps: []*regexp.Regexp{
			regexp.MustCompile(fmt.Sprintf(workloadRegex, "name")),
			regexp.MustCompile(fmt.Sprintf(workloadRegex, "namespace")),
		},
	}
}

func (wa *WorkloadAnonymizer) isEnabled() bool {
	return true
}

func (wa *WorkloadAnonymizer) anonymizeData(memoryRecord *record.MemoryRecord) {
	klog.Info("WorkloadAnonymizer")
	klog.Infof("Before anonymizing Data: %s", string(memoryRecord.Data))

	// Anonymization with the use of regex can not be used here anymore
	// because it leaves the dangling comma in place. An example would
	// be `{ "name": "name-1", "data": 1} => {, "data": 1}`.
	// We probably need to serialize it and then iterate and remove the
	// name and namespace fields from the struct. I am not sure how much would the performance
	// degrade - there would probably need to be some recursive calls on the json data...
	var jsonData map[string]interface{}

	// TODO: only metrics with the DVO prefix should be gathered.
	err := json.Unmarshal(memoryRecord.Data, &jsonData)
	if err != nil {
		klog.Errorf("unmarshal error: %e", err)
	}

	// Remove the keys that should be anonymized
	deleteKeysRecursively(jsonData, []string{"name", "namespace"})

	memoryRecord.Data, err = json.Marshal(jsonData)
	if err != nil {
		klog.Errorf("marshal error: %e", err)
	}

	klog.Infof("Anonymized data: %s", string(memoryRecord.Data))
}

// deleteKeysRecursively traverses the data structure and removes the specified keys.
func deleteKeysRecursively(data interface{}, keysToRemove []string) {
	switch v := data.(type) {
	// If it's a map, delete the keys and then recurse into its values.
	case map[string]interface{}:
		for _, keyToDel := range keysToRemove {
			delete(v, keyToDel)
		}
		for _, val := range v {
			deleteKeysRecursively(val, keysToRemove)
		}
	// If it's a slice, recurse into each item.
	case []interface{}:
		for _, item := range v {
			deleteKeysRecursively(item, keysToRemove)
		}
	}
}

type NetworkAnonymizer struct{}

func NewNetworkAnonymizer() *NetworkAnonymizer {
	return &NetworkAnonymizer{}
}

func (na *NetworkAnonymizer) isEnabled() bool {
	return true
}

func (na *NetworkAnonymizer) anonymizeData(memoryRecord *record.MemoryRecord) {
	klog.Info("NetworkAnonymizer")
	klog.Infof("Anonymizing Data: %s", string(memoryRecord.Data))
	// TODO: implement the network anonymization here
}
