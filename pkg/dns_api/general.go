package dns_api

import (
	"os"
	"strconv"
	"time"
)

func GetSyncFrequencySeconds() time.Duration {
	frequency := GetEnv("DNS_SYNC_FREQUENCY")

	if frequency == "" {
		return 600
	}

	frequencyInt, err := strconv.Atoi(frequency)

	if err != nil {
		return 600
	}

	return time.Duration(frequencyInt)
}

func GetEnv(name string) string {
	if val, exist := os.LookupEnv(name); exist {
		return val
	}
	return ""
}

