package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TODO(rdelvalle): Add more thourough testing. Generate mock files
// that mimic the powercap subsystem and create test to operate on it.
func TestCapNode(t *testing.T) {

	err := capNode("/sys/devices/virtual/powercap/intel-rapl", 95)

	if err != nil {
		t.Fail()
	}
}

func TestMaxPower(t *testing.T) {
	const maxWattage uint64 = 1500000

	tmpfile, err := ioutil.TempFile("", maxPowerFileShortWindow)
	assert.NoError(t, err)

	defer os.Remove(tmpfile.Name())

	fmt.Println(tmpfile.Name())
	_, err = tmpfile.Write([]byte(strconv.FormatUint(maxWattage, 10)))
	assert.NoError(t, err)

	maxWatts, err := maxPower(tmpfile.Name())
	assert.NoError(t, err)
	assert.Equal(t, maxWattage, maxWatts)

	err = tmpfile.Close()
	assert.NoError(t, err)
}
