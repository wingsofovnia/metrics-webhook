package metricwebhook

import (
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8s "k8s.io/client-go/kubernetes"
	metricsclient "k8s.io/kubernetes/pkg/controller/podautoscaler/metrics"
)

type MetricMeasurementClient struct {
	metricsClient metricsclient.MetricsClient
	k8sClient     *k8s.Clientset
}

func NewMetricValuesClient(metricsClient metricsclient.MetricsClient, k8sClient *k8s.Clientset) *MetricMeasurementClient {
	return &MetricMeasurementClient{metricsClient: metricsClient, k8sClient: k8sClient}
}

func (f *MetricMeasurementClient) GetCurrentPodAverageValue(name string, namespace string, labelSelector metav1.LabelSelector, targetAverageValue resource.Quantity) (averageValue resource.Quantity, time time.Time, err error) {
	podSelector, err := metav1.LabelSelectorAsSelector(&labelSelector)
	if err != nil {
		return resource.Quantity{}, time, err
	}

	metrics, timestamp, err := f.metricsClient.GetRawMetric(name, namespace, podSelector, labels.Nothing())
	if err != nil {
		return resource.Quantity{}, time, err
	}
	_, currentUtilization := metricsclient.GetMetricUtilizationRatio(metrics, targetAverageValue.MilliValue())
	return *resource.NewMilliQuantity(currentUtilization, resource.DecimalSI), timestamp, nil
}

func (f *MetricMeasurementClient) GetCurrentResourceAverageValue(name v1.ResourceName, namespace string, labelSelector metav1.LabelSelector, targetAverageValue resource.Quantity) (averageValue resource.Quantity, time time.Time, err error) {
	podSelector, err := metav1.LabelSelectorAsSelector(&labelSelector)
	if err != nil {
		return resource.Quantity{}, time, err
	}

	metrics, timestamp, err := f.metricsClient.GetResourceMetric(name, namespace, podSelector)
	if err != nil {
		return resource.Quantity{}, time, err
	}
	_, currentUtilization := metricsclient.GetMetricUtilizationRatio(metrics, targetAverageValue.MilliValue())
	return *resource.NewMilliQuantity(currentUtilization, resource.DecimalSI), timestamp, nil
}

func (f *MetricMeasurementClient) GetCurrentResourceAverageUtilization(name v1.ResourceName, namespace string, labelSelector metav1.LabelSelector, targetAverageUtilization int32) (averageUtilization int32, averageValue resource.Quantity, time time.Time, err error) {
	podSelector, err := metav1.LabelSelectorAsSelector(&labelSelector)
	if err != nil {
		return 0, resource.Quantity{}, time, err
	}

	// Get all matching pods
	podLabels, err := metav1.LabelSelectorAsMap(&labelSelector)
	if err != nil {
		return 0, resource.Quantity{}, time, err
	}
	allPods, err := f.k8sClient.CoreV1().Pods(namespace).List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(podLabels).String(),
	})
	if err != nil {
		return 0, resource.Quantity{}, time, err
	}

	// Get pod metrics
	metrics, timestamp, err := f.metricsClient.GetResourceMetric(name, namespace, podSelector)
	if err != nil {
		return 0, resource.Quantity{}, time, err
	}

	// Filter out pods that are either not running or not present in metrics
	var eligiblePods []v1.Pod
	for _, pod := range allPods.Items {
		if pod.Status.Phase != v1.PodRunning {
			continue
		}
		if _, found := metrics[pod.Name]; !found {
			continue
		}
		eligiblePods = append(eligiblePods, pod)
	}

	requests, err := calculatePodRequests(eligiblePods, name)
	if err != nil {
		return 0, resource.Quantity{}, time, err
	}

	_, utilization, rawUtilization, err := metricsclient.GetResourceUtilizationRatio(metrics, requests, targetAverageUtilization)
	return utilization, *resource.NewMilliQuantity(rawUtilization, resource.DecimalSI), timestamp, err
}

// Source: https://github.com/kubernetes/kubernetes/blob/928817a26a84d9e3076d110ea30ba994912aa477/pkg/controller/podautoscaler/replica_calculator.go#L405
func calculatePodRequests(pods []v1.Pod, resource v1.ResourceName) (map[string]int64, error) {
	requests := make(map[string]int64, len(pods))
	for _, pod := range pods {
		podSum := int64(0)
		for _, container := range pod.Spec.Containers {
			if containerRequest, ok := container.Resources.Requests[resource]; ok {
				podSum += containerRequest.MilliValue()
			} else {
				return nil, fmt.Errorf("missing request for %s", resource)
			}
		}
		requests[pod.Name] = podSum
	}
	return requests, nil
}
