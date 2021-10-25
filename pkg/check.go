package pkg

import "time"

type CheckRequest struct {
	Target  Target
	Time    time.Time
	Healthy bool
}

type HealthCheck struct {
	TargetID Target
	Time     time.Time
	Healthy  bool
	Requests []CheckRequest
}
