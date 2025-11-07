package periodic

import (
	"context"
	"slices"
	"strings"

	insightsv1alpha2 "github.com/openshift/api/insights/v1alpha2"
	insightsInformers "github.com/openshift/client-go/insights/informers/externalversions"
	insightsListers "github.com/openshift/client-go/insights/listers/insights/v1alpha2"
	"github.com/openshift/insights-operator/pkg/controller/status"
	"github.com/openshift/library-go/pkg/controller/factory"
	"github.com/openshift/library-go/pkg/operator/events"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

const (
	periodicGatheringPrefix = "periodic-gathering-"
)

// DataGatherInformer is an interface providing information
// about newly create DataGather resources
type DataGatherInformer interface {
	factory.Controller
	DataGatherCreated() <-chan string
	// This could be used to list all the gatherers with cache
	// which means there would be no api request, maybe we would
	// not needs this function
	Lister() insightsListers.DataGatherLister
	// This should be triggered on the status change of the DataGather
	// That is important for checking if the DataGathering is finished
	DatagaGatherStatusChanged() <-chan struct{}
}

// dataGatherController is type implementing DataGatherInformer
type dataGatherController struct {
	factory.Controller
	ch            chan string
	lister        insightsListers.DataGatherLister
	statusChanged chan struct{}
}

// NewDataGatherInformer creates a new instance of the DataGatherInformer interface
func NewDataGatherInformer(eventRecorder events.Recorder, insightsInf insightsInformers.SharedInformerFactory) (DataGatherInformer, error) {
	inf := insightsInf.Insights().V1alpha2().DataGathers().Informer()
	lister := insightsInf.Insights().V1alpha2().DataGathers().Lister()

	dgCtrl := &dataGatherController{
		ch:            make(chan string),
		statusChanged: make(chan struct{}, 10), // buffered
		lister:        lister,
	}
	_, err := inf.AddEventHandler(dgCtrl.eventHandler())
	if err != nil {
		return nil, err
	}

	ctrl := factory.New().WithInformers(inf).
		WithSync(dgCtrl.sync).
		ToController("DataGatherInformer", eventRecorder)

	dgCtrl.Controller = ctrl
	return dgCtrl, nil
}

func (d *dataGatherController) sync(_ context.Context, _ factory.SyncContext) error {
	return nil
}

// eventHandler returns a new ResourceEventHandler that handles the DataGather resources
// addition events. Resources with the prefix "periodic-gathering-" are filtered out to avoid conflicts
// with periodic data gathering.
func (d *dataGatherController) eventHandler() cache.ResourceEventHandler {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			dgMetadata, err := meta.Accessor(obj)
			if err != nil {
				klog.Errorf("Can't read metadata of newly added DataGather resource: %v", err)
				return
			}
			// filter out dataGathers created for periodic gathering
			if strings.HasPrefix(dgMetadata.GetName(), periodicGatheringPrefix) {
				return
			}
			d.ch <- dgMetadata.GetName()
		},
		// This is required, because we need to check if there
		// are some dataGathers that were not started because of
		// the limit of running gatherings at the same time
		UpdateFunc: func(oldObj, newObj interface{}) {
			klog.Info("UpdateFunc informer")

			oldDG, ok := oldObj.(*insightsv1alpha2.DataGather)
			if !ok {
				return
			}

			newDG, ok := newObj.(*insightsv1alpha2.DataGather)
			if !ok {
				return
			}

			// filter out dataGathers created for periodic gathering
			if strings.HasPrefix(newDG.GetName(), periodicGatheringPrefix) {
				return
			}

			// Maybe if we can first check the newCondition that it is in a
			// finished state we could avoid checking oldCondition
			newCondition := status.GetConditionByType(newDG, status.Progressing)
			// missing condition
			if newCondition == nil {
				return
			}

			// helper slice - define it globally maybe?
			finishedReasons := []string{status.GatheringFailedReason, status.GatheringSucceededReason}
			if !slices.Contains(finishedReasons, newCondition.Reason) {
				klog.Infof("newCondition reason is not finished")
				return
			}

			oldCondition := status.GetConditionByType(oldDG, status.Progressing)
			// missing condition
			if oldCondition == nil {
				return
			}

			klog.Infof("old conditions: %v", oldCondition)
			klog.Infof("new conditions: %v", newCondition)

			// TODO
			// Maybe it should be enough to check only Progressing condition
			// In the next step we could check if the confition changed to Succeeded/Finished
			// and not only started
			// Check if status.conditions changed (job completed or progressed)
			if oldCondition != newCondition {
				klog.Infof("DataGather %s status changed, signaling reconciliation", newDG.Name)
				// Non-blocking send
				select {
				case d.statusChanged <- struct{}{}:
				default:
					// Channel full, signal already pending
				}
			}
		},
	}
}

// DataGatherCreated returns a channel providing the name of
// newly created DataGather resource
func (d *dataGatherController) DataGatherCreated() <-chan string {
	return d.ch
}

// Lister returns a DataGatherLister that can be used to query
// the informer's cache without making API requests
func (d *dataGatherController) Lister() insightsListers.DataGatherLister {
	return d.lister
}

func (d *dataGatherController) DatagaGatherStatusChanged() <-chan struct{} {
	return d.statusChanged
}
