package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"skill-arena/internal/workers"
)

func main() {
	backupPath := flag.String("backup", "", "path to a Skill Arena backup directory")
	reportPath := flag.String("report", "", "optional path to write the recovery report JSON")
	flag.Parse()

	if *backupPath == "" {
		log.Fatal("-backup is required")
	}

	report, err := workers.ValidateRecovery(context.Background(), *backupPath)
	if err != nil {
		log.Fatalf("recovery validation failed: %v", err)
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Fatalf("failed to encode report: %v", err)
	}
	if *reportPath != "" {
		if err := os.WriteFile(*reportPath, data, 0o644); err != nil {
			log.Fatalf("failed to write report: %v", err)
		}
	}
	fmt.Println(string(data))
	if !report.Passed {
		os.Exit(1)
	}
}
