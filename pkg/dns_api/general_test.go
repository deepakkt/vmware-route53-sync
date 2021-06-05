package dns_api

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

func TestEnvOut(t *testing.T) {
	assert.NotEqual(t, GetEnv("PATH"), "")
	assert.Equal(t, GetEnv("I-DONT-EXIST"), "")
}

func TestSyncFrequency(t *testing.T) {
	assert.Equal(t, GetSyncFrequencySeconds(), time.Duration(600))

	os.Setenv("DNS_SYNC_FREQUENCY", "300")
	assert.Equal(t, GetSyncFrequencySeconds(), time.Duration(300))

	os.Setenv("DNS_SYNC_FREQUENCY", "invalid")
	assert.Equal(t, GetSyncFrequencySeconds(), time.Duration(600))
}
