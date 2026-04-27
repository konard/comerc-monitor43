package model

type PeriodMetrics struct {
	TotalChecks      int64
	SuccessCount     int64
	FailedCount      int64
	DegradedCount    int64
	UptimePercentage float64
	P50ResponseMs    float64
	P95ResponseMs    float64
	P99ResponseMs    float64
}
