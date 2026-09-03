package helpers

import (
	"fmt"
	"strings"
)

func TransformSliceToInSqlString(s []string) string {
	var args []string = make([]string, len(s))
	for i, a := range s {
		args[i] = fmt.Sprintf("'%s'", a)
	}

	return strings.Join(args, ",")
}
