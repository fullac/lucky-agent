package agent

import "github.com/yurika0211/lucky-agent/agent/artifact"

// MediaReferences, ArtifactReferences, and StripMediaReferences stay on the
// agent package so published application builds can keep resolving channel
// attachments after the implementation moved into agent/artifact.

func MediaReferences(text string) []string {
	return artifact.MediaReferences(text)
}

func ArtifactReferences(text string) []string {
	return artifact.ArtifactReferences(text)
}

func StripMediaReferences(text string, references []string) string {
	return artifact.StripMediaReferences(text, references)
}
