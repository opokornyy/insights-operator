package configobserver

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/openshift/api/config/v1alpha1"
	configCliv1alpha1 "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1alpha1"
	configinformers "github.com/openshift/client-go/config/informers/externalversions"
	"github.com/openshift/library-go/pkg/controller/factory"
	"github.com/openshift/library-go/pkg/operator/events"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

type InsightsDataGatherObserver interface {
	factory.Controller
	GatherConfig() *v1alpha1.GatherConfig
	GatherDisabled() bool
}

type insightsDataGatherController struct {
	factory.Controller
	lock         sync.Mutex
	cli          configCliv1alpha1.ConfigV1alpha1Interface
	gatherConfig *v1alpha1.GatherConfig
}

func NewInsightsDataGatherObserver(kubeConfig *rest.Config,
	eventRecorder events.Recorder,
	configInformer configinformers.SharedInformerFactory,
) (InsightsDataGatherObserver, error) {
	inf := configInformer.Config().V1alpha1().InsightsDataGathers().Informer()
	configV1Alpha1Cli, err := configCliv1alpha1.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}
	c := &insightsDataGatherController{
		cli: configV1Alpha1Cli,
	}

	insightDataGatherConf, err := c.cli.InsightsDataGathers().Get(context.Background(), "cluster", metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Cannot read API gathering configuration: %v", err)
	}
	c.gatherConfig = &insightDataGatherConf.Spec.GatherConfig

	ctrl := factory.New().WithInformers(inf).
		WithSync(c.sync).
		ResyncEvery(2*time.Minute).
		ToController("InsightsDataGatherObserver", eventRecorder)
	c.Controller = ctrl
	return c, nil
}

func (i *insightsDataGatherController) sync(ctx context.Context, _ factory.SyncContext) error {
	insightDataGatherConf, err := i.cli.InsightsDataGathers().Get(ctx, "cluster", metav1.GetOptions{})
	if err != nil {
		return err
	}
	i.gatherConfig = &insightDataGatherConf.Spec.GatherConfig
	return nil
}

// GatherConfig provides the complete gather config in a thread-safe way.
func (i *insightsDataGatherController) GatherConfig() *v1alpha1.GatherConfig {
	i.lock.Lock()
	defer i.lock.Unlock()
	return i.gatherConfig
}

// GatherDisabled tells whether data gathering is disabled or not
func (i *insightsDataGatherController) GatherDisabled() bool {
	i.lock.Lock()
	defer i.lock.Unlock()

	if slices.Contains(i.gatherConfig.DisabledGatherers, v1alpha1.DisabledGatherer("all")) ||
		slices.Contains(i.gatherConfig.DisabledGatherers, v1alpha1.DisabledGatherer("ALL")) {
		return true
	}

	return false
}
