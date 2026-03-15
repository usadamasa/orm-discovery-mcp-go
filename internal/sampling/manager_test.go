package sampling

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/config"
)

func TestManager_CanSample_NilSession(t *testing.T) {
	sm := NewManager(&config.Config{Sampling: config.SamplingOpts{Enabled: true}})
	result := sm.CanSample(nil)
	assert.False(t, result, "CanSample should return false when session is nil")
}
