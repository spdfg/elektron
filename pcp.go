package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

func PCP() {
	cmd := exec.Command("sh", "-c", "pmdumptext -m -l -o -d , -c config")
	stdout, err := os.Create("./output.txt")
	cmd.Stdout = stdout
    fmt.Println("PCP started: ", stdout)

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