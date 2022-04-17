package config_reward

import (
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

func Test_blocksPerCycle(t *testing.T) {
	assert.Equal(t, 1, 1<<int(math.Floor(float64(0)/blocksPerCycle())))
	assert.Equal(t, 1, 1<<int(math.Floor(float64(1)/blocksPerCycle())))
	assert.Equal(t, 1, 1<<int(math.Floor(float64(2)/blocksPerCycle())))
	assert.Equal(t, 1, 1<<int(math.Floor(float64(3)/blocksPerCycle())))
	assert.Equal(t, 1, 1<<int(math.Floor(float64(315574)/blocksPerCycle())))
	assert.Equal(t, 1, 1<<int(math.Floor(float64(315575)/blocksPerCycle())))
	assert.Equal(t, 2, 1<<int(math.Floor(float64(315576)/blocksPerCycle())))
	assert.Equal(t, 2, 1<<int(math.Floor(float64(315577)/blocksPerCycle())))
	assert.Equal(t, 2, 1<<int(math.Floor(float64(315576*2-2)/blocksPerCycle())))
	assert.Equal(t, 2, 1<<int(math.Floor(float64(315576*2-1)/blocksPerCycle())))
	assert.Equal(t, 4, 1<<int(math.Floor(float64(315576*2)/blocksPerCycle())))
	assert.Equal(t, 4, 1<<int(math.Floor(float64(315576*2+1)/blocksPerCycle())))
	assert.Equal(t, 4, 1<<int(math.Floor(float64(315576*3-1)/blocksPerCycle())))
	assert.Equal(t, 8, 1<<int(math.Floor(float64(315576*3)/blocksPerCycle())))
}
