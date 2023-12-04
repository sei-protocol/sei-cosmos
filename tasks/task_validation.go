package tasks

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"sort"
)

func (s *scheduler) findConflicts(task *TxTask) (bool, []int) {
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

func (s *scheduler) invalidateTask(task *TxTask) {
	for _, mv := range s.multiVersionStores {
		mv.InvalidateWriteset(task.Index, task.Incarnation)
		mv.ClearReadset(task.Index)
		mv.ClearIterateset(task.Index)
	}
}

func (s *scheduler) mockValidateTask(ctx sdk.Context, task *TxTask) {
	task.SetStatus(statusValidated)
}

func (s *scheduler) validateTask(ctx sdk.Context, task *TxTask) {
	_, span := s.traceSpan(ctx, "SchedulerValidate", task)
	defer span.End()

	valid, conflicts := s.findConflicts(task)
	task.Parents = conflicts

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
