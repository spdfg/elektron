package main

import "testing"

// TODO(rdelvalle): Add more thourough testing. Generate mock files
// that mimic the powercap subsystem and create test to operate on it.
func TestCap(t *testing.T) {

	err := capNode("/sys/devices/virtual/powercap/intel-rapl", 95)

	if err != nil {
		t.Fail()
	}
}
