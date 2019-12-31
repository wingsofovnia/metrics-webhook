package metricwebhook

import (
	"context"
	"fmt"
	metricsv1alpha1 "github.com/wingsofovnia/metrics-webhook/pkg/apis/metrics/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/controller/podautoscaler/metrics"
	metricsclientv1beta1 "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
	"k8s.io/metrics/pkg/client/custom_metrics"
	"k8s.io/metrics/pkg/client/external_metrics"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_metricwebhook")

// Add creates a new MetricWebhook Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	reconciler, err := newReconciler(mgr)
	if err != nil {
		return err
	}
	return add(mgr, reconciler)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) (reconcile.Reconciler, error) {
	clientConfig := mgr.GetConfig()
	restMapper := mgr.GetRESTMapper()
	clientSet, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return &ReconcileMetricWebhook{}, err
	}

	apisGetter := custom_metrics.NewAvailableAPIsGetter(clientSet.Discovery())
	metricsClient := metrics.NewRESTMetricsClient(
		metricsclientv1beta1.NewForConfigOrDie(clientConfig),
		custom_metrics.NewForConfig(clientConfig, restMapper, apisGetter),
		external_metrics.NewForConfigOrDie(clientConfig),
	)
	metricValuesClient := NewMetricValuesClient(metricsClient, clientSet)

	return &ReconcileMetricWebhook{
		client:            mgr.GetClient(),
		scheme:            mgr.GetScheme(),
		metricsClient:     metricValuesClient,
		metricAlertClient: NewDefaultMetricAlertClient(),
	}, nil
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("metricwebhook-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource MetricWebhook
	err = c.Watch(&source.Kind{Type: &metricsv1alpha1.MetricWebhook{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileMetricWebhook implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileMetricWebhook{}

// ReconcileMetricWebhook reconciles a MetricWebhook object
type ReconcileMetricWebhook struct {
	client            client.Client
	scheme            *runtime.Scheme
	metricsClient     *MetricValuesClient
	metricAlertClient *MetricAlertClient
}

// Reconcile reads that state of the cluster for a MetricWebhook object and makes changes based on the state read
// and what is in the MetricWebhook.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileMetricWebhook) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling MetricWebhook")

	// Fetch the MetricWebhook instance
	metricWebhook := &metricsv1alpha1.MetricWebhook{}
	err := r.client.Get(context.TODO(), request.NamespacedName, metricWebhook)
	if err != nil {
		reqLogger.Error(err, "failed to fetch MetricWebhook instance")
		if errors.IsNotFound(err) {
			return reconcile.Result{Requeue: false}, nil
		}
		return reconcile.Result{}, err
	}
	defer func() {
		err = r.client.Status().Update(context.TODO(), metricWebhook)
		if err != nil {
			reqLogger.Error(err, "failed to update MetricWebhook status")
		}
	}()

	// Fetch metric values and update MetricWebhook instance status
	metricStatuses, err := r.fetchCurrentMetrics(metricWebhook.Spec.Metrics, metricWebhook.Namespace, metricWebhook.Spec.Selector)
	if err != nil {
		reqLogger.Error(err, "failed to fetch current metric values",
			"Spec.Selector", metricWebhook.Spec.Selector,
		)
		return reconcile.Result{}, err
	}
	metricWebhook.Status.Metrics = metricStatuses

	// Check if any current metric values violate target thresholds and notify webhook
	alerts := r.findMetricAlerts(metricWebhook.Status.Metrics)
	if len(alerts) > 0 {
		webhookUrl, err := r.getWebhookUrl(metricWebhook.Namespace, metricWebhook.Spec.Webhook)
		if err != nil {
			reqLogger.Error(err, "failed to resolve webhook url")
			return reconcile.Result{}, err
		}
		reqLogger.Info("notifying webhook",
			"Spec.Webhook.Url(resolved)", webhookUrl,
			"alerts", alerts,
		)
		err = r.metricAlertClient.notify(webhookUrl, alerts)
		if err != nil {
			reqLogger.Info("failed to notify webhook",
				"Spec.Webhook.Url(resolved)", webhookUrl,
				"Error", err,
			)
			// Stay resilient, proceed normally
		}
	}

	return reconcile.Result{
		Requeue:      true,
		RequeueAfter: metricWebhook.Spec.ScrapeInterval.Duration,
	}, nil
}

func (r *ReconcileMetricWebhook) fetchCurrentMetrics(metricSpecs []metricsv1alpha1.MetricSpec, namespace string, labelSelector metav1.LabelSelector) ([]metricsv1alpha1.MetricStatus, error) {
	var metricStatuses []metricsv1alpha1.MetricStatus
	for _, metricSpec := range metricSpecs {
		metricStatus, err := r.fetchCurrentMetric(metricSpec, namespace, labelSelector)
		if err != nil {
			return metricStatuses, err
		}
		metricStatuses = append(metricStatuses, metricStatus)
	}
	return metricStatuses, nil
}

func (r *ReconcileMetricWebhook) fetchCurrentMetric(spec metricsv1alpha1.MetricSpec, namespace string, labelSelector metav1.LabelSelector) (metricsv1alpha1.MetricStatus, error) {
	switch spec.Type {
	case metricsv1alpha1.PodsMetricSourceType:
		currPodMetric, exceedsThreshold, err := r.fetchCurrentPodMetric(spec.Pods, namespace, labelSelector)
		if err != nil {
			return metricsv1alpha1.MetricStatus{}, err
		}
		return metricsv1alpha1.MetricStatus{
			Type:       metricsv1alpha1.PodsMetricSourceType,
			Alerting:   exceedsThreshold,
			Pods:       &currPodMetric,
			ScrapeTime: metav1.Now(),
		}, nil
	case metricsv1alpha1.ResourceMetricSourceType:
		currResourceMetric, exceedsThreshold, err := r.fetchCurrentResourceMetric(spec.Resource, namespace, labelSelector)
		if err != nil {
			return metricsv1alpha1.MetricStatus{}, err
		}
		return metricsv1alpha1.MetricStatus{
			Type:       metricsv1alpha1.ResourceMetricSourceType,
			Alerting:   exceedsThreshold,
			Resource:   &currResourceMetric,
			ScrapeTime: metav1.Now(),
		}, nil
	default:
		return metricsv1alpha1.MetricStatus{}, fmt.Errorf("invalid metric source type %s", spec.Type)
	}
}

func (r *ReconcileMetricWebhook) fetchCurrentPodMetric(spec *metricsv1alpha1.PodsMetricSource, namespace string, labelSelector metav1.LabelSelector) (metricsv1alpha1.PodsMetricStatus, bool, error) {
	averageValue, err := r.metricsClient.GetCurrentPodAverageValue(spec.Name, namespace, labelSelector, spec.TargetAverageValue)
	if err != nil {
		return metricsv1alpha1.PodsMetricStatus{}, false, err
	}

	exceedsThreshold := averageValue.Cmp(spec.TargetAverageValue) > 0
	return metricsv1alpha1.PodsMetricStatus{
		Name:                spec.Name,
		CurrentAverageValue: averageValue,
		TargetAverageValue:  spec.TargetAverageValue,
	}, exceedsThreshold, nil
}

func (r *ReconcileMetricWebhook) fetchCurrentResourceMetric(spec *metricsv1alpha1.ResourceMetricSource, namespace string, labelSelector metav1.LabelSelector) (metricsv1alpha1.ResourceMetricStatus, bool, error) {
	if spec.TargetAverageValue != nil {
		averageValue, err := r.metricsClient.GetCurrentResourceAverageValue(spec.Name, namespace, labelSelector, *spec.TargetAverageValue)
		if err != nil {
			return metricsv1alpha1.ResourceMetricStatus{}, false, err
		}

		exceedsThreshold := averageValue.Cmp(*spec.TargetAverageValue) > 0
		return metricsv1alpha1.ResourceMetricStatus{
			Name:                spec.Name,
			CurrentAverageValue: averageValue,
			TargetAverageValue:  spec.TargetAverageValue,
		}, exceedsThreshold, nil
	} else {
		if spec.TargetAverageUtilization == nil {
			return metricsv1alpha1.ResourceMetricStatus{}, false, fmt.Errorf("invalid resource metric source: neither a utilization target nor a value target set")
		}

		averageUtilization, averageValue, err := r.metricsClient.GetCurrentResourceAverageUtilization(spec.Name, namespace, labelSelector, *spec.TargetAverageUtilization)
		if err != nil {
			return metricsv1alpha1.ResourceMetricStatus{}, false, err
		}

		exceedsThreshold := averageUtilization > *spec.TargetAverageUtilization
		return metricsv1alpha1.ResourceMetricStatus{
			Name:                      spec.Name,
			CurrentAverageUtilization: &averageUtilization,
			TargetAverageUtilization:  spec.TargetAverageUtilization,
			CurrentAverageValue:       averageValue,
		}, exceedsThreshold, nil
	}
}

func (r *ReconcileMetricWebhook) findMetricAlerts(metrics []metricsv1alpha1.MetricStatus) metricsv1alpha1.MetricReport {
	if len(metrics) == 0 {
		return metricsv1alpha1.MetricReport{}
	}

	// Create alerts
	var alerts []metricsv1alpha1.MetricAlert
	for _, status := range metrics {
		if status.Alerting {
			continue
		}
		switch status.Type {
		case metricsv1alpha1.PodsMetricSourceType:
			alerts = append(alerts, metricsv1alpha1.MetricAlert{
				Type: status.Type,
				Name: status.Pods.Name,

				CurrentAverageValue: status.Pods.CurrentAverageValue,
				TargetAverageValue:  &status.Pods.TargetAverageValue,

				ScrapeTime: status.ScrapeTime.Time,
			})
		case metricsv1alpha1.ResourceMetricSourceType:
			alerts = append(alerts, metricsv1alpha1.MetricAlert{
				Type: status.Type,
				Name: status.Resource.Name.String(),

				CurrentAverageValue: status.Resource.CurrentAverageValue,
				TargetAverageValue:  status.Resource.TargetAverageValue,

				CurrentAverageUtilization: status.Resource.CurrentAverageUtilization,
				TargetAverageUtilization:  status.Resource.TargetAverageUtilization,

				ScrapeTime: status.ScrapeTime.Time,
			})
		}
	}

	return alerts
}

func (r *ReconcileMetricWebhook) getWebhookUrl(namespace string, webhookSpec metricsv1alpha1.Webhook) (string, error) {
	if specWebhookUrl := webhookSpec.Url; specWebhookUrl != "" {
		return specWebhookUrl, nil
	}

	specWebhookServiceKey := types.NamespacedName{
		Name:      webhookSpec.Service,
		Namespace: namespace,
	}

	webhookService := &v1.Service{}
	err := r.client.Get(context.TODO(), specWebhookServiceKey, webhookService)
	if err != nil {
		return "", fmt.Errorf("failed to fetch webhook service: %v", err)
	}

	portKnown := false
	for _, portSpec := range webhookService.Spec.Ports {
		if portSpec.Port == webhookSpec.Port {
			portKnown = true
		}
	}
	if !portKnown {
		return "", fmt.Errorf("webhook service doesnt expose port '%d' required (available = %v)",
			webhookSpec.Port, webhookService.Spec.Ports)
	}

	webhookUrl := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d",
		webhookService.Name, namespace, webhookSpec.Port)
	if webhookPath := webhookSpec.Path; webhookPath != "" {
		webhookUrl = webhookUrl + webhookPath
	}
	return webhookUrl, nil
}
