package lib

import "k8s.io/apimachinery/pkg/api/resource"

type Measurement struct {
	Value       float64
	Utilization float64
}

func NewMeasurement(quantity resource.Quantity, utilization *int32) Measurement {
	utilizationFloat64 := float64(0)
	if utilization != nil {
		utilizationFloat64 = float64(*utilization)
	}
	return Measurement{
		Value:       float64FromQuantityUnsafe(quantity),
		Utilization: utilizationFloat64,
	}
}

func NewMeasurementDelta(was Measurement, now Measurement) Measurement {
	return Measurement{
		Value:       was.Value - now.Value,
		Utilization: was.Utilization - now.Utilization,
	}
}

func (i *Measurement) Scale(f float64) Measurement {
	return Measurement{
		Value:       i.Value * f,
		Utilization: i.Utilization * f,
	}
}

func (i *Measurement) Divide(divider Measurement) float64 {
	return i.Value / divider.Value
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

func (a *AverageMeasurement) Concat(improvements ...Measurement) AverageMeasurement {
	for i := 0; i < a.Among; i++ {
		improvements = append(improvements, a.Value)
	}
	return NewAverageMeasurement(improvements...)
}
