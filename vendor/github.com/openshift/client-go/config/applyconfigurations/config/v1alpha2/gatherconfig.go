// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha2

import (
	configv1alpha2 "github.com/openshift/api/config/v1alpha2"
)

// GatherConfigApplyConfiguration represents a declarative configuration of the GatherConfig type for use
// with apply.
type GatherConfigApplyConfiguration struct {
	DataPolicy []configv1alpha2.DataPolicyOption `json:"dataPolicy,omitempty"`
	Gatherers  *GatherersApplyConfiguration      `json:"gatherers,omitempty"`
	Storage    *StorageApplyConfiguration        `json:"storage,omitempty"`
}

// GatherConfigApplyConfiguration constructs a declarative configuration of the GatherConfig type for use with
// apply.
func GatherConfig() *GatherConfigApplyConfiguration {
	return &GatherConfigApplyConfiguration{}
}

// WithDataPolicy adds the given value to the DataPolicy field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the DataPolicy field.
func (b *GatherConfigApplyConfiguration) WithDataPolicy(values ...configv1alpha2.DataPolicyOption) *GatherConfigApplyConfiguration {
	for i := range values {
		b.DataPolicy = append(b.DataPolicy, values[i])
	}
	return b
}

// WithGatherers sets the Gatherers field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Gatherers field is set to the value of the last call.
func (b *GatherConfigApplyConfiguration) WithGatherers(value *GatherersApplyConfiguration) *GatherConfigApplyConfiguration {
	b.Gatherers = value
	return b
}

// WithStorage sets the Storage field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Storage field is set to the value of the last call.
func (b *GatherConfigApplyConfiguration) WithStorage(value *StorageApplyConfiguration) *GatherConfigApplyConfiguration {
	b.Storage = value
	return b
}
