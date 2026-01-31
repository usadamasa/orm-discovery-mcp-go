package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSamplingManager_CanSample_NilSession(t *testing.T) {
	sm := NewSamplingManager(&Config{EnableSampling: true})
	result := sm.CanSample(nil)
	assert.False(t, result, "CanSample should return false when session is nil")
}
