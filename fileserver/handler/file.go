package handler

import (
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const filename = "file.txt"

type LineNotFoundError struct{}

func (e LineNotFoundError) Error() string {
	return "Line not found error"
}

func file(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL.Path)
	var lineNum int
	if strings.HasPrefix(r.URL.Path, "/file/") {
		var err error
		lineNumStr := r.URL.Path[len("/file/"):]
		lineNum, err = strconv.Atoi(lineNumStr)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}

	switch r.Method {
	case http.MethodGet:
		if lineNum == 0 {
			if file, err := getFile(); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					w.WriteHeader(http.StatusNotFound)
				} else {
					w.WriteHeader(http.StatusInternalServerError)
				}
			} else {
				w.WriteHeader(http.StatusOK)
				if _, err := w.Write(file); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
				}
			}
		} else {
			if line, err := getLine(lineNum); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					w.WriteHeader(http.StatusNotFound)
				} else if errors.Is(err, LineNotFoundError{}) {
					w.WriteHeader(http.StatusOK)
				} else {
					w.WriteHeader(http.StatusInternalServerError)
				}
			} else {
				w.WriteHeader(http.StatusOK)
				w.Write(line)
			}
		}

	case http.MethodPost:
		content, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		if file, err := createFile(content); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusCreated)
			if _, err := w.Write(file); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}

	case http.MethodPut:
		newLine, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		if file, err := replaceLine(lineNum, newLine); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				w.WriteHeader(http.StatusNotFound)
			} else if errors.Is(err, LineNotFoundError{}) {
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(file)
		}

	case http.MethodDelete:
		if file, err := deleteLine(lineNum); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				w.WriteHeader(http.StatusNotFound)
			} else if errors.Is(err, LineNotFoundError{}) {
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(file)
		}

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func getFile() ([]byte, error) {
	if file, err := os.ReadFile(filename); err != nil {
		return nil, err
	} else {
		return file, nil
	}
}

func createFile(content []byte) ([]byte, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if _, err := file.Write(content); err != nil {
		return nil, err
	}

	return content, nil
}

func getLine(lineNum int) ([]byte, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")

	if lineNum > 0 && lineNum <= len(lines) {
		return []byte(lines[lineNum-1]), nil
	} else {
		return nil, LineNotFoundError{}
	}
}

func replaceLine(lineNum int, newLine []byte) ([]byte, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")

	if lineNum > 0 && lineNum <= len(lines) {
		lines[lineNum-1] = string(newLine)
	} else {
		return nil, LineNotFoundError{}
	}

	updatedFile := strings.Join(lines, "\n")
	if err := os.WriteFile(filename, []byte(updatedFile), 0644); err != nil {
		return nil, err
	}

	return []byte(updatedFile), nil
}

func deleteLine(lineNum int) ([]byte, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	if lineNum > 0 && lineNum <= len(lines) {
		lines = append(lines[:lineNum-1], lines[lineNum:]...)
	} else {
		return nil, LineNotFoundError{}
	}

	updatedFile := strings.Join(lines, "\n")
	if err := os.WriteFile(filename, []byte(updatedFile), 0644); err != nil {
		return nil, err
	}

	return []byte(updatedFile), nil
}
