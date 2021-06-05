package dns_api

import (
	"github.com/stretchr/testify/assert"
	"vmc-dns-sync/pkg/model"

	"os"
	"testing"
)

func TestRegionValue(t *testing.T) {
	assert.Equal(t, "us-east-1", getAWSRegion())

	os.Setenv("R53_SYNC_REGION", "us-west-1")
	assert.Equal(t, "us-west-1", getAWSRegion())
}

func TestBatchSizes(t *testing.T) {
	assert.Equal(t, 25, getUpdateBatchSize())

	os.Setenv("R53_UPDATE_BATCH_SIZE", "INVALID")
	assert.Equal(t, 25, getUpdateBatchSize())

	os.Setenv("R53_UPDATE_BATCH_SIZE", "400")
	assert.Equal(t, 400, getUpdateBatchSize())
}

func TestRoute53Formatter(t *testing.T) {
	assert.Equal(t, "google.com", getAWSAName("http://google.com"))
	assert.Equal(t, "google.com", getAWSAName("google.com"))
	assert.Equal(t, "google.com", getAWSAName("google.com."))
	assert.Equal(t, "www.google.com", getAWSAName("http://www.google.com."))
	assert.Equal(t, "pwww.google.com", getAWSAName("http://pwww.google.com."))
}

func TestAWSActions(t *testing.T) {
	assert.Equal(t, "UPSERT", getAWSAction(model.IPTriageAddR53))
	assert.Equal(t, "DELETE", getAWSAction(model.IPTriageDeleteR53))
	assert.Equal(t, "UPSERT", getAWSAction(model.IPTriageUpdateR53))
	assert.Equal(t, "UNKNOWN", getAWSAction(model.IPTriageNoChange))
	assert.Equal(t, "UNKNOWN", getAWSAction(400))
}

func TestBatching(t *testing.T) {
	sample := make([]int, 100)

	for i := range sample {
		sample[i] = i
	}

	sampleSize := len(sample)

	pairs := getBatchPairs(sampleSize, 25)
	assert.Equal(t, 0, pairs[0].start)
	assert.Equal(t, 25, pairs[0].end)
	assert.Equal(t, 25, pairs[1].start)
	assert.Equal(t, 50, pairs[1].end)
	assert.Equal(t, 50, pairs[2].start)
	assert.Equal(t, 75, pairs[2].end)
	assert.Equal(t, 75, pairs[3].start)
	assert.Equal(t, 100, pairs[3].end)
	assert.Equal(t, 4, len(pairs))

	pairs = getBatchPairs(sampleSize, 150)
	assert.Equal(t, 0, pairs[0].start)
	assert.Equal(t, 100, pairs[0].end)
	assert.Equal(t, 1, len(pairs))

	pairs = getBatchPairs(sampleSize, 1)
	assert.Equal(t, 0, pairs[0].start)
	assert.Equal(t, 1, pairs[0].end)
	assert.Equal(t, 99, pairs[99].start)
	assert.Equal(t, 100, pairs[99].end)
	assert.Equal(t, 100, len(pairs))

	pairs = getBatchPairs(sampleSize, 13)
	assert.Equal(t, 8, len(pairs))
	assert.Equal(t, 0, pairs[0].start)
	assert.Equal(t, 13, pairs[0].end)
	assert.Equal(t, 13, pairs[1].start)
	assert.Equal(t, 26, pairs[1].end)
	assert.Equal(t, 26, pairs[2].start)
	assert.Equal(t, 39, pairs[2].end)
	assert.Equal(t, 39, pairs[3].start)
	assert.Equal(t, 52, pairs[3].end)
	assert.Equal(t, 52, pairs[4].start)
	assert.Equal(t, 65, pairs[4].end)
	assert.Equal(t, 65, pairs[5].start)
	assert.Equal(t, 78, pairs[5].end)
	assert.Equal(t, 78, pairs[6].start)
	assert.Equal(t, 91, pairs[6].end)
	assert.Equal(t, 91, pairs[7].start)
	assert.Equal(t, 100, pairs[7].end)
}