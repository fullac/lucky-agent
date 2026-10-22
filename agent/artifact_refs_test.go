package agent

import "testing"

func TestMediaReferenceAliasesMatchArtifactPackage(t *testing.T) {
	text := "saved\nMEDIA:/tmp/report.md\nsee ~/.luckyagent/workspace/report.md"
	references := MediaReferences(text)
	if len(references) != 1 || references[0] != "/tmp/report.md" {
		t.Fatalf("MediaReferences() = %#v", references)
	}
	artifacts := ArtifactReferences(text)
	if len(artifacts) != 1 {
		t.Fatalf("ArtifactReferences() = %#v", artifacts)
	}
	stripped := StripMediaReferences(text, references)
	if stripped == text || containsMediaLine(stripped) {
		t.Fatalf("StripMediaReferences() = %q", stripped)
	}
}

func containsMediaLine(text string) bool {
	return len(MediaReferences(text)) != 0
}
