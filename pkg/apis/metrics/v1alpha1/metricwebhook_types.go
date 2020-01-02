package v1alpha1

import (
	"fmt"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MetricWebhookSpec defines the desired state of MetricWebhook
// +k8s:openapi-gen=true
type MetricWebhookSpec struct {
	// Selector is a label selector for pods for which metrics should be collected
	Selector metav1.LabelSelector `json:"selector"`
	// webhook points to the web endpoint that going to get metric alerts
	Webhook Webhook `json:"webhook"`
	// metrics contains the specifications for metrics thresholds
	// used to trigger webhook
	// +listType=set
	Metrics []MetricSpec `json:"metrics"`
	// scrapeInterval defines how frequently to scrape metrics
	ScrapeInterval metav1.Duration `json:"scrapeInterval"`
	// Determines whether a metric alert sent one more time after values go
	// under thresholds so that the client can track its adjustments improvements
	CooldownAlert bool `json:"cooldownAlert"`
}

// Webhook describes the web endpoint that the operator calls on metrics reaching their thresholds
// +k8s:openapi-gen=true
type Webhook struct {
	// Explicit URL to hit, instead of matching service.
	// If neither Url nor Service are specified, the webhook
	// triggers all matching pods
	// +optional
	Url string `json:"url"`
	// Referent service
	// If neither Url nor Service are specified, the webhook
	// triggers all matching pods
	// +optional
	Service string `json:"service"`
	// Service port the webserver serves on
	// +optional
	Port int32 `json:"port"`
	// URL path to the webhook
	// +optional
	Path string `json:"path"`
}

// +k8s:openapi-gen=true
// +kubebuilder:validation:Enum=Pods;Resource
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
	// pods refers to a metric describing each pod matching the selector
	// (for example, transactions-processed-per-second). The values will be
	// averaged together before being compared to the target value.
	// +optional
	Pods *PodsMetricSource `json:"pods,omitempty"`
	// resource refers to a resource metric (such as those specified in
	// requests and limits) known to Kubernetes describing each pod matching
	// the selector (e.g. CPU or memory).
	// +optional
	Resource *ResourceMetricSource `json:"resource,omitempty"`
}

// PodsMetricSource indicates when to call webhook on a metric describing each pod
// matching the selector (for example, transactions-processed-per-second).
// The values will be averaged together before being compared to the target value.
// +k8s:openapi-gen=true
type PodsMetricSource struct {
	// name is the name of the metric in question
	Name string `json:"name"`
	// targetAverageValue is the target value of the average of the
	// metric across all relevant pods (as a quantity)
	TargetAverageValue resource.Quantity `json:"targetAverageValue"`
}

// ResourceMetricSource indicates when to call webhook on a resource metric known to
// Kubernetes, as specified in requests and limits, describing each pod
// matching the selector (e.g. CPU or memory).
// The values will be averaged together before being compared to the target value.
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
	// metrics is the last read state of the metrics used by this MetricWebhook.
	// +listType=set
	// +optional
	Metrics []MetricStatus `json:"metrics"`
}

// MetricStatus describes the last-read state of a single metric.
// +k8s:openapi-gen=true
type MetricStatus struct {
	// type is the type of metric source.  It should be "Pods" or "Resource",
	// each mapping to a matching field in the object.
	Type MetricSourceType `json:"type"`
	// alerting flags the metrics those values exceed defined thresholds
	Alerting bool `json:"alerting"`
	// pods refers to a metric describing each pod matching the selector
	// (for example, transactions-processed-per-second). The values will be
	// averaged together before being compared to the target value.
	// +optional
	Pods *PodsMetricStatus `json:"pods,omitempty"`
	// resource refers to a resource metric (such as those specified in
	// requests and limits) known to Kubernetes describing each pod
	// matching the selector (e.g. CPU or memory). Such metrics are built in to
	// Kubernetes, and have special scaling options on top of those available
	// to normal per-pod metrics using the "pods" source.
	// +optional
	Resource *ResourceMetricStatus `json:"resource,omitempty"`
	// scrapeTime is the last time the MetricWebhook scraped metrics
	ScrapeTime metav1.Time `json:"scrapeTime"`
}

// PodsMetricStatus indicates the current value of a metric describing each pod
// matching the selector (for example, transactions-processed-per-second).
// +k8s:openapi-gen=true
type PodsMetricStatus struct {
	// name is the name of the metric in question
	Name string `json:"name"`
	// currentAverageValue is the current value of the average of the
	// metric across all relevant pods (as a quantity)
	CurrentAverageValue resource.Quantity `json:"currentAverageValue"`
	// targetAverageValue is the target value of the average of the
	// metric across all relevant pods (as a quantity) defined for this metric in specs
	TargetAverageValue resource.Quantity `json:"targetAverageValue"`
}

// ResourceMetricStatus indicates the current value of a resource metric known to
// Kubernetes, as specified in requests and limits, describing each pod
// matching the selector (e.g. CPU or memory). Such metrics are built in to
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
	// targetAverageUtilization is the target value of the average of the
	// resource metric across all relevant pods defined for this metric in specs
	// +optional
	TargetAverageUtilization *int32 `json:"targetAverageUtilization,omitempty"`
	// currentAverageValue is the current value of the average of the
	// resource metric across all relevant pods, as a raw value (instead of as
	// a percentage of the request), similar to the "pods" metric source type.
	// It will always be set, regardless of the corresponding metric specification.
	CurrentAverageValue resource.Quantity `json:"currentAverageValue"`
	// targetAverageValue is the target value of the average of the
	// metric across all relevant pods (as a quantity) defined for this metric in specs
	TargetAverageValue *resource.Quantity `json:"targetAverageValue,omitempty"`
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

// +k8s:deepcopy-gen=false
// +k8s:openapi-gen=false
// +kubebuilder:skipversion
type MetricNotificationType string

const (
	// Alert metric notification informs about a metric those
	// values exceeded the ones set by its thresholds.
	Alert MetricNotificationType = "Alert"
	// Cooldown metric notification informs about a metric that
	// has improved to the extend of meeting thresholds set. This
	// notification may be used by the target application in order
	// to correlate internal adjustments with metric values improvements.
	Cooldown MetricNotificationType = "Cooldown"
)

// +k8s:deepcopy-gen=false
// +k8s:openapi-gen=false
// +kubebuilder:skipversion
type MetricNotification struct {
	Type                      MetricNotificationType `json:"type"`
	MetricType                MetricSourceType       `json:"metricType"`
	Name                      string                 `json:"name"`
	CurrentAverageValue       resource.Quantity      `json:"currentAverageValue,omitempty"`
	TargetAverageValue        *resource.Quantity     `json:"targetAverageValue,omitempty"`
	CurrentAverageUtilization *int32                 `json:"currentAverageUtilization,omitempty"`
	TargetAverageUtilization  *int32                 `json:"targetAverageUtilization,omitempty"`
	ScrapeTime                time.Time              `json:"scrapeTime"`
}

func (n *MetricNotification) String() string {
	var tokens []string

	tokens = append(tokens, fmt.Sprintf("name = %s", n.Name))
	if n.TargetAverageUtilization != nil {
		tokens = append(tokens, fmt.Sprintf("avg utilization = %d%%/%d%%", *n.CurrentAverageUtilization, *n.TargetAverageUtilization))
	} else {
		tokens = append(tokens, fmt.Sprintf("avg value = %s/%s", n.CurrentAverageValue.String(), n.TargetAverageValue.String()))
	}

	return string(n.Type) + "[" + strings.Join(tokens, ", ") + "]"
}

// +k8s:deepcopy-gen=false
// +k8s:openapi-gen=false
// +kubebuilder:skipversion
type MetricReport []MetricNotification

func init() {
	SchemeBuilder.Register(&MetricWebhook{}, &MetricWebhookList{})
}
