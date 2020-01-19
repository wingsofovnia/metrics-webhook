package lib

import (
	"k8s.io/apimachinery/pkg/api/resource"
	"strconv"
)

type Measurement struct {
	Value       float64
	Utilization float64
}

func NewMeasurement(quantity resource.Quantity, utilization *int32) Measurement {
	utilizationAsFloat64 := float64(0)
	if utilization != nil {
		utilizationAsFloat64 = float64(*utilization)
	}

	return Measurement{
		Value:       quantityAsFloat64(quantity),
		Utilization: utilizationAsFloat64,
	}
}

func (i Measurement) Sub(m Measurement) Measurement {
	return Measurement{
		Value:       i.Value - m.Value,
		Utilization: i.Utilization - m.Utilization,
	}
}

func (i Measurement) GoesInto(m Measurement) float64 {
	utilTimes := float64(0)
	if i.Utilization != 0 && m.Utilization != 0 {
		utilTimes = m.Utilization / i.Utilization
	}

	valTimes := float64(0)
	if i.Value != 0 {
		valTimes = m.Value / i.Value
	}

	if utilTimes != 0 && valTimes != 0 {
		return (utilTimes + valTimes) / 2
	} else if utilTimes != 0 {
		return utilTimes
	} else {
		return valTimes
	}
}

func (i Measurement) Scale(f float64) Measurement {
	return Measurement{
		Value:       i.Value * f,
		Utilization: i.Utilization * f,
	}
}

type AverageMeasurement struct {
	Value Measurement
	Among int
}

func NewAverageMeasurement(improvements ...Measurement) AverageMeasurement {
	if len(improvements) == 0 {
		return AverageMeasurement{
			Value: Measurement{},
			Among: 0,
		}
	}

	averageValueSum := float64(0)
	averageUtilizationSum := float64(0)

	for _, improvement := range improvements {
		averageUtilizationSum = averageUtilizationSum + improvement.Utilization
		averageValueSum = averageValueSum + improvement.Value
	}

	return AverageMeasurement{
		Value: Measurement{
			Value:       averageValueSum / float64(len(improvements)),
			Utilization: averageUtilizationSum / float64(len(improvements)),
		},
		Among: len(improvements),
	}
}

func (a AverageMeasurement) Concat(improvements ...Measurement) AverageMeasurement {
	for i := 0; i < a.Among; i++ {
		improvements = append(improvements, a.Value)
	}
	return NewAverageMeasurement(improvements...)
}

func quantityAsFloat64(quantity resource.Quantity) float64 {
	quantityCopy := (&quantity).DeepCopy()
	quantityAsFloat64, _ := strconv.ParseFloat((&quantityCopy).AsDec().String(), 64)
	return quantityAsFloat64
}
