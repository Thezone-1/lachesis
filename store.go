package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Sample is one observation of the foreground window.
type Sample struct {
	TS    time.Time `json:"ts"`
	App   string    `json:"app"`
	Title string    `json:"title"`
	Idle  bool      `json:"idle"`
}

func dataDir() string {
	return filepath.Join(baseDir(), "data")
}

func dayFile(day string) string {
	return filepath.Join(dataDir(), day+".jsonl")
}

// appendSample appends one sample to the JSONL file for its local date.
func appendSample(s Sample) error {
	if err := os.MkdirAll(dataDir(), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(dayFile(s.TS.Format("2006-01-02")), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	line, err := json.Marshal(s)
	if err != nil {
		return err
	}
	_, err = f.Write(append(line, '\n'))
	return err
}

// readDay loads all samples recorded on a local date (YYYY-MM-DD).
func readDay(day string) ([]Sample, error) {
	f, err := os.Open(dayFile(day))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var samples []Sample
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		var s Sample
		if err := json.Unmarshal(sc.Bytes(), &s); err != nil {
			continue // skip torn/corrupt lines
		}
		samples = append(samples, s)
	}
	return samples, sc.Err()
}
