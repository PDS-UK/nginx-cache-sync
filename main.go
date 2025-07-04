package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbName := os.Getenv("DB_NAME")

	if dbUser == "" || dbPass == "" || dbHost == "" || dbName == "" {
		logJSON("Missing DB_USER, DB_PASSWORD, DB_HOST, or DB_NAME", nil)
		os.Exit(1)
	}

	cachePath := getEnv("NGINX_CACHE_PATH", "/var/run/nginx-cache/")
	stateFile := getEnv("NGINX_CACHE_STATE_FILE", "/var/run/nginx-cache-sync.last")
	checkIntervalStr := getEnv("NGINX_CACHE_CHECK_INTERVAL", "60")

	checkInterval, err := time.ParseDuration(checkIntervalStr + "s")
	if err != nil {
		logJSON("Invalid NGINX_CACHE_CHECK_INTERVAL", map[string]string{"error": err.Error()})
		os.Exit(1)
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s", dbUser, dbPass, dbHost, dbName)

	for {
		func() {
			db, err := sql.Open("mysql", dsn)
			if err != nil {
				logJSON("DB connection error", map[string]string{"error": err.Error()})
				return
			}
			defer db.Close()

			var remoteTime string
			err = db.QueryRow(`SELECT option_value FROM wp_options WHERE option_name = 'nginx_cache_last_cleared'`).Scan(&remoteTime)
			if err != nil {
				logJSON("Query error", map[string]string{"error": err.Error()})
				return
			}

			localTime := readLocalTimestamp(stateFile)
			if remoteTime != localTime {
				logJSON("Detected cache clear signal — clearing cache", map[string]string{"path": cachePath})
				clearCache(cachePath)
				writeLocalTimestamp(stateFile, remoteTime)
			} else {
				logJSON("No cache change detected", nil)
			}
		}()

		time.Sleep(checkInterval)
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
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
		logJSON("Error clearing cache", map[string]string{
			"path":  cachePath,
			"error": err.Error(),
			"output": strings.TrimSpace(string(output)),
		})
	} else {
		logJSON("Cache cleared successfully", map[string]string{"path": cachePath})
	}
}

func logJSON(msg string, fields map[string]string) {
	entry := map[string]string{
		"msg": "[nginx-cache-sync] " + msg,
	}
	for k, v := range fields {
		entry[k] = v
	}
	_ = json.NewEncoder(os.Stdout).Encode(entry)
}