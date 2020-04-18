package logstf

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestParseLatestLogId(t *testing.T) {
	b, err := ioutil.ReadFile(getExamplePath("logs_tf_index.html"))
	assert.NoError(t, err)
	assert.Equal(t, int64(2401850), parseLatestLogId(b))
}

func TestLogCacheFile(t *testing.T) {
	assert.Equal(t, "1555000/logs_1555152.zip", LogCacheFile(1555152, ZipFormat))
}

func TestUpdateCache(t *testing.T) {
	dir, err := ioutil.TempDir("", "hackerman-logstf")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)
	assert.NoError(t, UpdateCache(dir, -1))
}
