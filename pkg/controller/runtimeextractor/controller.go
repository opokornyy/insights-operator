// Package runtimeextractor manages the lifecycle of the runtime-extractor deployment
// based on the DisableRuntimeExtractor configuration option.
package runtimeextractor

import (
	"context"

	"github.com/openshift/insights-operator/pkg/config"
	"k8s.io/klog/v2"
)

type ConfigNotifier interface {
	ConfigChanged() (<-chan struct{}, func())
	Config() *config.InsightsConfiguration
}

type UpdateNotifier interface {
	// TODO: add method that would notify controller about
	// cluster update
}

type runtimeExtractorController struct {
	// config watcher
	config ConfigNotifier
	// update watcher
	update UpdateNotifier
	// client to work with deployments
}

// NewRuntimeExtractorController is a constructor for runtimeExtractorController
// that is in charge of runtime-extractor deployment lifecycle
func NewRuntimeExtractorController(configNotifier ConfigNotifier) *runtimeExtractorController {
	return &runtimeExtractorController{
		config: configNotifier,
	}
}

func (re *runtimeExtractorController) Run(ctx context.Context) {
	klog.Info("[RuntimeExtractorController]: hello world")

	// TODO: default setup, the IO pod could be restarted on random
	// occasions, make sure to handle that
	re.handleConfigChange(ctx)

	disableRuntimeExtractor := re.config.Config().DataReporting.DisableRuntimeExtractor
	klog.Infof("[RuntimeExtractorController]: disableRuntimeExtractor: %t", disableRuntimeExtractor)

	configChan, configClose := re.config.ConfigChanged()
	defer configClose()

	// Check ConfigMap if the DisableRuntimeExtractor is set
	// Based on that Create/Delete the RuntimeExtractor deployment
	// Also watch for Updates if the deployment needs some changes
	for {
		select {
		case <-configChan:
			klog.Info("[RuntimeExtractorController]: Configuration Changed")
			// Check if disableRuntimeExtractor was changed
			re.handleConfigChange(ctx)
		case <-ctx.Done():
			klog.Info("[RuntimeExtractorController]: Context Done")
		}
	}
}

func (re *runtimeExtractorController) handleConfigChange(ctx context.Context) {
	cfg := re.config.Config()

	if cfg.DataReporting.DisableRuntimeExtractor {
		re.deleteDeployment(ctx)
	} else {
		re.createDeployment(ctx)
	}
}

func (re *runtimeExtractorController) isCreated(ctx context.Context) bool {
	// TODO: fetch deployment
	return true
}

func (re *runtimeExtractorController) createDeployment(ctx context.Context) {
	klog.Info("[RuntimeExtractorController]: Create Deployment")

	if re.isCreated(ctx) {
		return
	}
	// Create
}

func (re *runtimeExtractorController) deleteDeployment(ctx context.Context) {
	klog.Info("[RuntimeExtractorController]: Delete Deployment")

	if !re.isCreated(ctx) {
		return
	}
	// Delete
}

func (re *runtimeExtractorController) updateDeployment(ctx context.Context) {
	klog.Info("[RuntimeExtractorController]: Update Deployment")
}
