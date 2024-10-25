package utils

import (
	"sort"
	"sync"
	"time"
)

type DataPoint struct {
	Timestamp time.Time  `json:"timestamp"`
	Value     ProxyStats `json:"value"`
}

type ProxyStats struct {
	Sent     uint64 `json:"sent"`
	Received uint64 `json:"received"`
}

type TimeSeriesData struct {
	Points  []DataPoint `json:"points"`
	Total   ProxyStats  `json:"total"`
	bucket  time.Duration
	maxSize int
}

type TimeSeries struct {
	mu   sync.Mutex
	Data TimeSeriesData
}

func NewTimeSeries(bucket time.Duration, maxSize int) *TimeSeries {
	return &TimeSeries{
		Data: TimeSeriesData{
			Points:  make([]DataPoint, 0, maxSize),
			Total:   ProxyStats{},
			maxSize: maxSize,
			bucket:  bucket,
		},
	}
}

func (tsd *TimeSeriesData) Add(point ProxyStats) {
	now := time.Now().Truncate(tsd.bucket)
	if len(tsd.Points) > 0 && tsd.Points[len(tsd.Points)-1].Timestamp == now {
		current := tsd.Points[len(tsd.Points)-1].Value
		current.Sent += point.Sent
		current.Received += point.Received
		tsd.Points[len(tsd.Points)-1].Value = current
	} else {
		tsd.Points = append(tsd.Points, DataPoint{Timestamp: now, Value: point})
	}
	if len(tsd.Points) > tsd.maxSize {
		tsd.Points = tsd.Points[1:]
	}
	ps := ProxyStats{Sent: 0, Received: 0}
	for _, stat := range tsd.Points {
		ps.Received += stat.Value.Received
		ps.Sent += stat.Value.Sent
	}
	tsd.Total = ps
}

func (ts *TimeSeries) LogSent(value uint64) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.Data.Add(ProxyStats{Sent: value})
}

func (ts *TimeSeries) LogRecived(value uint64) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.Data.Add(ProxyStats{Received: value})
}

func CombineTimeSeriesData(ts1, ts2 TimeSeriesData) TimeSeriesData {
	// Create a new slice for the combined points
	combinedPoints := append(ts1.Points, ts2.Points...)

	// Sort the points by timestamp (optional, in case order matters)
	sort.Slice(combinedPoints, func(i, j int) bool {
		return combinedPoints[i].Timestamp.Before(combinedPoints[j].Timestamp)
	})

	// Sum the totals from both time series
	totalSent := ts1.Total.Sent + ts2.Total.Sent
	totalReceived := ts1.Total.Received + ts2.Total.Received

	// Create the combined TimeSeriesData
	combinedTimeSeries := TimeSeriesData{
		Points:  combinedPoints,
		Total:   ProxyStats{Sent: totalSent, Received: totalReceived},
		bucket:  ts1.bucket,  // assuming the bucket is the same, you can adjust this if necessary
		maxSize: ts1.maxSize, // same for maxSize
	}

	return combinedTimeSeries
}
