package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"vg/internal/app"
)

func main() {
	// Get executable directory for log file
	exePath, err := os.Executable()
	if err != nil {
		exePath = "."
	}
	logPath := filepath.Join(filepath.Dir(exePath), "vgo.log")

	// Open log file
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// If we can't open log file, try current directory
		logFile, err = os.OpenFile("vgo.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			// Last resort: continue without file logging
			log.SetFlags(log.Ltime | log.Lshortfile)
		}
	}

	if logFile != nil {
		defer logFile.Close()
		log.SetOutput(logFile)
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	}

	// Log startup
	log.Printf("=== VGO Editor starting at %s ===", time.Now().Format("2006-01-02 15:04:05"))

	// Recover from panics and log them
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC: %v", r)
			// Also write to a separate crash file
			crashPath := filepath.Join(filepath.Dir(exePath), "vgo_crash.log")
			if f, err := os.OpenFile(crashPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
				fmt.Fprintf(f, "[%s] PANIC: %v\n", time.Now().Format("2006-01-02 15:04:05"), r)
				f.Close()
			}
		}
	}()

	if err := app.Run(); err != nil {
		log.Printf("ERROR: %v", err)
		os.Exit(1)
	}

	log.Printf("=== VGO Editor exited normally ===")
}
