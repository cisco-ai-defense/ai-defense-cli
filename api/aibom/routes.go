package aibom

import "fmt"

const (
	AIBoms        = "aibom/boms"
	AIBomAnalysis = "aibom/analysis"
)

func Boms() string {
	return AIBoms
}

func BomByID(analysisID string) string {
	return fmt.Sprintf("%s/%s", AIBoms, analysisID)
}

func BomComponents(analysisID string) string {
	return fmt.Sprintf("%s/%s/components", AIBoms, analysisID)
}

func BomsSummary() string {
	return fmt.Sprintf("%s:summary", AIBoms)
}
