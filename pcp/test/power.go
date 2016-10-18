package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"strconv"
	"math"
	"container/ring"
	"sort"
)

type Victim struct {
	Watts float64
	Host string
}

type VictimSorter []Victim

func (slice VictimSorter) Len() int {
	return len(slice)
}

func (slice VictimSorter) Less(i, j int) bool {
	return slice[i].Watts >= slice[j].Watts
}

func (slice VictimSorter) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

var RAPLUnits = math.Pow(2, -32)

func mean(values *ring.Ring) float64 {

	total := 0.0
	count := 0.0

	values.Do(func(x interface{}){
		if val, ok := x.(float64); ok { //Add it if we can get a float
			total += val
			count++
		}
	})

	if count == 0.0 {
		return 0.0
	}


	count /= 2

	return (total/count)
}

//func median(values *ring.Ring) {

//}


func main() {

	prefix := "test"
	logging := new(bool)
	*logging = true
	const pcpCommand string = "pmdumptext -m -l -f '' -t 1.0 -d , -c config"
	cmd := exec.Command("sh", "-c", pcpCommand)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	startTime := time.Now().Format("20060102150405")

	logFile, err := os.Create("./" + prefix + startTime + ".pcplog")
	if err != nil {
		log.Fatal(err)
	}

	defer logFile.Close()

	pipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	//cmd.Stdout = stdout

	scanner := bufio.NewScanner(pipe)

	go func(logging *bool) {
		// Get names of the columns
		scanner.Scan()

		// Write to logfile
		logFile.WriteString(scanner.Text() + "\n")

		headers := strings.Split(scanner.Text(), ",")

		powerIndexes := make([]int, 0, 0)
		powerAverage := make(map[string]*ring.Ring)
		indexToHost := make(map[int]string)

		for i, hostMetric := range headers {
			split := strings.Split(hostMetric, ":")
			fmt.Printf("%d Host %s: Metric: %s\n", i, split[0], split[1])

			if strings.Contains(split[1], "RAPL_ENERGY_PKG") {
				fmt.Println("Index: ", i)
				powerIndexes = append(powerIndexes, i)
				indexToHost[i] = split[0]
				powerAverage[split[0]] = ring.New(10) // Two PKS per node, 10 = 5 seconds tracking
			}
		}

		// Throw away first set of results
		scanner.Scan()

		seconds := 0
		for scanner.Scan() {

			if *logging {
				log.Println("Logging PCP...")
				split := strings.Split(scanner.Text(), ",")
				logFile.WriteString(scanner.Text() + "\n")


				totalPower := 0.0
				for _,powerIndex := range powerIndexes {
					power, _ := strconv.ParseFloat(split[powerIndex], 64)

					host := indexToHost[powerIndex]

					powerAverage[host].Value = power
					powerAverage[host] = powerAverage[host].Next()

					log.Printf("Host: %s, Index: %d, Power: %f", indexToHost[powerIndex], powerIndex, (power * RAPLUnits))

					totalPower += power
				}

				log.Println("Total power: ", totalPower * RAPLUnits)

				victims := make([]Victim, 8, 8)

				// TODO: Just keep track of the largest to reduce fron nlogn to n
				for name,ring := range powerAverage {
					victims = append(victims, Victim{mean(ring), name})
					//log.Printf("host: %s, Avg: %f", name, mean(ring) * RAPLUnits)
				}
				sort.Sort(VictimSorter(victims))
				log.Printf("Current Victim %s Avg. Wattage: %f", victims[0].Host, victims[0].Watts * RAPLUnits)
			}

			/*
				fmt.Printf("Second: %d\n", seconds)
				for i, val := range strings.Split(scanner.Text(), ",") {
					fmt.Printf("host metric: %s val: %s\n", headers[i], val)
				}*/

			seconds++

			// fmt.Println("--------------------------------")
		}
	}(logging)

	log.Println("PCP logging started")

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}

	/*
		pgid, err := syscall.Getpgid(cmd.Process.Pid)

		select {
		case <-quit:
			log.Println("Stopping PCP logging in 5 seconds")
			time.Sleep(5 * time.Second)

		// http://stackoverflow.com/questions/22470193/why-wont-go-kill-a-child-process-correctly
		// kill process and all children processes
			syscall.Kill(-pgid, 15)
			return
		}*/
}
