package tasksv2

import (
	"github.com/google/uuid"

	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type Timer struct {
	name    string
	mx      sync.Mutex
	reports map[string]*TimerReport
	ch      chan timeEvt
}

type timeEvt struct {
	start     bool
	name      string
	id        string
	timestamp time.Time
}

type TimerReport struct {
	name    string
	initial time.Time
	starts  map[string]time.Time
	times   []time.Duration
}

func NewTimer(name string) *Timer {
	t := &Timer{
		name:    name,
		reports: make(map[string]*TimerReport),
		ch:      make(chan timeEvt, 10000),
	}
	return t
}

func (t *Timer) PrintReport() {
	t.mx.Lock()
	defer t.mx.Unlock()

	var reports []*TimerReport
	for name, rpt := range t.reports {
		rpt.name = name
		reports = append(reports, rpt)
	}

	// Sort the slice by the sum of durations
	sort.Slice(reports, func(i, j int) bool {
		sumI := time.Duration(0)
		for _, d := range reports[i].times {
			sumI += d
		}

		sumJ := time.Duration(0)
		for _, d := range reports[j].times {
			sumJ += d
		}

		return sumI < sumJ
	})

	lines := []string{}
	for _, rpt := range reports {
		var sum time.Duration
		count := len(rpt.times)
		minDuration := time.Hour
		maxDuration := time.Duration(0)
		for _, d := range rpt.times {
			sum += d
			if d < minDuration {
				minDuration = d
			}
			if d > maxDuration {
				maxDuration = d
			}
		}
		if count == 0 {
			continue
		}
		avg := sum / time.Duration(count)
		lines = append(lines, fmt.Sprintf("%-15s: \tsum=%-15s\tavg=%-15s\tmin=%-15s\tmax=%-15s\tcount=%-15d %s", t.name, sum, avg, minDuration, maxDuration, count, rpt.name))
	}
	fmt.Println(strings.Join(lines, "\n"))
}

func WithTimer(t *Timer, name string, work func()) {
	id := t.Start(name)
	work()
	t.End(name, id)
}

func (t *Timer) Start(name string) string {
	id := uuid.New().String()
	go func() {
		t.mx.Lock()
		defer t.mx.Unlock()

		if _, ok := t.reports[name]; !ok {
			t.reports[name] = &TimerReport{
				starts: make(map[string]time.Time),
				times:  nil,
			}
		}
		t.reports[name].starts[id] = time.Now()
	}()
	return id
}

func (t *Timer) End(name string, id string) {
	t.mx.Lock()
	defer t.mx.Unlock()
	if rpt, ok := t.reports[name]; ok {
		if start, ok := rpt.starts[id]; ok {
			rpt.times = append(rpt.times, time.Now().Sub(start))
		}
	}
}
