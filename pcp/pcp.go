package main

import (
	"fmt"
	"log"
	"os/exec"
	"bufio"
	"strings"
)

func main() {
	const pcpCommand string = "pmdumptext -m -l -f '' -t 1.0 -d , -c config" // We always want the most granular
	cmd := exec.Command("sh", "-c", pcpCommand)
//	time := time.Now().Format("200601021504")

//	stdout, err := os.Create("./"+time+".txt")
	pipe, err := cmd.StdoutPipe()

	//cmd.Stdout = stdout

	scanner := bufio.NewScanner(pipe)

	go func() {
		// Get names of the columns
		scanner.Scan()

		headers := strings.Split(scanner.Text(), ",")

		for _, hostMetric := range headers {
			split := strings.Split(hostMetric, ":")
			fmt.Printf("Host %s: Metric: %s\n", split[0], split[1])
		}

		// Throw away first set of results
		scanner.Scan()


		seconds := 0
		for scanner.Scan() {
			fmt.Println("Second ", seconds , " val: ",  strings.Split(scanner.Text(), ","))
			seconds++
		}
	}()

    fmt.Println("PCP started: ")

	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
}
