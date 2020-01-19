package lib

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestNewMeasurement(t *testing.T) {
	wasAverageUtilization := int32(80)
	wasAverageValue, err := resource.ParseQuantity("1Gi")
	assert.NoError(t, err)
	wasMeasurement := NewMeasurement(wasAverageValue, &wasAverageUtilization)

	nowAverageUtilization := int32(40)
	nowAverageValue, err := resource.ParseQuantity("499Mi")
	assert.NoError(t, err)
	nowMeasurement := NewMeasurement(nowAverageValue, &nowAverageUtilization)

	expectedAverageUtilizationMeasurement := int32(40)
	expectedAverageValueMeasurement, err := resource.ParseQuantity("525Mi") // 1024 - 499
	assert.NoError(t, err)

	improvement := wasMeasurement.Sub(nowMeasurement)
	assert.Equal(t, quantityAsFloat64(expectedAverageValueMeasurement), improvement.Value)
	assert.InDelta(t, float64(expectedAverageUtilizationMeasurement), improvement.Utilization, 0.01)
}

func TestMeasurement_Scale(t *testing.T) {
	wasAverageUtilization := int32(80)
	wasAverageValue, err := resource.ParseQuantity("1Gi")
	assert.NoError(t, err)
	wasMeasurement := NewMeasurement(wasAverageValue, &wasAverageUtilization)

	const factor = float64(0.5)

	expectedAverageUtilization := int32(40)
	expectedAverageValue, err := resource.ParseQuantity("512Mi") // 1024 * 0.5
	assert.NoError(t, err)

	scaledMeasurement := wasMeasurement.Scale(factor)
	assert.Equal(t, quantityAsFloat64(expectedAverageValue), scaledMeasurement.Value)
	assert.InDelta(t, float64(expectedAverageUtilization), scaledMeasurement.Utilization, 0.01)
}

func TestNewAverageMeasurement(t *testing.T) {
	averageUtilizationMeasurementOne := int32(30)
	averageValueMeasurementOne, err := resource.ParseQuantity("1Gi")
	improvementOne := NewMeasurement(averageValueMeasurementOne, &averageUtilizationMeasurementOne)
	assert.NoError(t, err)

	averageUtilizationMeasurementTwo := int32(70)
	averageValueMeasurementTwo, err := resource.ParseQuantity("3Gi")
	assert.NoError(t, err)
	improvementTwo := NewMeasurement(averageValueMeasurementTwo, &averageUtilizationMeasurementTwo)

	expectedAvgAverageUtilization := int32(50)                    // (30 + 70) / 2
	expectedAvgAverageValue, err := resource.ParseQuantity("2Gi") // (1Gi + 3Gi) / 2
	assert.NoError(t, err)

	averageMeasurement := NewAverageMeasurement(improvementOne, improvementTwo)
	assert.Equal(t, 2, averageMeasurement.Among)
	assert.Equal(t, quantityAsFloat64(expectedAvgAverageValue), averageMeasurement.Value.Value)
	assert.InDelta(t, float64(expectedAvgAverageUtilization), averageMeasurement.Value.Utilization, 0.01)
}

func TestAverageMeasurement_Concat(t *testing.T) {
	averageUtilizationMeasurementFirst := int32(10)
	averageValueMeasurementFirst, err := resource.ParseQuantity("1Gi")
	improvementFirst := NewMeasurement(averageValueMeasurementFirst, &averageUtilizationMeasurementFirst)
	assert.NoError(t, err)

	averageUtilizationMeasurementMedian := int32(20)
	averageValueMeasurementMedian, err := resource.ParseQuantity("2Gi")
	assert.NoError(t, err)
	improvementMedian := NewMeasurement(averageValueMeasurementMedian, &averageUtilizationMeasurementMedian)

	firstAndMedianMeasurement := NewAverageMeasurement(improvementFirst, improvementMedian)

	averageUtilizationMeasurementLast := int32(30)
	averageValueMeasurementLast, err := resource.ParseQuantity("3Gi")
	assert.NoError(t, err)
	improvementLast := NewMeasurement(averageValueMeasurementLast, &averageUtilizationMeasurementLast)

	expectedAvgAverageUtilization := averageUtilizationMeasurementMedian
	expectedAvgAverageValue := averageValueMeasurementMedian
	assert.NoError(t, err)

	firstAndMedianAndLastMeasurement := firstAndMedianMeasurement.Concat(improvementLast)
	assert.Equal(t, 3, firstAndMedianAndLastMeasurement.Among)
	assert.Equal(t, quantityAsFloat64(expectedAvgAverageValue), firstAndMedianAndLastMeasurement.Value.Value)
	assert.InDelta(t, float64(expectedAvgAverageUtilization), firstAndMedianAndLastMeasurement.Value.Utilization, 0.01)
}

func TestMeasurement_GoesInto(t *testing.T) {
	firstAverageUtilization := int32(80)
	firstAverageValue, err := resource.ParseQuantity("100Mi")
	assert.NoError(t, err)
	fullSmallerMeasurement := NewMeasurement(firstAverageValue, &firstAverageUtilization)

	secondAverageUtilization := int32(160)
	secondAverageValue, err := resource.ParseQuantity("200Mi")
	assert.NoError(t, err)
	fullBiggerMeasurement := NewMeasurement(secondAverageValue, &secondAverageUtilization)

	assert.InDelta(t, float64(2), fullSmallerMeasurement.GoesInto(fullBiggerMeasurement), 0.01)

	onlyValSmallerMeasurement := NewMeasurement(firstAverageValue, nil)
	onlyValBiggerMeasurement := NewMeasurement(secondAverageValue, nil)

	assert.InDelta(t, float64(2), onlyValSmallerMeasurement.GoesInto(onlyValBiggerMeasurement), 0.01)
}
