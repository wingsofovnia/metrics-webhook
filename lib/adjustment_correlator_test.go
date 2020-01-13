package lib

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/wingsofovnia/metrics-webhook/pkg/apis/metrics/v1alpha1"
)

func TestAdjustmentCorrelator_Recorrelation(t *testing.T) {
	correlator := NewAdjustmentCorrelator(-1) // cap < 1 ~ manual Recorrelation()

	// Round one
	correlator.RegisterAdjustments(
		v1alpha1.MetricReport{
			v1alpha1.MetricNotification{
				Type:                      v1alpha1.Alert,
				Name:                      "cpu",
				CurrentAverageValue:       func() resource.Quantity { q, _ := resource.ParseQuantity("100Mi"); return q }(),
				CurrentAverageUtilization: func(i int32) *int32 { return &i }(100),
			},
			v1alpha1.MetricNotification{
				Type:                      v1alpha1.Alert,
				Name:                      "ram",
				CurrentAverageValue:       func() resource.Quantity { q, _ := resource.ParseQuantity("40Mi"); return q }(),
				CurrentAverageUtilization: func(i int32) *int32 { return &i }(40),
			},
		}, Adjustments{
			"quality": float64(-8),
			"pages":   float64(-4),
		})

	// Round two
	correlator.RegisterAdjustments(
		v1alpha1.MetricReport{
			v1alpha1.MetricNotification{
				Type:                      v1alpha1.Alert,
				Name:                      "cpu",
				CurrentAverageValue:       func() resource.Quantity { q, _ := resource.ParseQuantity("60Mi"); return q }(),
				CurrentAverageUtilization: func(i int32) *int32 { return &i }(60),
			},
			v1alpha1.MetricNotification{
				Type:                      v1alpha1.Alert,
				Name:                      "ram",
				CurrentAverageValue:       func() resource.Quantity { q, _ := resource.ParseQuantity("20Mi"); return q }(),
				CurrentAverageUtilization: func(i int32) *int32 { return &i }(20),
			},
		}, Adjustments{
			"quality": float64(-6),
			"pages":   float64(-2),
		})

	// Round three
	correlator.RegisterAdjustments(
		v1alpha1.MetricReport{
			v1alpha1.MetricNotification{
				Type:                      v1alpha1.Cooldown,
				Name:                      "cpu",
				CurrentAverageValue:       func() resource.Quantity { q, _ := resource.ParseQuantity("40Mi"); return q }(),
				CurrentAverageUtilization: func(i int32) *int32 { return &i }(40),
			},
			v1alpha1.MetricNotification{
				Type:                      v1alpha1.Cooldown,
				Name:                      "ram",
				CurrentAverageValue:       func() resource.Quantity { q, _ := resource.ParseQuantity("10Mi"); return q }(),
				CurrentAverageUtilization: func(i int32) *int32 { return &i }(10),
			},
		}, Adjustments{})

	correlator.Recorrelate()

	assert.Contains(t, correlator.averageCorrelations, "quality")
	assert.Contains(t, correlator.averageCorrelations["quality"], "cpu")
	assert.Contains(t, correlator.averageCorrelations["quality"], "ram")
	assert.InDelta(t,
		(40.0/2.0/-8.0+20.0/2.0/-6.0)/2.0, // Avg between first and second improvement (measurement delta)
		correlator.averageCorrelations["quality"]["cpu"].Value.Utilization,
		0.1)
	assert.InDelta(t,
		(20.0/2.0/-8.0+10.0/2.0/-6.0)/2.0,
		correlator.averageCorrelations["quality"]["ram"].Value.Utilization,
		0.1)

	assert.Contains(t, correlator.averageCorrelations, "pages")
	assert.Contains(t, correlator.averageCorrelations["pages"], "cpu")
	assert.Contains(t, correlator.averageCorrelations["pages"], "ram")
	assert.InDelta(t,
		(40.0/2.0/-4.0+20.0/2.0/-2.0)/2.0, // Avg between first and second improvement (measurement delta)
		correlator.averageCorrelations["pages"]["cpu"].Value.Utilization,
		0.1)
	assert.InDelta(t,
		(20.0/2.0/-4.0+20.0/2.0/-4.0)/2.0,
		correlator.averageCorrelations["pages"]["ram"].Value.Utilization,
		0.1)
}

func TestAdjustmentCorrelator_SuggestAdjustments(t *testing.T) {
	correlator := NewAdjustmentCorrelator(-1) // cap < 1 ~ manual Recorrelation()

	// Round one
	correlator.RegisterAdjustments(
		v1alpha1.MetricReport{
			v1alpha1.MetricNotification{
				Type:                      v1alpha1.Alert,
				Name:                      "cpu",
				CurrentAverageValue:       func() resource.Quantity { q, _ := resource.ParseQuantity("100Mi"); return q }(),
				CurrentAverageUtilization: func(i int32) *int32 { return &i }(100),
			},
			v1alpha1.MetricNotification{
				Type:                      v1alpha1.Alert,
				Name:                      "ram",
				CurrentAverageValue:       func() resource.Quantity { q, _ := resource.ParseQuantity("100Mi"); return q }(),
				CurrentAverageUtilization: func(i int32) *int32 { return &i }(100),
			},
		}, Adjustments{
			"quality": float64(-5),
			"pages":   float64(-5),
		})

	// Round two
	correlator.RegisterAdjustments(
		v1alpha1.MetricReport{
			v1alpha1.MetricNotification{
				Type:                      v1alpha1.Alert,
				Name:                      "cpu",
				CurrentAverageValue:       func() resource.Quantity { q, _ := resource.ParseQuantity("50Mi"); return q }(),
				CurrentAverageUtilization: func(i int32) *int32 { return &i }(50),
			},
			v1alpha1.MetricNotification{
				Type:                      v1alpha1.Alert,
				Name:                      "ram",
				CurrentAverageValue:       func() resource.Quantity { q, _ := resource.ParseQuantity("50Mi"); return q }(),
				CurrentAverageUtilization: func(i int32) *int32 { return &i }(50),
			},
		}, Adjustments{})

	correlator.Recorrelate()

	assert.Contains(t, correlator.averageCorrelations, "quality")
	assert.Contains(t, correlator.averageCorrelations["quality"], "cpu")
	assert.Contains(t, correlator.averageCorrelations["quality"], "ram")
	assert.InDelta(t,
		50.0/2.0/-5.0, // Avg between first and second improvement (measurement delta)
		correlator.averageCorrelations["quality"]["cpu"].Value.Utilization,
		0.1)
	assert.InDelta(t,
		50.0/2.0/-5.0,
		correlator.averageCorrelations["quality"]["ram"].Value.Utilization,
		0.1)

	assert.Contains(t, correlator.averageCorrelations, "pages")
	assert.Contains(t, correlator.averageCorrelations["pages"], "cpu")
	assert.Contains(t, correlator.averageCorrelations["pages"], "ram")
	assert.InDelta(t,
		50.0/2.0/-5.0, // Avg between first and second improvement (measurement delta)
		correlator.averageCorrelations["pages"]["cpu"].Value.Utilization,
		0.1)
	assert.InDelta(t,
		50.0/2.0/-5.0,
		correlator.averageCorrelations["pages"]["ram"].Value.Utilization,
		0.1)

	// Ask for a suggestion for a report that matches the situation exact to observed one
	suggestions := correlator.SuggestAdjustments(v1alpha1.MetricReport{
		v1alpha1.MetricNotification{
			Type:                      v1alpha1.Alert,
			Name:                      "cpu",
			CurrentAverageValue:       func() resource.Quantity { q, _ := resource.ParseQuantity("100Mi"); return q }(),
			TargetAverageValue:        func() *resource.Quantity { q, _ := resource.ParseQuantity("50Mi"); return &q }(),
			CurrentAverageUtilization: func(i int32) *int32 { return &i }(100),
			TargetAverageUtilization:  func(i int32) *int32 { return &i }(50),
		},
		v1alpha1.MetricNotification{
			Type:                      v1alpha1.Alert,
			Name:                      "ram",
			CurrentAverageValue:       func() resource.Quantity { q, _ := resource.ParseQuantity("100Mi"); return q }(),
			TargetAverageValue:        func() *resource.Quantity { q, _ := resource.ParseQuantity("50Mi"); return &q }(),
			CurrentAverageUtilization: func(i int32) *int32 { return &i }(100),
			TargetAverageUtilization:  func(i int32) *int32 { return &i }(50),
		},
	})

	// Should suggest exactly the adjustments that was applied in the same situation
	assert.InDelta(t, -5.0, suggestions["quality"], 0.1)
	assert.InDelta(t, -5.0, suggestions["pages"], 0.1)
}
