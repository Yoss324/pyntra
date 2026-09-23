package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)
type ResultStorage interface {
	SaveResult(executionID string, toolName string, result string) error
	GetResult(executionID string) (string, error)
	GetResultPage(executionID string, page int, limit int) (*ResultPage, error)
	SearchResult(executionID string, keyword string, useRegex bool) ([]string, error)
	FilterResult(executionID string, filter string, useRegex bool) ([]string, error)
	GetResultMetadata(executionID string) (*ResultMetadata, error)
	GetResultPath(executionID string) string
	DeleteResult(executionID string) error
}
type ResultPage struct {
	Lines []string `json:"lines"`
	Page int `json:"page"`
	Limit int `json:"limit"`
	TotalLines int `json:"total_lines"`
	TotalPages int `json:"total_pages"`
}
type ResultMetadata struct {
	ExecutionID string `json:"execution_id"`
	ToolName string `json:"tool_name"`
	TotalSize int `json:"total_size"`
	TotalLines int `json:"total_lines"`
	CreatedAt time.Time `json:"created_at"`
}
type FileResultStorage struct {
	baseDir string
	logger *zap.Logger
	mu sync.RWMutex
}
func NewFileResultStorage(baseDir string, logger *zap.Logger) (*FileResultStorage, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("createdirectoryfailed: %w", err)
	}

	return &FileResultStorage{
		baseDir: baseDir,
		logger: logger,
	}, nil
}
func (s *FileResultStorage) getResultPath(executionID string) string {
	return filepath.Join(s.baseDir, executionID+".txt")
}
func (s *FileResultStorage) getMetadataPath(executionID string) string {
	return filepath.Join(s.baseDir, executionID+".meta.json")
}
func (s *FileResultStorage) SaveResult(executionID string, toolName string, result string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	resultPath := s.getResultPath(executionID)
	if err := os.WriteFile(resultPath, []byte(result), 0644); err != nil {
		return fmt.Errorf("saveresultFilesfailed: %w", err)
	}
	lines := strings.Split(result, "\n")
	metadata := &ResultMetadata{
		ExecutionID: executionID,
		ToolName: toolName,
		TotalSize: len(result),
		TotalLines: len(lines),
		CreatedAt: time.Now(),
	}
	metadataPath := s.getMetadataPath(executionID)
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("metadatafailed: %w", err)
	}

	if err := os.WriteFile(metadataPath, metadataJSON, 0644); err != nil {
		return fmt.Errorf("savemetadataFilesfailed: %w", err)
	}

	s.logger.Info("savetoolresult",
		zap.String("executionID", executionID),
		zap.String("toolName", toolName),
		zap.Int("size", len(result)),
		zap.Int("lines", len(lines)),
	)

	return nil
}
func (s *FileResultStorage) GetResult(executionID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resultPath := s.getResultPath(executionID)
	data, err := os.ReadFile(resultPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("resultdoes not exist: %s", executionID)
		}
		return "", fmt.Errorf("resultFilesfailed: %w", err)
	}

	return string(data), nil
}
func (s *FileResultStorage) GetResultMetadata(executionID string) (*ResultMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metadataPath := s.getMetadataPath(executionID)
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("resultdoes not exist: %s", executionID)
		}
		return nil, fmt.Errorf("metadataFilesfailed: %w", err)
	}

	var metadata ResultMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("parsemetadatafailed: %w", err)
	}

	return &metadata, nil
}
func (s *FileResultStorage) GetResultPage(executionID string, page int, limit int) (*ResultPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result, err := s.GetResult(executionID)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(result, "\n")
	totalLines := len(lines)
	totalPages := (totalLines + limit - 1) / limit
	if page < 1 {
		page = 1
	}
	if page > totalPages && totalPages > 0 {
		page = totalPages
	}
	start := (page - 1) * limit
	end := start + limit
	if end > totalLines {
		end = totalLines
	}
	var pageLines []string
	if start < totalLines {
		pageLines = lines[start:end]
	} else {
		pageLines = []string{}
	}

	return &ResultPage{
		Lines: pageLines,
		Page: page,
		Limit: limit,
		TotalLines: totalLines,
		TotalPages: totalPages,
	}, nil
}
func (s *FileResultStorage) SearchResult(executionID string, keyword string, useRegex bool) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result, err := s.GetResult(executionID)
	if err != nil {
		return nil, err
	}
	var regex *regexp.Regexp
	if useRegex {
		compiledRegex, err := regexp.Compile(keyword)
		if err != nil {
			return nil, fmt.Errorf("invalid: %w", err)
		}
		regex = compiledRegex
	}
	lines := strings.Split(result, "\n")
	var matchedLines []string

	for _, line := range lines {
		var matched bool
		if useRegex {
			matched = regex.MatchString(line)
		} else {
			matched = strings.Contains(line, keyword)
		}

		if matched {
			matchedLines = append(matchedLines, line)
		}
	}

	return matchedLines, nil
}
func (s *FileResultStorage) FilterResult(executionID string, filter string, useRegex bool) ([]string, error) {
	return s.SearchResult(executionID, filter, useRegex)
}
func (s *FileResultStorage) GetResultPath(executionID string) string {
	return s.getResultPath(executionID)
}
func (s *FileResultStorage) DeleteResult(executionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	resultPath := s.getResultPath(executionID)
	metadataPath := s.getMetadataPath(executionID)
	if err := os.Remove(resultPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleteresultFilesfailed: %w", err)
	}
	if err := os.Remove(metadataPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deletemetadataFilesfailed: %w", err)
	}

	s.logger.Info("deletetoolresult",
		zap.String("executionID", executionID),
	)

	return nil
}
