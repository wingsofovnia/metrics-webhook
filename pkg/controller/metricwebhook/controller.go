package metricwebhook

import (
	metricsv1alpha1 "github.com/wingsofovnia/metrics-webhook/pkg/apis/metrics/v1alpha1"

	"github.com/operator-framework/operator-sdk/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const ControllerName = "metricwebhook-controller"

// Add creates a new MetricWebhook Controller and adds it to the
// Manager. It will start it when the Manager is started.
func Add(mgr manager.Manager) error {
	reconciler, err := NewReconcileMetricWebhook(mgr)
	if err != nil {
		return err
	}
	return add(mgr, reconciler)
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource MetricWebhook
	err = c.Watch(&source.Kind{Type: &metricsv1alpha1.MetricWebhook{}}, &handler.EnqueueRequestForObject{}, predicate.GenerationChangedPredicate{})
	if err != nil {
		return err
	}

	return nil
}
