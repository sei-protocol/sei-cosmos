package tasksv2

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/cosmos/cosmos-sdk/store/multiversion"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/abci/types"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"sort"
	"strings"
	"time"
)

// TaskLog logs basic task information for debugging purposes
func TaskLog(task *TxTask, msg string) {
	// helpful for debugging state transitions
	//fmt.Println(fmt.Sprintf("%d: Task(%d/%s/%d):\t%s", time.Now().UnixMicro(), task.AbsoluteIndex, task.status, task.Incarnation, msg))
}

type Endable interface {
	End(options ...trace.SpanEndOption)
}

type mockEndable struct{}

func (m *mockEndable) End(options ...trace.SpanEndOption) {}

func (s *scheduler) traceSpan(ctx sdk.Context, name string, task *TxTask) (sdk.Context, trace.Span) {
	spanCtx, span := s.tracingInfo.StartWithContext(name, ctx.TraceSpanContext())
	if task != nil {
		span.SetAttributes(attribute.String("txHash", task.TxHash))
		span.SetAttributes(attribute.Int("absoluteIndex", task.AbsoluteIndex))
		span.SetAttributes(attribute.Int("txIncarnation", task.Incarnation))
	}
	ctx = ctx.WithTraceSpanContext(spanCtx)
	return ctx, span
}

// hangDebug prints a message after 1s if not cancelled (detects hangs)
func hangDebug(msg func()) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	ticker := time.NewTicker(1 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				msg()
			case <-ctx.Done():
				return
			}
		}
	}()
	return cancel
}

func toTasks(ctx sdk.Context, reqs []*sdk.DeliverTxEntry) []*TxTask {
	res := make([]*TxTask, 0, len(reqs))
	for _, r := range reqs {
		res = append(res, &TxTask{
			Request:       r.Request,
			AbsoluteIndex: r.AbsoluteIndex,
			SdkTx:         r.SdkTx,
			Checksum:      r.Checksum,
			TxHash:        fmt.Sprintf("%X", sha256.Sum256(r.Request.Tx)),
			Parents: &intSetMap{
				m: make(map[int]struct{}),
			},
			Dependents: &intSetMap{
				m: make(map[int]struct{}),
			},
			Ctx:    ctx,
			status: statusPending,
		})
	}
	// sort by AbsoluteIndex
	sort.Slice(res, func(i, j int) bool {
		return res[i].AbsoluteIndex < res[j].AbsoluteIndex
	})
	return res
}

func collectResponses(tasks []*TxTask) []types.ResponseDeliverTx {
	res := make([]types.ResponseDeliverTx, 0, len(tasks))
	for _, t := range tasks {
		res = append(res, *t.Response)
	}
	return res
}

func (s *scheduler) initMultiVersionStore(ctx sdk.Context) {
	mvs := make(map[sdk.StoreKey]multiversion.MultiVersionStore)
	keys := ctx.MultiStore().StoreKeys()
	for _, sk := range keys {
		mvs[sk] = multiversion.NewMultiVersionStore(ctx.MultiStore().GetKVStore(sk))
	}
	s.multiVersionStores = mvs
}

func (s *scheduler) PrefillEstimates(reqs []*sdk.DeliverTxEntry) {
	// iterate over TXs, update estimated writesets where applicable
	for i, req := range reqs {
		mappedWritesets := req.EstimatedWritesets
		// order shouldnt matter for storeKeys because each storeKey partitioned MVS is independent
		for storeKey, writeset := range mappedWritesets {
			// we use `-1` to indicate a prefill incarnation
			s.multiVersionStores[storeKey].SetEstimatedWriteset(i, -1, writeset)
		}
	}
}

func (s *scheduler) printSummary() {
	var lines []string
	for i, t := range s.tasks {
		line := fmt.Sprintf("Task index=%d absoluteIndex=%d status=%s incarnation=%d parents=%v responseExists=%v", i, t.AbsoluteIndex, t.Status(), t.Incarnation, t.Parents.List(), t.Response != nil)
		lines = append(lines, line)
	}

	fmt.Println(strings.Join(lines, "\n"))
}
