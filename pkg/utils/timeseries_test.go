package utils

import (
	"sync"
	"testing"
	"time"
)

func TestSnapshotDuringTraffic(t *testing.T) {
	series := NewTimeSeries(time.Hour, 10)
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 1000 {
				series.LogSent(1)
				series.LogRecived(2)
			}
		}()
	}
	for range 1000 {
		snapshot := series.Snapshot()
		if len(snapshot.Points) > 0 {
			snapshot.Points[0].Value.Sent = 999999
		}
	}
	wg.Wait()
	snapshot := series.Snapshot()
	if snapshot.Total.Sent != 4000 || snapshot.Total.Received != 8000 {
		t.Fatalf("totals: %+v", snapshot.Total)
	}
	if snapshot.Points[0].Value.Sent != 4000 {
		t.Fatal("snapshot modified live data")
	}
}
