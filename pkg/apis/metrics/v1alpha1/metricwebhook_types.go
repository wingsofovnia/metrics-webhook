package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MetricWebhookSpec defines the desired state of MetricWebhook
// +k8s:openapi-gen=true
type MetricWebhookSpec struct {
	// targetRef points to the target resource used to fetch pods
	// for which metrics should be collected
	TargetRef TargetRef `json:"targetRef"`
	// webhook points to the web endpoint that going to get metric alerts
	Webhook Webhook `json:"webhook"`
	// metrics contains the specifications for metrics thresholds
	// used to trigger webhook
	// +listType=set
	Metrics []MetricSpec `json:"metrics"`
	// scrapeInterval defines how frequently to scrape metrics
	ScrapeInterval metav1.Duration `json:"scrapeInterval"`
}

// TargetRef points to deployment or pod to watch metrics for
// +k8s:openapi-gen=true
type TargetRef struct {
	// Kind of the referent; More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds"
	Kind string `json:"kind"`
	// Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names
	Name string `json:"name"`
}

// Webhook describes the web endpoint that the operator calls on metrics reaching their thresholds
// +k8s:openapi-gen=true
type Webhook struct {
	// Kind of the referent service
	Service string `json:"service"`
	// Service port the webserver serves on
	Port string `json:"port"`
	// URL path to the webhook
	Path string `json:"path"`
}

// +k8s:openapi-gen=true
// MetricSourceType indicates the type of metric.
type MetricSourceType string

const (
	// PodsMetricSourceType is a metric describing each pod in the current
	// target (for example, transactions-processed-per-second).  The values
	// will be averaged together before being compared to the target value.
	PodsMetricSourceType MetricSourceType = "Pods"
	// ResourceMetricSourceType is a resource metric known to Kubernetes, as
	// specified in requests and limits, describing each pod in the current
	// target (e.g. CPU or memory).
	ResourceMetricSourceType MetricSourceType = "Resource"
)

// MetricSpec specifies metrics thresholds before webhook gets called
// +k8s:openapi-gen=true
type MetricSpec struct {
	// type is the type of metric source.  It should be "Pods" or "Resource",
	// each mapping to a matching field in the object.
	Type MetricSourceType `json:"type"`

	// pods refers to a metric describing each pod in the current scale target
	// (for example, transactions-processed-per-second). The values will be
	// averaged together before being compared to the target value.
	// +optional
	Pods *PodsMetricSource `json:"pods,omitempty"`
	// resource refers to a resource metric (such as those specified in
	// requests and limits) known to Kubernetes describing each pod in the
	// current target (e.g. CPU or memory).
	// +optional
	Resource *ResourceMetricSource `json:"resource,omitempty"`
}

// PodsMetricSource indicates when to call webhook on a metric describing each pod in
// the current target (for example, transactions-processed-per-second).
// The values will be averaged together before being compared to the target
// value.
// +k8s:openapi-gen=true
type PodsMetricSource struct {
	// metricName is the name of the metric in question
	MetricName string `json:"metricName"`
	// targetAverageValue is the target value of the average of the
	// metric across all relevant pods (as a quantity)
	TargetAverageValue resource.Quantity `json:"targetAverageValue"`
}

// ResourceMetricSource indicates when to call webhook on a resource metric known to
// Kubernetes, as specified in requests and limits, describing each pod in the
// current target (e.g. CPU or memory).
// The values will be averaged together before being compared to the target
// +k8s:openapi-gen=true
type ResourceMetricSource struct {
	// name is the name of the resource in question.
	Name v1.ResourceName `json:"name"`
	// targetAverageUtilization is the target value of the average of the
	// resource metric across all relevant pods, represented as a percentage of
	// the requested value of the resource for the pods.
	// +optional
	TargetAverageUtilization *int32 `json:"targetAverageUtilization,omitempty"`
	// targetAverageValue is the target value of the average of the
	// resource metric across all relevant pods, as a raw value (instead of as
	// a percentage of the request), similar to the "pods" metric source type.
	// +optional
	TargetAverageValue *resource.Quantity `json:"targetAverageValue,omitempty"`
}

// MetricWebhookStatus defines the observed state of MetricWebhook
// +k8s:openapi-gen=true
type MetricWebhookStatus struct {
	// lastScrapeTime is the last time the MetricWebhook scraped metrics
	// +optional
	LastScrapeTime *metav1.Time `json:"lastScrapeTime,omitempty"`
	// currentMetrics is the last read state of the metrics used by this MetricWebhook.
	// +listType=set
	// +optional
	CurrentMetrics []MetricStatus `json:"currentMetrics"`
}

// MetricStatus describes the last-read state of a single metric.
// +k8s:openapi-gen=true
type MetricStatus struct {
	// type is the type of metric source.  It should be "Pods" or "Resource",
	// each mapping to a matching field in the object.
	Type MetricSourceType `json:"type"`
	// pods refers to a metric describing each pod in the current target
	// (for example, transactions-processed-per-second). The values will be
	// averaged together before being compared to the target value.
	// +optional
	Pods *PodsMetricStatus `json:"pods,omitempty"`
	// resource refers to a resource metric (such as those specified in
	// requests and limits) known to Kubernetes describing each pod in the
	// current target (e.g. CPU or memory). Such metrics are built in to
	// Kubernetes, and have special scaling options on top of those available
	// to normal per-pod metrics using the "pods" source.
	// +optional
	Resource *ResourceMetricStatus `json:"resource,omitempty"`
}

// PodsMetricStatus indicates the current value of a metric describing each pod in
// the current scale target (for example, transactions-processed-per-second).
// +k8s:openapi-gen=true
type PodsMetricStatus struct {
	// metricName is the name of the metric in question
	MetricName string `json:"metricName"`
	// currentAverageValue is the current value of the average of the
	// metric across all relevant pods (as a quantity)
	CurrentAverageValue resource.Quantity `json:"currentAverageValue"`
}

// ResourceMetricStatus indicates the current value of a resource metric known to
// Kubernetes, as specified in requests and limits, describing each pod in the
// current target (e.g. CPU or memory). Such metrics are built in to
// Kubernetes, and have special scaling options on top of those available to
// normal per-pod metrics using the "pods" source.
// +k8s:openapi-gen=true
type ResourceMetricStatus struct {
	// name is the name of the resource in question.
	Name v1.ResourceName `json:"name"`
	// currentAverageUtilization is the current value of the average of the
	// resource metric across all relevant pods, represented as a percentage of
	// the requested value of the resource for the pods.  It will only be
	// present if `targetAverageValue` was set in the corresponding metric
	// specification.
	// +optional
	CurrentAverageUtilization *int32 `json:"currentAverageUtilization,omitempty"`
	// currentAverageValue is the current value of the average of the
	// resource metric across all relevant pods, as a raw value (instead of as
	// a percentage of the request), similar to the "pods" metric source type.
	// It will always be set, regardless of the corresponding metric specification.
	CurrentAverageValue resource.Quantity `json:"currentAverageValue"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MetricWebhook is the Schema for the metricwebhooks API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=metricwebhooks,scope=Namespaced
type MetricWebhook struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MetricWebhookSpec   `json:"spec,omitempty"`
	Status MetricWebhookStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MetricWebhookList contains a list of MetricWebhook
type MetricWebhookList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MetricWebhook `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MetricWebhook{}, &MetricWebhookList{})
}
