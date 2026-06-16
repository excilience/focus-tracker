package main

import (
	"encoding/json"
	"os"
)

const dataDir = "Data"
const sessionsFile = dataDir + "/sessions.json"

func ensureDataDir() error {
	return os.MkdirAll(dataDir, 0755)
}

func writeJsonAtomic(path string, data []byte) error {
	tmp := path + ".tmp"

	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)

}
func loadSessions() ([]Session, error) {
	var allSessions []Session

	data, err := os.ReadFile(sessionsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []Session{}, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return []Session{}, nil
	}
	if err := json.Unmarshal(data, &allSessions); err != nil {
		return []Session{}, err
	}
	return allSessions, nil
}

func saveSessions(sessions []Session) error {

	if err := ensureDataDir(); err != nil {
		return err
	}

	jsonData, err := json.MarshalIndent(sessions, "", "\t")
	if err != nil {
		return err
	}

	return os.WriteFile(sessionsFile, jsonData, 0644)
}
