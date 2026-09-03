package libs

import "github.com/google/uuid"

func GenerateUlid() string {

	return uuid.New().String()
}
