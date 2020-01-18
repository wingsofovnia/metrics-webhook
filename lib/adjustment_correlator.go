package lib

import (
	"strconv"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/wingsofovnia/metrics-webhook/pkg/apis/metrics/v1alpha1"
)

type Config = string
type Metric = string

type Adjustments = map[Config]float64
type Measurements = map[Metric]Measurement

type Correlations = map[Config]map[Metric][]Measurement
type AverageCorrelations = map[Config]map[Metric]AverageMeasurement

type AdjustmentRound struct {
	Measurements Measurements
	Adjustments  Adjustments
}

type AdjustmentCorrelator struct {
	adjustmentsBuffer         []AdjustmentRound
	adjustmentsBufferFlushCap int

	averageCorrelations AverageCorrelations
}

const minAdjustmentsBufferFlushCap = 2

func NewAdjustmentCorrelator(adjustmentsBufferFlushCap int) *AdjustmentCorrelator {
	return &AdjustmentCorrelator{
		adjustmentsBufferFlushCap: adjustmentsBufferFlushCap,
		averageCorrelations:       make(AverageCorrelations),
	}
}

func (c *AdjustmentCorrelator) RegisterAdjustments(report v1alpha1.MetricReport, appliedAdjustments Adjustments) {
	reportedMeasurements := make(Measurements)
	for _, m := range report {
		utilization := float64(0)
		if m.CurrentAverageUtilization != nil {
			utilization = float64(*m.CurrentAverageUtilization)
		}

		reportedMeasurements[m.Name] = Measurement{
			Value:       float64FromQuantityUnsafe(m.CurrentAverageValue),
			Utilization: utilization,
		}
	}
	c.adjustmentsBuffer = append(c.adjustmentsBuffer, AdjustmentRound{
		Measurements: reportedMeasurements,
		Adjustments:  appliedAdjustments,
	})

	if c.adjustmentsBufferFlushCap >= minAdjustmentsBufferFlushCap &&
		len(c.adjustmentsBuffer) >= c.adjustmentsBufferFlushCap {
		c.Recorrelate()
	}
}

// Recorrelate correlate changes in metric reports with adjustments made
// (Config changes) in response. This information is then used to calculate
// how changing a Config value by 1.0 improves metrics, i.e. allows to answer
// a question: how will changing Config X by 1.0 improve metric Y value?
//
// Algorithm
//	Given the following input:
//		[0] measurements = [100 (cpu utilization), 40 (ram)], adjustments = [(quality = -8),(pages = -4)]
//		[1] measurements = [60  (cpu utilization), 20 (ram)], adjustments = [(quality = -6),(pages = -2)]
//		[2] measurements = [40  (cpu utilization), 10 (ram)], adjustments = []
//
//	In Stage 1:
//		[0] [(quality = -8),(pages = -4)] -> [40 (100[0] - 60[1], cpu utilization improvement), 20 (40[0] - 20[1], ram improvement)]
//		[1] [(quality = -6),(pages = -2)] -> [20 (60[1] - 40[2],  cpu utilization improvement), 10 (20[1] - 10[2], ram improvement)]
//
//  In Stage 2:
//		(quality = 1) -> {
//			"cpu utilization improvement": [
//				[0]		-2.5 (40/2/-8),																				 2 = len(improvements)
//				[1]		-1.6 (20/2/-6),
//			],
//			"ram improvement": [
//				[0]		-1.25 (20/2/-8),
//				[1]		-0.83 (10/2/-6),
//			]
//		},
//		(pages = 1)   -> {
//			"cpu utilization improvement": [
//				[0]		-5   (40/2/-4),
//				[1]		-5   (20/2/-2),
//			],
//			"ram improvement": [
//				[0]		-2.5  (20/2/-4),
//				[1]		-2.5  (20/2/-4),
//			]
//		}
//
//	In Stage 3:
//		(quality = 1) -> {
//			"cpu utilization improvement": -2.05 ((-2.5 + -1.6) / 2),
//			"ram improvement":             -1.04 ((-1.25 + -0.83) / 2),
//		},
//		(pages = 1)   -> {
//			"cpu utilization improvement": -5    ((-5 + -5) / 2),
//			"ram improvement":             -2.5  ((-2.5 + -2.5) / 2),
//		},
func (c *AdjustmentCorrelator) Recorrelate() {
	if len(c.adjustmentsBuffer) < c.adjustmentsBufferFlushCap {
		return
	}

	correlations := make(Correlations)
	for i, k := 0, 1; k < len(c.adjustmentsBuffer); i, k = i+1, k+1 {
		prevRound := c.adjustmentsBuffer[i]
		currRound := c.adjustmentsBuffer[k]

		roundAdjustments := prevRound.Adjustments
		if len(roundAdjustments) == 0 {
			continue
		}

		// Stage 1: [roundAdjustments] -> [roundImprovements]
		roundImprovements := make(Measurements)
		for metric, prevMeasurement := range prevRound.Measurements {
			currMeasurement := currRound.Measurements[metric]
			roundImprovements[metric] = NewMeasurementDelta(prevMeasurement, currMeasurement)
		}

		// Stage 2
		for config, adjustment := range roundAdjustments {
			if correlations[config] == nil {
				correlations[config] = make(map[Metric][]Measurement)
			}

			for metric, improvement := range roundImprovements {
				scaledImprovement := improvement.Scale(1.0 / float64(len(roundAdjustments)) / adjustment)

				if prev, set := correlations[config][metric]; set {
					correlations[config][metric] = append(prev, scaledImprovement)
				} else {
					correlations[config][metric] = []Measurement{scaledImprovement}
				}
			}
		}

	}

	// Stage 3
	for config, correlation := range correlations {
		if c.averageCorrelations[config] == nil {
			c.averageCorrelations[config] = make(map[Metric]AverageMeasurement)
		}

		for metric, improvements := range correlation {
			if oldCorrelation, set := c.averageCorrelations[config][metric]; set {
				c.averageCorrelations[config][metric] = oldCorrelation.Concat(improvements...)
			} else {
				c.averageCorrelations[config][metric] = NewAverageMeasurement(improvements...)
			}
		}
	}

	c.adjustmentsBuffer = nil
}

func (c *AdjustmentCorrelator) SuggestAdjustments(metricsReported v1alpha1.MetricReport) Adjustments {
	targetMeasurements := make(Measurements)
	for _, notification := range metricsReported {
		if notification.Type != v1alpha1.Alert {
			continue
		}

		var utilizationImprovementNeeded float64
		if notification.CurrentAverageUtilization != nil {
			utilizationImprovementNeeded = float64(*notification.CurrentAverageUtilization - *notification.TargetAverageUtilization)
		}

		var valueImprovementNeeded float64
		if notification.TargetAverageValue != nil {
			valueImprovementNeeded = float64FromQuantityUnsafe(notification.CurrentAverageValue) - float64FromQuantityUnsafe(*notification.TargetAverageValue)
		}

		if utilizationImprovementNeeded > 0 || valueImprovementNeeded > 0 {
			targetMeasurements[notification.Name] = Measurement{
				Value:       valueImprovementNeeded,
				Utilization: utilizationImprovementNeeded,
			}
		}
	}

	if len(targetMeasurements) == 0 {
		return nil
	}

	suggestions := make(Adjustments)
	for config, correlations := range c.averageCorrelations {
		for metric, improvement := range correlations {
			if reqImprovement, requested := targetMeasurements[metric]; requested {
				suggestions[config] = reqImprovement.Divide(improvement.Value) / float64(len(correlations))
			}
		}
	}

	return suggestions
}

func float64FromQuantity(q resource.Quantity) (float64, error) {
	qCp := (&q).DeepCopy()
	return strconv.ParseFloat((&qCp).AsDec().String(), 64)
}

func float64FromQuantityUnsafe(q resource.Quantity) float64 {
	f, err := float64FromQuantity(q)
	if err != nil {
		panic(err)
	}
	return f
}
