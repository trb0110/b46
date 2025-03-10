package logging

import (
	"b46/b46/models"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

var (
	logDir         = "trade-sessions"
	currentSession = ""
	mu             sync.Mutex
)

type Logger struct {
	file   *os.File
	writer *csv.Writer
	mutex  sync.Mutex
}

// Map of loggers (key: filename)
var Loggers = make(map[string]*Logger)
var GlobalMutex sync.Mutex

// **Initialize the logging session**
func InitLogSession() error {
	mu.Lock()
	defer mu.Unlock()

	// **Step 1: Ensure the `trade-sessions` directory exists**
	if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	// **Step 2: Find the latest session**
	sessionPath, err := getNextSession()
	if err != nil {
		return err
	}
	currentSession = sessionPath // Store session path

	log.Println("Logger initialized at session: 	", currentSession)
	return nil
}

// **Finds the next available session (e.g., session-0, session-1, session-2, ...)**
func getNextSession() (string, error) {
	files, err := os.ReadDir(logDir)
	if err != nil {
		return "", fmt.Errorf("failed to read log directory: %v", err)
	}

	highestSession := -1

	for _, file := range files {
		if file.IsDir() && strings.HasPrefix(file.Name(), "session-") {
			sessionNumStr := strings.TrimPrefix(file.Name(), "session-")
			sessionNum, err := strconv.Atoi(sessionNumStr)
			if err == nil && sessionNum > highestSession {
				highestSession = sessionNum
			}
		}
	}

	// **Create the next session directory**
	newSession := fmt.Sprintf("session-%d", highestSession+1)
	newSessionPath := filepath.Join(logDir, newSession)

	if err := os.MkdirAll(newSessionPath, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create session folder: %v", err)
	}

	return newSessionPath, nil
}

// InitLogger initializes a new logger for a given filename
func InitLogger(file string) error {

	filename := filepath.Join(currentSession, file)
	GlobalMutex.Lock()
	defer GlobalMutex.Unlock()

	// Check if logger already exists
	if _, exists := Loggers[filename]; exists {
		return nil
	}

	// Create and open the file
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file %q: %w", filename, err)
	}

	// Create a new Logger
	Loggers[filename] = &Logger{
		file:   f,
		writer: csv.NewWriter(f),
	}

	return nil
}

// PrintToLog writes a single record to the specified log file
func PrintToLog(file string, record []string) error {

	filename := filepath.Join(currentSession, file)
	GlobalMutex.Lock()
	logger, exists := Loggers[filename]
	GlobalMutex.Unlock()

	if !exists {
		return fmt.Errorf("logger for file %q is not initialized", filename)
	}

	logger.mutex.Lock()
	defer logger.mutex.Unlock()

	if err := logger.writer.Write(record); err != nil {
		return fmt.Errorf("error writing record to csv: %w", err)
	}

	logger.writer.Flush()
	return nil
}

// FlushLog ensures all buffered CSV data is written to the file
func FlushLog(file string) error {

	filename := filepath.Join(currentSession, file)
	GlobalMutex.Lock()
	logger, exists := Loggers[filename]
	GlobalMutex.Unlock()

	if !exists {
		return fmt.Errorf("logger for file %q is not initialized", filename)
	}

	logger.mutex.Lock()
	defer logger.mutex.Unlock()

	logger.writer.Flush()
	if err := logger.writer.Error(); err != nil {
		return fmt.Errorf("error flushing CSV writer: %w", err)
	}

	return nil
}

// CloseLogger closes a specified log file
func CloseLoggerFile(file string) error {

	filename := filepath.Join(currentSession, file)
	GlobalMutex.Lock()
	logger, exists := Loggers[filename]
	delete(Loggers, filename)
	GlobalMutex.Unlock()

	if !exists {
		return nil // Already closed or never opened
	}

	logger.mutex.Lock()
	defer logger.mutex.Unlock()

	// Flush before closing
	logger.writer.Flush()
	if err := logger.writer.Error(); err != nil {
		return fmt.Errorf("error flushing CSV writer: %w", err)
	}

	// Close the file
	if err := logger.file.Close(); err != nil {
		return fmt.Errorf("error closing file: %w", err)
	}

	return nil
}

// CloseAllLoggers closes all open log files
func CloseAllLoggers() {

	GlobalMutex.Lock()
	defer GlobalMutex.Unlock()

	for filename, logger := range Loggers {
		logger.mutex.Lock()

		// Flush before closing
		logger.writer.Flush()
		logger.file.Close()

		// Remove from map
		delete(Loggers, filename)

		logger.mutex.Unlock()
	}
}

// ClearFileLog empties the content of a given file.
func ClearFileLog(file string) error {

	filename := filepath.Join(currentSession, file)
	// Ensure file exists before attempting to open it.
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filename)
	} else if err != nil {
		return fmt.Errorf("failed to check file status: %w", err)
	}

	// Open the file with truncation flag.
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file for truncation: %w", err)
	}
	defer f.Close()

	// Ensure changes are flushed immediately.
	err = f.Sync()
	if err != nil {
		return fmt.Errorf("failed to flush file changes: %w", err)
	}

	return nil
}

// toString-like function for a slice of MemeInfo
func MemeInfosToString(infos []models.MemeInfo) string {
	var sb strings.Builder

	sb.WriteString("[\n")
	for i, info := range infos {
		sb.WriteString(fmt.Sprintf("  %d: %s\n", i, info.String()))
	}
	sb.WriteString("]")

	return sb.String()
}

// toString-like function for a slice of MemeInfo
func AnalysisInfosToString(infos []models.TokenAnalysis) string {
	var sb strings.Builder

	sb.WriteString("[\n")
	for i, info := range infos {
		sb.WriteString(fmt.Sprintf("  %d: %s\n", i, info.String()))
	}
	sb.WriteString("]")

	return sb.String()
}
