package metricwebhook

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"

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
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
)

const ReconcilerName = "metricwebhook-reconciler"

type MetricWebhookReconciler struct {
	client                   client.Client
	scheme                   *runtime.Scheme
	metricsClient            *MetricMeasurementClient
	metricNotificationClient *MetricNotificationClient
	eventRecorder            record.EventRecorder
	logger                   logr.Logger
}

func NewReconcileMetricWebhook(mgr manager.Manager) (reconcile.Reconciler, error) {
	restMapper := mgr.GetRESTMapper()
	clientSet, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return &MetricWebhookReconciler{}, err
	}

	apisGetter := custom_metrics.NewAvailableAPIsGetter(clientSet.Discovery())
	metricsClient := metrics.NewRESTMetricsClient(
		metricsclientv1beta1.NewForConfigOrDie(mgr.GetConfig()),
		custom_metrics.NewForConfig(mgr.GetConfig(), restMapper, apisGetter),
		external_metrics.NewForConfigOrDie(mgr.GetConfig()),
	)

	return &MetricWebhookReconciler{
		client:                   mgr.GetClient(),
		scheme:                   mgr.GetScheme(),
		metricsClient:            NewMetricValuesClient(metricsClient, clientSet),
		metricNotificationClient: NewDefaultMetricAlertClient(),
		eventRecorder:            mgr.GetEventRecorderFor(ControllerName),
		logger:                   logf.Log.WithName(ReconcilerName),
	}, nil
}

// Reconcile reads that state of the cluster for a MetricWebhook object and
// sends out metric notification based on the config defined in MetricWebhook.Spec
// and current metric measurements
func (r *MetricWebhookReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := r.logger.WithValues("Resource", request.NamespacedName)

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
	prevMetrics := metricWebhook.Status.DeepCopy().Metrics
	if currMetrics != nil {
		metricWebhook.Status.Metrics = currMetrics
	}

	// Diff and group metric to improved and unimproved metrics
	improvedMetrics, alertingMetrics := r.findImprovedAndAlertingMetrics(prevMetrics, currMetrics)

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
		webhookUrls, err := r.compileWebhookUrl(metricWebhook.Spec.Webhook, metricWebhook.Namespace, metricWebhook.Spec.Selector)
		if err != nil {
			r.eventRecorder.Event(metricWebhook, v1.EventTypeWarning, "FailedSendReport", err.Error())
			reqLogger.Error(err, "failed to resolve webhook url")
			return reconcile.Result{}, err
		}
		for _, webhookUrl := range webhookUrls {
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
			}
		}
	}

	return reconcile.Result{
		Requeue:      true,
		RequeueAfter: metricWebhook.Spec.ScrapeInterval.Duration,
	}, nil
}

func (r *MetricWebhookReconciler) fetchCurrentMetrics(metricSpecs []metricsv1alpha1.MetricSpec, namespace string, labelSelector metav1.LabelSelector) ([]metricsv1alpha1.MetricStatus, error) {
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

func (r *MetricWebhookReconciler) fetchCurrentMetric(spec metricsv1alpha1.MetricSpec, namespace string, labelSelector metav1.LabelSelector) (metricsv1alpha1.MetricStatus, error) {
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

func (r *MetricWebhookReconciler) fetchCurrentPodMetric(spec *metricsv1alpha1.PodsMetricSource, namespace string, labelSelector metav1.LabelSelector) (metricsv1alpha1.PodsMetricStatus, bool, error) {
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

func (r *MetricWebhookReconciler) fetchCurrentResourceMetric(spec *metricsv1alpha1.ResourceMetricSource, namespace string, labelSelector metav1.LabelSelector) (metricsv1alpha1.ResourceMetricStatus, bool, error) {
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

func (r *MetricWebhookReconciler) findImprovedAndAlertingMetrics(orig []metricsv1alpha1.MetricStatus, upd []metricsv1alpha1.MetricStatus) (improvedMetrics []metricsv1alpha1.MetricStatus, alertingMetrics []metricsv1alpha1.MetricStatus) {
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
			metricNameToUpdated[metric.Pods.Name] = metric
		case metricsv1alpha1.ResourceMetricSourceType:
			metricNameToUpdated[metric.Resource.Name.String()] = metric
		}
	}

	for name, origMetric := range metricNameToOrigin {
		updMetric := metricNameToUpdated[name]

		if origMetric.Alerting && !updMetric.Alerting {
			improvedMetrics = append(improvedMetrics, updMetric)
		} else if updMetric.Alerting {
			alertingMetrics = append(alertingMetrics, updMetric)
		}
	}

	return
}

func (r *MetricWebhookReconciler) createMetricReport(alertingMetrics, improvedMetrics []metricsv1alpha1.MetricStatus) metricsv1alpha1.MetricReport {
	var report metricsv1alpha1.MetricReport
	for _, metric := range alertingMetrics {
		if !metric.Alerting {
			continue
		}
		report = append(report, createMetricNotification(metricsv1alpha1.Alert, metric))
	}
	for _, metric := range improvedMetrics {
		report = append(report, createMetricNotification(metricsv1alpha1.Cooldown, metric))
	}

	return report
}

func (r *MetricWebhookReconciler) compileWebhookUrl(spec metricsv1alpha1.Webhook, namespace string, labelSelector metav1.LabelSelector) ([]string, error) {
	switch {
	case spec.Url != "":
		// Use explicit url if set
		return []string{spec.Url}, nil
	case spec.Service != "":
		// Lookup for a service to assert the port from spec if set
		// and compile url to the service
		webhookService := &v1.Service{}
		err := r.client.Get(context.TODO(), types.NamespacedName{
			Name:      spec.Service,
			Namespace: namespace,
		}, webhookService)
		if err != nil {
			return []string{}, fmt.Errorf("failed to fetch webhook service: %v", err)
		}

		portKnown := false
		for _, portSpec := range webhookService.Spec.Ports {
			if portSpec.Port == spec.Port {
				portKnown = true
			}
		}
		if !portKnown {
			return []string{}, fmt.Errorf("webhook service doesnt expose port '%d' required (available = %v)",
				spec.Port, webhookService.Spec.Ports)
		}

		webhookUrl := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d",
			webhookService.Name, namespace, spec.Port)
		if webhookPath := spec.Path; webhookPath != "" {
			webhookUrl = webhookUrl + webhookPath
		}
		return []string{webhookUrl}, nil
	default:
		// Match all pods by label
		podSelector, err := metav1.LabelSelectorAsSelector(&labelSelector)
		if err != nil {
			return []string{}, err
		}

		var pods v1.PodList
		err = r.client.List(context.TODO(), &pods, &client.ListOptions{
			LabelSelector: podSelector,
			Namespace:     namespace,
		})
		if err != nil {
			return []string{}, err
		}

		var webhookUrls []string
		for _, pod := range pods.Items {
			webhookUrl := fmt.Sprintf("http://%s:%d",
				pod.Status.PodIP, spec.Port)
			if webhookPath := spec.Path; webhookPath != "" {
				webhookUrl = webhookUrl + webhookPath
			}
			webhookUrls = append(webhookUrls, webhookUrl)
		}

		return webhookUrls, nil
	}
}

func (r *MetricWebhookReconciler) postMetricReportEvents(o runtime.Object, report metricsv1alpha1.MetricReport) {
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
		r.eventRecorder.Event(o, v1.EventTypeNormal, "NewAlerts", alertingMetricsStatus)
	}
	if len(cooldownMetric) > 0 {
		cooldownMetricStatus := strings.Join(cooldownMetric, ", ")
		r.eventRecorder.Event(o, v1.EventTypeNormal, "NewCooldowns", cooldownMetricStatus)
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
			Type: typ,

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
