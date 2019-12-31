// +build !ignore_autogenerated

// This file was autogenerated by openapi-gen. Do not edit it manually!

package v1alpha1

import (
	spec "github.com/go-openapi/spec"
	common "k8s.io/kube-openapi/pkg/common"
)

func GetOpenAPIDefinitions(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	return map[string]common.OpenAPIDefinition{
		"./pkg/apis/metrics/v1alpha1.MetricSpec":           schema_pkg_apis_metrics_v1alpha1_MetricSpec(ref),
		"./pkg/apis/metrics/v1alpha1.MetricStatus":         schema_pkg_apis_metrics_v1alpha1_MetricStatus(ref),
		"./pkg/apis/metrics/v1alpha1.MetricWebhook":        schema_pkg_apis_metrics_v1alpha1_MetricWebhook(ref),
		"./pkg/apis/metrics/v1alpha1.MetricWebhookSpec":    schema_pkg_apis_metrics_v1alpha1_MetricWebhookSpec(ref),
		"./pkg/apis/metrics/v1alpha1.MetricWebhookStatus":  schema_pkg_apis_metrics_v1alpha1_MetricWebhookStatus(ref),
		"./pkg/apis/metrics/v1alpha1.PodsMetricSource":     schema_pkg_apis_metrics_v1alpha1_PodsMetricSource(ref),
		"./pkg/apis/metrics/v1alpha1.PodsMetricStatus":     schema_pkg_apis_metrics_v1alpha1_PodsMetricStatus(ref),
		"./pkg/apis/metrics/v1alpha1.ResourceMetricSource": schema_pkg_apis_metrics_v1alpha1_ResourceMetricSource(ref),
		"./pkg/apis/metrics/v1alpha1.ResourceMetricStatus": schema_pkg_apis_metrics_v1alpha1_ResourceMetricStatus(ref),
		"./pkg/apis/metrics/v1alpha1.Webhook":              schema_pkg_apis_metrics_v1alpha1_Webhook(ref),
	}
}

func schema_pkg_apis_metrics_v1alpha1_MetricSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "MetricSpec specifies metrics thresholds before webhook gets called",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"type": {
						SchemaProps: spec.SchemaProps{
							Description: "type is the type of metric source.  It should be \"Pods\" or \"Resource\", each mapping to a matching field in the object.",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"pods": {
						SchemaProps: spec.SchemaProps{
							Description: "pods refers to a metric describing each pod matching the selector (for example, transactions-processed-per-second). The values will be averaged together before being compared to the target value.",
							Ref:         ref("./pkg/apis/metrics/v1alpha1.PodsMetricSource"),
						},
					},
					"resource": {
						SchemaProps: spec.SchemaProps{
							Description: "resource refers to a resource metric (such as those specified in requests and limits) known to Kubernetes describing each pod matching the selector (e.g. CPU or memory).",
							Ref:         ref("./pkg/apis/metrics/v1alpha1.ResourceMetricSource"),
						},
					},
				},
				Required: []string{"type"},
			},
		},
		Dependencies: []string{
			"./pkg/apis/metrics/v1alpha1.PodsMetricSource", "./pkg/apis/metrics/v1alpha1.ResourceMetricSource"},
	}
}

func schema_pkg_apis_metrics_v1alpha1_MetricStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "MetricStatus describes the last-read state of a single metric.",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"type": {
						SchemaProps: spec.SchemaProps{
							Description: "type is the type of metric source.  It should be \"Pods\" or \"Resource\", each mapping to a matching field in the object.",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"pods": {
						SchemaProps: spec.SchemaProps{
							Description: "pods refers to a metric describing each pod matching the selector (for example, transactions-processed-per-second). The values will be averaged together before being compared to the target value.",
							Ref:         ref("./pkg/apis/metrics/v1alpha1.PodsMetricStatus"),
						},
					},
					"resource": {
						SchemaProps: spec.SchemaProps{
							Description: "resource refers to a resource metric (such as those specified in requests and limits) known to Kubernetes describing each pod matching the selector (e.g. CPU or memory). Such metrics are built in to Kubernetes, and have special scaling options on top of those available to normal per-pod metrics using the \"pods\" source.",
							Ref:         ref("./pkg/apis/metrics/v1alpha1.ResourceMetricStatus"),
						},
					},
					"scrapeTime": {
						SchemaProps: spec.SchemaProps{
							Description: "scrapeTime is the last time the MetricWebhook scraped metrics",
							Ref:         ref("k8s.io/apimachinery/pkg/apis/meta/v1.Time"),
						},
					},
				},
				Required: []string{"type", "scrapeTime"},
			},
		},
		Dependencies: []string{
			"./pkg/apis/metrics/v1alpha1.PodsMetricStatus", "./pkg/apis/metrics/v1alpha1.ResourceMetricStatus", "k8s.io/apimachinery/pkg/apis/meta/v1.Time"},
	}
}

func schema_pkg_apis_metrics_v1alpha1_MetricWebhook(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "MetricWebhook is the Schema for the metricwebhooks API",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"),
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("./pkg/apis/metrics/v1alpha1.MetricWebhookSpec"),
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("./pkg/apis/metrics/v1alpha1.MetricWebhookStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"./pkg/apis/metrics/v1alpha1.MetricWebhookSpec", "./pkg/apis/metrics/v1alpha1.MetricWebhookStatus", "k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"},
	}
}

func schema_pkg_apis_metrics_v1alpha1_MetricWebhookSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "MetricWebhookSpec defines the desired state of MetricWebhook",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"selector": {
						SchemaProps: spec.SchemaProps{
							Description: "Selector is a label selector for pods for which metrics should be collected",
							Ref:         ref("k8s.io/apimachinery/pkg/apis/meta/v1.LabelSelector"),
						},
					},
					"webhook": {
						SchemaProps: spec.SchemaProps{
							Description: "webhook points to the web endpoint that going to get metric alerts",
							Ref:         ref("./pkg/apis/metrics/v1alpha1.Webhook"),
						},
					},
					"metrics": {
						VendorExtensible: spec.VendorExtensible{
							Extensions: spec.Extensions{
								"x-kubernetes-list-type": "set",
							},
						},
						SchemaProps: spec.SchemaProps{
							Description: "metrics contains the specifications for metrics thresholds used to trigger webhook",
							Type:        []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Ref: ref("./pkg/apis/metrics/v1alpha1.MetricSpec"),
									},
								},
							},
						},
					},
					"scrapeInterval": {
						SchemaProps: spec.SchemaProps{
							Description: "scrapeInterval defines how frequently to scrape metrics",
							Ref:         ref("k8s.io/apimachinery/pkg/apis/meta/v1.Duration"),
						},
					},
				},
				Required: []string{"selector", "webhook", "metrics", "scrapeInterval"},
			},
		},
		Dependencies: []string{
			"./pkg/apis/metrics/v1alpha1.MetricSpec", "./pkg/apis/metrics/v1alpha1.Webhook", "k8s.io/apimachinery/pkg/apis/meta/v1.Duration", "k8s.io/apimachinery/pkg/apis/meta/v1.LabelSelector"},
	}
}

func schema_pkg_apis_metrics_v1alpha1_MetricWebhookStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "MetricWebhookStatus defines the observed state of MetricWebhook",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"metrics": {
						VendorExtensible: spec.VendorExtensible{
							Extensions: spec.Extensions{
								"x-kubernetes-list-type": "set",
							},
						},
						SchemaProps: spec.SchemaProps{
							Description: "metrics is the last read state of the metrics used by this MetricWebhook.",
							Type:        []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Ref: ref("./pkg/apis/metrics/v1alpha1.MetricStatus"),
									},
								},
							},
						},
					},
				},
			},
		},
		Dependencies: []string{
			"./pkg/apis/metrics/v1alpha1.MetricStatus"},
	}
}

func schema_pkg_apis_metrics_v1alpha1_PodsMetricSource(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "PodsMetricSource indicates when to call webhook on a metric describing each pod matching the selector (for example, transactions-processed-per-second). The values will be averaged together before being compared to the target value.",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"name": {
						SchemaProps: spec.SchemaProps{
							Description: "name is the name of the metric in question",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"targetAverageValue": {
						SchemaProps: spec.SchemaProps{
							Description: "targetAverageValue is the target value of the average of the metric across all relevant pods (as a quantity)",
							Ref:         ref("k8s.io/apimachinery/pkg/api/resource.Quantity"),
						},
					},
				},
				Required: []string{"name", "targetAverageValue"},
			},
		},
		Dependencies: []string{
			"k8s.io/apimachinery/pkg/api/resource.Quantity"},
	}
}

func schema_pkg_apis_metrics_v1alpha1_PodsMetricStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "PodsMetricStatus indicates the current value of a metric describing each pod matching the selector (for example, transactions-processed-per-second).",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"name": {
						SchemaProps: spec.SchemaProps{
							Description: "name is the name of the metric in question",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"currentAverageValue": {
						SchemaProps: spec.SchemaProps{
							Description: "currentAverageValue is the current value of the average of the metric across all relevant pods (as a quantity)",
							Ref:         ref("k8s.io/apimachinery/pkg/api/resource.Quantity"),
						},
					},
				},
				Required: []string{"name", "currentAverageValue"},
			},
		},
		Dependencies: []string{
			"k8s.io/apimachinery/pkg/api/resource.Quantity"},
	}
}

func schema_pkg_apis_metrics_v1alpha1_ResourceMetricSource(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "ResourceMetricSource indicates when to call webhook on a resource metric known to Kubernetes, as specified in requests and limits, describing each pod matching the selector (e.g. CPU or memory). The values will be averaged together before being compared to the target value.",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"name": {
						SchemaProps: spec.SchemaProps{
							Description: "name is the name of the resource in question.",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"targetAverageUtilization": {
						SchemaProps: spec.SchemaProps{
							Description: "targetAverageUtilization is the target value of the average of the resource metric across all relevant pods, represented as a percentage of the requested value of the resource for the pods.",
							Type:        []string{"integer"},
							Format:      "int32",
						},
					},
					"targetAverageValue": {
						SchemaProps: spec.SchemaProps{
							Description: "targetAverageValue is the target value of the average of the resource metric across all relevant pods, as a raw value (instead of as a percentage of the request), similar to the \"pods\" metric source type.",
							Ref:         ref("k8s.io/apimachinery/pkg/api/resource.Quantity"),
						},
					},
				},
				Required: []string{"name"},
			},
		},
		Dependencies: []string{
			"k8s.io/apimachinery/pkg/api/resource.Quantity"},
	}
}

func schema_pkg_apis_metrics_v1alpha1_ResourceMetricStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "ResourceMetricStatus indicates the current value of a resource metric known to Kubernetes, as specified in requests and limits, describing each pod matching the selector (e.g. CPU or memory). Such metrics are built in to Kubernetes, and have special scaling options on top of those available to normal per-pod metrics using the \"pods\" source.",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"name": {
						SchemaProps: spec.SchemaProps{
							Description: "name is the name of the resource in question.",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"currentAverageUtilization": {
						SchemaProps: spec.SchemaProps{
							Description: "currentAverageUtilization is the current value of the average of the resource metric across all relevant pods, represented as a percentage of the requested value of the resource for the pods.  It will only be present if `targetAverageValue` was set in the corresponding metric specification.",
							Type:        []string{"integer"},
							Format:      "int32",
						},
					},
					"currentAverageValue": {
						SchemaProps: spec.SchemaProps{
							Description: "currentAverageValue is the current value of the average of the resource metric across all relevant pods, as a raw value (instead of as a percentage of the request), similar to the \"pods\" metric source type. It will always be set, regardless of the corresponding metric specification.",
							Ref:         ref("k8s.io/apimachinery/pkg/api/resource.Quantity"),
						},
					},
				},
				Required: []string{"name", "currentAverageValue"},
			},
		},
		Dependencies: []string{
			"k8s.io/apimachinery/pkg/api/resource.Quantity"},
	}
}

func schema_pkg_apis_metrics_v1alpha1_Webhook(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "Webhook describes the web endpoint that the operator calls on metrics reaching their thresholds",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"url": {
						SchemaProps: spec.SchemaProps{
							Description: "Explicit URL to hit, instead of matching service",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"service": {
						SchemaProps: spec.SchemaProps{
							Description: "Referent service",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"port": {
						SchemaProps: spec.SchemaProps{
							Description: "Service port the webserver serves on",
							Type:        []string{"integer"},
							Format:      "int32",
						},
					},
					"path": {
						SchemaProps: spec.SchemaProps{
							Description: "URL path to the webhook",
							Type:        []string{"string"},
							Format:      "",
						},
					},
				},
			},
		},
	}
}
