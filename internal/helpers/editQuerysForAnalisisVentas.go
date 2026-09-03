package helpers

import (
	"fmt"
)

func TranformQuerysAddParametersStringsFlag(query string, parameters []any) string {
	return fmt.Sprintf(
		query,
		parameters...,
	)
}
