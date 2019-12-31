package metricwebhook

import (
	"context"
	"fmt"
	"strings"

	metricsv1alpha1 "github.com/wingsofovnia/metrics-webhook/pkg/apis/metrics/v1alpha1"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
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

const controllerName = "metricwebhook-controller"

var log = logf.Log.WithName(controllerName)

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
		client:                   mgr.GetClient(),
		scheme:                   mgr.GetScheme(),
		metricsClient:            metricValuesClient,
		metricNotificationClient: NewDefaultMetricAlertClient(),
		eventRecorder:            mgr.GetEventRecorderFor(controllerName),
	}, nil
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r})
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
	client                   client.Client
	scheme                   *runtime.Scheme
	metricsClient            *MetricValuesClient
	metricNotificationClient *MetricNotificationClient
	eventRecorder            record.EventRecorder
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
			r.eventRecorder.Event(metricWebhook, v1.EventTypeWarning, "FailedSaveStatus", err.Error())
			reqLogger.Error(err, "failed to update MetricWebhook status")
		}
	}()

	// Fetch metric values and update MetricWebhook instance status
	currMetrics, err := r.fetchCurrentMetrics(metricWebhook.Spec.Metrics, metricWebhook.Namespace, metricWebhook.Spec.Selector)
	if err != nil {
		r.eventRecorder.Event(metricWebhook, v1.EventTypeWarning, "FailedFetchMetrics", err.Error())
		reqLogger.Error(err, "failed to fetch current metric values",
			"Spec.Selector", metricWebhook.Spec.Selector,
		)
		return reconcile.Result{}, err
	}
	prevMetrics := metricWebhook.Status.Metrics
	metricWebhook.Status.Metrics = currMetrics

	// Diff and group metric to improved and unimproved metrics
	_, improvedMetrics, alertingMetrics := r.diffMetrics(prevMetrics, currMetrics)

	// Compile metric report to (not/)include cooldown notifications
	var metricReport metricsv1alpha1.MetricReport
	if metricWebhook.Spec.CooldownAlert {
		metricReport = r.createMetricReport(alertingMetrics, improvedMetrics)
	} else {
		metricReport = r.createMetricReport(alertingMetrics, []metricsv1alpha1.MetricStatus{})
	}

	// Post event(s) describing the metric notifications to be sent
	r.postMetricReportEvents(metricWebhook, metricReport)

	// Send out metric notifications
	if len(metricReport) > 0 {
		webhookUrl, err := r.compileWebhookUrl(metricWebhook.Namespace, metricWebhook.Spec.Webhook)
		if err != nil {
			r.eventRecorder.Event(metricWebhook, v1.EventTypeWarning, "FailedSendReport", err.Error())
			reqLogger.Error(err, "failed to resolve webhook url")
			return reconcile.Result{}, err
		}
		reqLogger.Info("notifying webhook",
			"Spec.Webhook.Url(resolved)", webhookUrl,
			"metricReport", metricReport,
		)
		err = r.metricNotificationClient.notify(webhookUrl, metricReport)
		if err != nil {
			r.eventRecorder.Event(metricWebhook, v1.EventTypeWarning, "FailedSendReport", err.Error())
			reqLogger.Info("failed to notify webhook",
				"Spec.Webhook.Url(resolved)", webhookUrl,
				"Error", err,
			)
			// Stay resilient, proceed normally
		} else {
			r.eventRecorder.Eventf(metricWebhook, v1.EventTypeNormal, "SucceededReport", "The webhook (url = %s) was successfully notified with a report of all %d notifications", webhookUrl, len(metricReport))
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

func (r *ReconcileMetricWebhook) diffMetrics(orig []metricsv1alpha1.MetricStatus, upd []metricsv1alpha1.MetricStatus) (unimprovedMetrics []metricsv1alpha1.MetricStatus, improvedMetrics []metricsv1alpha1.MetricStatus, alertingMetrics []metricsv1alpha1.MetricStatus) {
	// Group Orig(in) metrics by name
	metricNameToOrigin := make(map[string]metricsv1alpha1.MetricStatus)
	for _, metric := range orig {
		switch metric.Type {
		case metricsv1alpha1.PodsMetricSourceType:
			metricNameToOrigin[metric.Pods.Name] = metric
		case metricsv1alpha1.ResourceMetricSourceType:
			metricNameToOrigin[metric.Resource.Name.String()] = metric
		}
	}

	// Group Upd(ated) metrics by name
	metricNameToUpdated := make(map[string]metricsv1alpha1.MetricStatus)
	for _, metric := range upd {
		switch metric.Type {
		case metricsv1alpha1.PodsMetricSourceType:
			metricNameToOrigin[metric.Pods.Name] = metric
		case metricsv1alpha1.ResourceMetricSourceType:
			metricNameToOrigin[metric.Resource.Name.String()] = metric
		}
	}

	for name, origMetric := range metricNameToOrigin {
		updMetric := metricNameToUpdated[name]

		if origMetric.Alerting && !updMetric.Alerting {
			improvedMetrics = append(improvedMetrics, updMetric)
		} else if !origMetric.Alerting && updMetric.Alerting {
			alertingMetrics = append(alertingMetrics, updMetric)
		} else {
			unimprovedMetrics = append(unimprovedMetrics, updMetric)
		}
	}

	return
}

func (r *ReconcileMetricWebhook) createMetricReport(alertingMetrics, improvedMetrics []metricsv1alpha1.MetricStatus) metricsv1alpha1.MetricReport {
	var report metricsv1alpha1.MetricReport
	for _, metric := range alertingMetrics {
		if metric.Alerting {
			continue
		}
		report = append(report, createMetricNotification(metricsv1alpha1.Alert, metric))
	}
	for _, metric := range improvedMetrics {
		report = append(report, createMetricNotification(metricsv1alpha1.Alert, metric))
	}

	return report
}

func (r *ReconcileMetricWebhook) compileWebhookUrl(namespace string, webhookSpec metricsv1alpha1.Webhook) (string, error) {
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

func (r *ReconcileMetricWebhook) postMetricReportEvents(o runtime.Object, report metricsv1alpha1.MetricReport) {
	var alertingMetrics []string
	var cooldownMetric []string
	for _, notification := range report {
		switch notification.Type {
		case metricsv1alpha1.Alert:
			alertingMetrics = append(alertingMetrics, notification.String())
		case metricsv1alpha1.Cooldown:
			cooldownMetric = append(cooldownMetric, notification.String())
		}
	}

	if len(alertingMetrics) > 0 {
		alertingMetricsStatus := strings.Join(alertingMetrics, ", ")
		r.eventRecorder.Eventf(o, v1.EventTypeNormal, "NewAlerts", "New alerting notifications: %s", alertingMetricsStatus)
	}
	if len(cooldownMetric) > 0 {
		cooldownMetricStatus := strings.Join(cooldownMetric, ", ")
		r.eventRecorder.Eventf(o, v1.EventTypeNormal, "NewCooldowns", "New cooldown notifications: %s", cooldownMetricStatus)
	}
}

func createMetricNotification(typ metricsv1alpha1.MetricNotificationType, metric metricsv1alpha1.MetricStatus) metricsv1alpha1.MetricNotification {
	switch metric.Type {
	case metricsv1alpha1.PodsMetricSourceType:
		return metricsv1alpha1.MetricNotification{
			Type: typ,

			MetricType: metric.Type,
			Name:       metric.Pods.Name,

			CurrentAverageValue: metric.Pods.CurrentAverageValue,
			TargetAverageValue:  &metric.Pods.TargetAverageValue,

			ScrapeTime: metric.ScrapeTime.Time,
		}
	case metricsv1alpha1.ResourceMetricSourceType:
		return metricsv1alpha1.MetricNotification{
			Type: metricsv1alpha1.Alert,

			MetricType: metric.Type,
			Name:       metric.Resource.Name.String(),

			CurrentAverageValue: metric.Resource.CurrentAverageValue,
			TargetAverageValue:  metric.Resource.TargetAverageValue,

			CurrentAverageUtilization: metric.Resource.CurrentAverageUtilization,
			TargetAverageUtilization:  metric.Resource.TargetAverageUtilization,

			ScrapeTime: metric.ScrapeTime.Time,
		}
	}
	return metricsv1alpha1.MetricNotification{}
}
