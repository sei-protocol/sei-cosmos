package tasks

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"sort"
)

func (s *scheduler) findConflicts(task *deliverTxTask) (bool, []int) {
	var conflicts []int
	uniq := make(map[int]struct{})
	valid := true
	for _, mv := range s.multiVersionStores {
		ok, mvConflicts := mv.ValidateTransactionState(task.Index)
		for _, c := range mvConflicts {
			if _, ok := uniq[c]; !ok {
				conflicts = append(conflicts, c)
				uniq[c] = struct{}{}
			}
		}
		// any non-ok value makes valid false
		valid = ok && valid
	}
	sort.Ints(conflicts)
	return valid, conflicts
}

func (s *scheduler) invalidateTask(task *deliverTxTask) {
	for _, mv := range s.multiVersionStores {
		mv.InvalidateWriteset(task.Index, task.Incarnation)
		mv.ClearReadset(task.Index)
		mv.ClearIterateset(task.Index)
	}
}

func (s *scheduler) validateTask(ctx sdk.Context, task *deliverTxTask) bool {
	// avoids validation races WITHIN a task
	task.LockTask()
	defer task.UnlockTask()

	_, span := s.traceSpan(ctx, "SchedulerValidate", task)
	defer span.End()

	if valid, conflicts := s.findConflicts(task); !valid {
		s.invalidateTask(task)
		task.SetStatus(statusInvalid)
		if len(conflicts) > 0 {
			task.Dependencies = conflicts
		}
		return false
	} else if len(conflicts) == 0 {
		// mark as validated, which will avoid re-validating unless a lower-index re-validates
		task.SetStatus(statusValidated)
		return true
	} else {
		task.Dependencies = conflicts
	}
	return false
}
