package usage

import "time"

// Aggregate combines multiple events into a summary.
// This is a PURE function.
func Aggregate(events []Event, periodStart, periodEnd time.Time) Summary {
	if len(events) == 0 {
		return Summary{
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
		}
	}

	var (
		requestCount  int64
		computeUnits  float64
		bytesIn       int64
		bytesOut      int64
		errorCount    int64
		totalLatency  int64
		userID        string
	)

	for _, e := range events {
		if userID == "" {
			userID = e.UserID
		}

		requestCount++
		computeUnits += e.CostMultiplier
		bytesIn += e.RequestBytes
		bytesOut += e.ResponseBytes
		totalLatency += e.LatencyMs

		if e.StatusCode >= 400 {
			errorCount++
		}
	}

	var avgLatency int64
	if requestCount > 0 {
		avgLatency = totalLatency / requestCount
	}

	return Summary{
		UserID:       userID,
		PeriodStart:  periodStart,
		PeriodEnd:    periodEnd,
		RequestCount: requestCount,
		ComputeUnits: computeUnits,
		BytesIn:      bytesIn,
		BytesOut:     bytesOut,
		ErrorCount:   errorCount,
		AvgLatencyMs: avgLatency,
	}
}

// MergeSummaries combines multiple summaries.
// This is a PURE function.
func MergeSummaries(summaries ...Summary) Summary {
	if len(summaries) == 0 {
		return Summary{}
	}

	result := summaries[0]
	for _, s := range summaries[1:] {
		result.RequestCount += s.RequestCount
		result.ComputeUnits += s.ComputeUnits
		result.BytesIn += s.BytesIn
		result.BytesOut += s.BytesOut
		result.ErrorCount += s.ErrorCount

		// Weighted average for latency
		if result.RequestCount > 0 {
			total := result.AvgLatencyMs*result.RequestCount + s.AvgLatencyMs*s.RequestCount
			result.AvgLatencyMs = total / (result.RequestCount + s.RequestCount)
		}

		// Expand period bounds
		if s.PeriodStart.Before(result.PeriodStart) {
			result.PeriodStart = s.PeriodStart
		}
		if s.PeriodEnd.After(result.PeriodEnd) {
			result.PeriodEnd = s.PeriodEnd
		}
	}

	return result
}

// CheckQuota checks usage against quota limits.
// This is a PURE function.
func CheckQuota(summary Summary, quota Quota) QuotaStatus {
	status := QuotaStatus{
		RequestsUsed:  summary.RequestCount,
		RequestsLimit: quota.RequestsPerMonth,
		BytesUsed:     summary.BytesIn + summary.BytesOut,
		BytesLimit:    quota.BytesPerMonth,
	}

	// Calculate percentages
	if quota.RequestsPerMonth > 0 {
		status.RequestsPercent = float64(summary.RequestCount) / float64(quota.RequestsPerMonth) * 100
		if summary.RequestCount > quota.RequestsPerMonth {
			status.IsOverQuota = true
			status.OverageRequests = summary.RequestCount - quota.RequestsPerMonth
		}
	}

	if quota.BytesPerMonth > 0 {
		totalBytes := summary.BytesIn + summary.BytesOut
		status.BytesPercent = float64(totalBytes) / float64(quota.BytesPerMonth) * 100
		if totalBytes > quota.BytesPerMonth {
			status.IsOverQuota = true
		}
	}

	return status
}

// CalculateOverage calculates overage charges.
// This is a PURE function.
func CalculateOverage(usage int64, included int64, pricePerUnit int64) int64 {
	if usage <= included || included < 0 { // -1 means unlimited
		return 0
	}
	return (usage - included) * pricePerUnit
}

// PeriodBounds returns the start and end of a billing period.
// This is a PURE function.
func PeriodBounds(t time.Time) (start, end time.Time) {
	start = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	end = start.AddDate(0, 1, 0).Add(-time.Nanosecond)
	return
}
