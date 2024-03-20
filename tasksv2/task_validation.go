package tasksv2

import (
	"fmt"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *scheduler) findConflicts(task *TxTask) (bool, []int) {
	var conflicts []int
	uniq := make(map[int]struct{})
	valid := true
	for _, mv := range s.multiVersionStores {
		ok, mvConflicts := mv.ValidateTransactionState(task.AbsoluteIndex)
		for _, c := range mvConflicts {
			if _, ok := uniq[c]; !ok {
				conflicts = append(conflicts, c)
				uniq[c] = struct{}{}
			}
		}
		// any non-ok value makes valid false
		valid = valid && ok
	}
	sort.Ints(conflicts)
	return valid, conflicts
}

func (s *scheduler) invalidateTask(task *TxTask) {
	for _, mv := range s.multiVersionStores {
		mv.InvalidateWriteset(task.AbsoluteIndex, task.Incarnation)
		mv.ClearReadset(task.AbsoluteIndex)
		mv.ClearIterateset(task.AbsoluteIndex)
	}
}

func (s *scheduler) mockValidateTask(ctx sdk.Context, task *TxTask) {
	task.SetStatus(statusValidated)
}

func (s *scheduler) validateAll(ctx sdk.Context) {
	rerun := make(map[int]struct{})
	for _, t := range s.tasks {
		if t.IsStatus(statusValidated) {
			s.validateTask(ctx, t)
			if t.IsStatus(statusValidated) {
				continue
			}
		} else {
			// validateTask already invalidates, so we need to invalidate here
			s.invalidateTask(t)
		}
		rerun[t.AbsoluteIndex] = struct{}{}
		// invalidate before incrementing so that old incarnation is invalidated
		t.ResetForExecution()
		t.Increment()
		s.executeTask(t)
		s.validateTask(ctx, t)
		if !t.IsStatus(statusValidated) {
			s.printSummary()
			_, reran := rerun[t.AbsoluteIndex]
			panic(fmt.Errorf("invalid task after sequential execution, index=%d, incarnation=%d, reran=%v", t.AbsoluteIndex, t.Incarnation, reran))
		}
	}
}

func (s *scheduler) validateTask(ctx sdk.Context, task *TxTask) {
	_, span := s.traceSpan(ctx, "SchedulerValidate", task)
	defer span.End()

	if task.Response == nil {
		task.SetStatus(statusInvalid)
		return
	}

	valid, conflicts := s.findConflicts(task)
	for _, c := range conflicts {
		task.Parents.Add(c)
	}

	if !valid {
		s.invalidateTask(task)
		if len(conflicts) > 0 {
			task.SetStatus(statusWaiting)
			return
		}
		task.SetStatus(statusInvalid)
		return
	}

	if len(conflicts) > 0 {
		task.SetStatus(statusWaiting)
		return
	}

	task.SetStatus(statusValidated)

}
