package modelscan

import "fmt"

const (
	Scans       = "scans"
	ScanObjects = "objects"
)

func ScanByID(scanID string) string {
	return fmt.Sprintf("%s/%s", Scans, scanID)
}

func ObjectByID(scanID, objectID string) string {
	return fmt.Sprintf("%s/%s/%s", ScanByID(scanID), ScanObjects, objectID)
}
