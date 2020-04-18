package logstf

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFetchAPI(t *testing.T) {
	ar, err := FetchAPI(2000000)
	assert.NoError(t, err)
	assert.NotNil(t, ar)
	if ar != nil {
		assert.Equal(t, `na.serveme.tf #237378 - faf vs BiBBa`, ar.Info.Title)
	}
}

func TestReadJSON(t *testing.T) {
	js, e := ReadJSON(672005)
	assert.NoError(t, e)
	s := js.Summary()
	assert.Equal(t, 3, s.ScoreBlu)
	assert.Equal(t, 2, s.ScoreRed)
}
