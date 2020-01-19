package main

import (
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

var raplDir string

const maxWattage uint64 = 1500000

func TestMain(m *testing.M) {
	var err error
	raplDir, err = ioutil.TempDir("", raplPrefixCPU)
	if err != nil {
		log.Fatal(err)
	}

	defer os.RemoveAll(raplDir)

	// Create temporary directory that mocks powercap subsytem
	zonePath := filepath.Join(raplDir, raplPrefixCPU+":0")
	err = os.Mkdir(zonePath, 755)
	if err != nil {
		log.Fatal(err)
	}

	initialWatts := strconv.FormatUint(maxWattage, 10)

	err = ioutil.WriteFile(filepath.Join(zonePath, maxPowerFileLongWindow), []byte(initialWatts), 0444)
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(filepath.Join(zonePath, powerLimitFileLongWindow), []byte(initialWatts), 0644)
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

// TODO(rdelvalle): Add tests where capping fails
func TestCapNode(t *testing.T) {
	capped, failed, err := capNode(raplDir, 95)
	assert.NoError(t, err)
	assert.Len(t, capped, 1)
	assert.Nil(t, failed)

	t.Run("bad-percentage", func(t *testing.T) {
		capped, failed, err := capNode(raplDir, 1000)
		assert.Error(t, err)
		assert.Nil(t, capped)
		assert.Nil(t, failed)
	})

	t.Run("zero-percent", func(t *testing.T) {
		capped, failed, err := capNode(raplDir, 0)
		assert.Error(t, err)
		assert.Nil(t, capped)
		assert.Nil(t, failed)
	})
}

func TestMaxPower(t *testing.T) {
	maxFile := filepath.Join(raplDir, raplPrefixCPU+":0", maxPowerFileLongWindow)

	maxWatts, err := maxPower(maxFile)
	assert.NoError(t, err)
	assert.Equal(t, maxWattage, maxWatts)

	t.Run("name-does-not-exist", func(t *testing.T) {
		_, err := maxPower("madeupname")
		assert.Error(t, err)
	})
}

func TestCapZone(t *testing.T) {
	const percentage float64 = .50

	powercap := uint64(math.Ceil(float64(maxWattage) * percentage))
	limitFile := filepath.Join(raplDir, raplPrefixCPU+":0", powerLimitFileLongWindow)
	err := capZone(limitFile, powercap)
	assert.NoError(t, err)

	newCap, err := currentCap(limitFile)
	assert.NoError(t, err)
	assert.Equal(t, powercap, newCap)

	t.Run("name-does-not-exist", func(t *testing.T) {
		err := capZone("madeupname", powercap)
		assert.Error(t, err)
	})
}
