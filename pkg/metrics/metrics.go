package metrics

import "github.com/penglongli/gin-metrics/ginmetrics"

var (
	MetricsMiddleware *ginmetrics.Monitor
	PodMutation       *ginmetrics.Metric
	LabelSuccess      = "SUCCESS"
	LabelFailure      = "FAILURE"
)

func init() {
	MetricsMiddleware = ginmetrics.GetMonitor()
	MetricsMiddleware.SetMetricPath("/metrics")

	PodMutation = &ginmetrics.Metric{
		Type:        ginmetrics.Counter,
		Name:        "mutator_pod_mutations",
		Description: "Pod mutations status",
		Labels:      []string{"namespace", "status"},
	}

	err := MetricsMiddleware.AddMetric(PodMutation)
	if err != nil {
		panic(err)
	}
}
