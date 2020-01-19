package main

import (
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const raplPrefixCPU = "intel-rapl"

// constraint_0 is usually the longer window while constraint_1 is usually the shorter window
const maxPowerFileLongWindow = "constraint_0_max_power_uw"
const powerLimitFileLongWindow = "constraint_0_power_limit_uw"

// capNode uses pseudo files made available by the Linux kernel
// in order to capNode CPU power. More information is available at:
// https://www.kernel.org/doc/html/latest/power/powercap/powercap.html
func capNode(base string, percentage int) ([]string, []string, error) {

	if percentage <= 0 || percentage > 100 {
		return nil, nil, fmt.Errorf("cap percentage must be between 0 (non-inclusive) and 100 (inclusive): %d", percentage)
	}

	files, err := ioutil.ReadDir(base)
	if err != nil {
		return nil, nil, err
	}

	var capped, failed []string
	for _, file := range files {
		fields := strings.Split(file.Name(), ":")

		// Fields should be in the form intel-rapl:X where X is the power zone
		// We ignore sub-zones which follow the form intel-rapl:X:Y
		if len(fields) != 2 {
			continue
		}

		if fields[0] == raplPrefixCPU {
			maxPower, err := maxPower(filepath.Join(base, file.Name(), maxPowerFileLongWindow))
			if err != nil {
				failed = append(failed, file.Name())
				fmt.Println("unable to retreive max power for zone ", err)
				continue
			}

			// We use floats to mitigate the possibility of an integer overflow.
			powercap := uint64(math.Ceil(float64(maxPower) * (float64(percentage) / 100)))

			if err := capZone(filepath.Join(base, file.Name(), powerLimitFileLongWindow), powercap); err != nil {
				failed = append(failed, file.Name())
				fmt.Println("unable to write powercap value: ", err)
				continue
			}
			capped = append(capped, file.Name())
		}
	}

	if len(failed) > 0 {
		return capped, failed, fmt.Errorf("some zones were not able to be powercapped")
	}

	return capped, nil, nil
}

// maxPower returns the value in float of the maximum watts a power zone can use.
func maxPower(maxFile string) (uint64, error) {
	maxPower, err := ioutil.ReadFile(maxFile)
	if err != nil {
		return 0, err
	}

	maxPoweruW, err := strconv.ParseUint(strings.TrimSpace(string(maxPower)), 10, 64)
	if err != nil {
		return 0, err
	}

	return maxPoweruW, nil
}

// capZone caps a power zone to a specific amount of watts specified by value
func capZone(limitFile string, value uint64) error {
	if _, err := os.Stat(limitFile); os.IsNotExist(err) {
		return err
	}

	err := ioutil.WriteFile(limitFile, []byte(strconv.FormatUint(value, 10)), 0644)
	if err != nil {
		return err
	}
	return nil
}

func currentCap(limit string) (uint64, error) {
	powercap, err := ioutil.ReadFile(limit)
	if err != nil {
		return 0, err
	}

	powercapuW, err := strconv.ParseUint(strings.TrimSpace(string(powercap)), 10, 64)
	if err != nil {
		return 0, err
	}

	return powercapuW, nil
}
