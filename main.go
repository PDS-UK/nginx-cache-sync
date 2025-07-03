package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
    log.SetPrefix("[nginx-cache-sync] ")
    log.SetOutput(os.Stdout)

	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbName := os.Getenv("DB_NAME")

	if dbUser == "" || dbPass == "" || dbHost == "" || dbName == "" {
		log.Fatal("Missing DB_USER, DB_PASSWORD, DB_HOST, or DB_NAME")
	}

	cachePath := getEnv("CACHE_PATH", "/var/run/nginx-cache/")
	stateFile := getEnv("STATE_FILE", "/var/run/nginx-cache-sync.last")
	checkIntervalStr := getEnv("CHECK_INTERVAL", "60")

	checkInterval, err := time.ParseDuration(checkIntervalStr + "s")
	if err != nil {
		log.Fatalf("Invalid CHECK_INTERVAL: %v", err)
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s", dbUser, dbPass, dbHost, dbName)

	for {
		func() {
			db, err := sql.Open("mysql", dsn)
			if err != nil {
				log.Printf("DB connection error: %v", err)
				return
			}
			defer db.Close()

			var remoteTime string
			err = db.QueryRow(`SELECT option_value FROM wp_options WHERE option_name = 'nginx_cache_last_cleared'`).Scan(&remoteTime)
			if err != nil {
				log.Printf("Query error: %v", err)
				return
			}

			localTime := readLocalTimestamp(stateFile)
			if remoteTime != localTime {
				log.Printf("Detected cache clear signal — clearing cache at %s", cachePath)
				clearCache(cachePath)
				writeLocalTimestamp(stateFile, remoteTime)
			} else {
				log.Println("No cache change detected")
			}
		}()

		time.Sleep(checkInterval)
	}
}

func readLocalTimestamp(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func writeLocalTimestamp(path, timestamp string) {
	_ = os.WriteFile(path, []byte(timestamp), 0644)
}

func clearCache(cachePath string) {
	cmd := exec.Command("find", cachePath, "-type", "f", "-delete")
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("Error clearing cache: %v\nOutput: %s", err, output)
	} else {
		log.Println("Cache cleared successfully")
	}
}