package helpers

import (
	"fmt"
	"time"
)

func GeneratesDatesNoMayorToday(start time.Time, end time.Time) ([]string, error) {
	var resultados []string = []string{}
	formatDate := "20060102"
	today := time.Now()

	if start.After(today) {
		return []string{}, fmt.Errorf("Error dates %s", "La fecha de inicio es mayor a la final permitida hasta ayer")
	}
	if start.After(end) {
		return []string{}, fmt.Errorf("Error dates %s", "La fecha de inicio es mayor a la final permitida hasta ayer")
	}

	resultados = append(resultados, start.Format(formatDate))
	for {
		if start.Format(formatDate) == end.Format(formatDate) {
			break
		}
		start = start.AddDate(0, 0, 1)
		resultados = append(resultados, start.Format(formatDate))
	}

	return resultados, nil
}
