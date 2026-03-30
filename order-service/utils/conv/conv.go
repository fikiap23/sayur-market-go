package conv

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

func StringToInt64(s string) (int64, error) {
	newData, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}

	return newData, nil
}

func Int64PointerToInt64(num *int64) int64 {
	if num != nil {
		return *num
	}
	return 0
}

func GenerateOrderCode() string {
	return fmt.Sprintf("ORD-%s-%d", time.Now().Format("20060102150405"), rand.Intn(1000000))
}
