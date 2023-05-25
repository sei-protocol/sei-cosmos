package types

import fmt "fmt"

func ValidateFeesParams(i interface{}) error {
	v, ok := i.(FeesParams)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	for _, fee := range v.GlobalMinimumFees {
		if err := fee.Validate(); err != nil {
			return err
		}
	}

	return nil
}
