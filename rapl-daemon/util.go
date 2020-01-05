package main

import (
	"fmt"
	"io/ioutil"
	"math"
	"path/filepath"
	"strconv"
	"strings"
)

const raplPrefixCPU = "intel-rapl"

// constraint_0 is usually the longer window while constraint_1 is usually the longer window
const maxPowerFileShortWindow = "constraint_0_max_power_uw"
const powerLimitFileShortWindow = "constraint_0_power_limit_uw"

// capNode uses pseudo files made available by the Linux kernel
// in order to capNode CPU power. More information is available at:
// https://www.kernel.org/doc/html/latest/power/powercap/powercap.html
func capNode(base string, percentage int) error {

	if percentage <= 0 || percentage > 100 {
		return fmt.Errorf("cap percentage must be between 0 (non-inclusive) and 100 (inclusive): %d", percentage)
	}

	files, err := ioutil.ReadDir(base)
	if err != nil {
		return err
	}

	for _, file := range files {

		fields := strings.Split(file.Name(), ":")

		// Fields should be in the form intel-rapl:X where X is the power zone
		// We ignore sub-zones which follow the form intel-rapl:X:Y
		if len(fields) != 2 {
			continue
		}

		if fields[0] == raplPrefixCPU {
			maxPower, err := maxPower(filepath.Join(base, file.Name(), maxPowerFileShortWindow))
			if err != nil {
				fmt.Println("unable to retreive max power for zone ", err)
				continue
			}

			// We use floats to mitigate the possibility of an integer overflows.
			powercap := uint64(math.Ceil(float64(maxPower) * (float64(percentage) / 100)))

			err = capZone(filepath.Join(base, file.Name(), powerLimitFileShortWindow), powercap)
			if err != nil {
				fmt.Println("unable to write powercap value: ", err)
				continue
			}
			fmt.Println(file.Name(), ": ", int(maxPower), ", ", int(powercap))
		}
	}

	return nil
}

// maxPower returns the value in float of the maximum watts a power zone
// can use.
func maxPower(zone string) (uint64, error) {
	maxPower, err := ioutil.ReadFile(zone)
	if err != nil {
		return 0.0, err
	}

	maxPoweruW, err := strconv.ParseUint(strings.TrimSpace(string(maxPower)), 10, 64)
	if err != nil {
		return 0.0, err
	}

	return maxPoweruW, nil

}

// capZone caps a power zone to a specific amount of watts specified by value
func capZone(zone string, value uint64) error {
	err := ioutil.WriteFile(zone, []byte(strconv.FormatUint(value, 10)), 0644)
	if err != nil {
		return err
	}
	return nil
}
