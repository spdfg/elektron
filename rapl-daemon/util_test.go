package main

import (
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TODO(rdelvalle): Add more thourough testing. Generate mock files
// that mimic the powercap subsystem and create test to operate on it.
func TestCapNode(t *testing.T) {

	RAPLdir, err := ioutil.TempDir("", "intel-rapl")
	assert.NoError(t, err)
	defer os.RemoveAll(RAPLdir)

	zonePath := filepath.Join(RAPLdir, raplPrefixCPU+":0")
	err = os.Mkdir(zonePath, 755)
	assert.NoError(t, err)
	err = ioutil.WriteFile(filepath.Join(zonePath, maxPowerFileShortWindow), []byte("1500000"), 0444)
	assert.NoError(t, err)
	err = ioutil.WriteFile(filepath.Join(zonePath, powerLimitFileShortWindow), []byte("1500000"), 0644)
	assert.NoError(t, err)

	err = capNode(RAPLdir, 95)
	assert.NoError(t, err)
}

func TestMaxPower(t *testing.T) {
	const maxWattage uint64 = 1500000

	tmpfile, err := ioutil.TempFile("", maxPowerFileShortWindow)
	assert.NoError(t, err)

	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.Write([]byte(strconv.FormatUint(maxWattage, 10)))
	assert.NoError(t, err)

	maxWatts, err := maxPower(tmpfile.Name())
	assert.NoError(t, err)
	assert.Equal(t, maxWattage, maxWatts)

	err = tmpfile.Close()
	assert.NoError(t, err)
}

func TestCapZone(t *testing.T) {
	const maxPower float64 = 1500000
	const percentage float64 = .50

	tmpfile, err := ioutil.TempFile("", powerLimitFileShortWindow)
	assert.NoError(t, err)

	defer os.Remove(tmpfile.Name())

	powercap := uint64(math.Ceil(maxPower * percentage))

	err = capZone(tmpfile.Name(), powercap)
	assert.NoError(t, err)

	newCapBytes, err := ioutil.ReadFile(tmpfile.Name())
	assert.NoError(t, err)

	newCap, err := strconv.ParseUint(strings.TrimSpace(string(newCapBytes)), 10, 64)
	assert.NoError(t, err)
	assert.Equal(t, powercap, newCap)

	err = tmpfile.Close()
	assert.NoError(t, err)
}
