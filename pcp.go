package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
)

func PCP() {
	cmd := exec.Command("sh", "-c", "pmdumptext -m -l -o -d , -c config")
	time := time.Now().Format("200601021504")
	stdout, err := os.Create("./"+time+".txt")
	cmd.Stdout = stdout
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
