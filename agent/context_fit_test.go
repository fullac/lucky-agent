package agent

import (
	"strings"
	"testing"

	"github.com/yurika0211/luckyagent-runtime/contextx"
	"github.com/yurika0211/luckyagent-runtime/provider"
)

func TestFitContextWindowTrimsTextHistoryWhenOneMessageHasContentParts(t *testing.T) {
	a, _ := contextTestAgent(t)
	a.contextWin = contextx.NewContextWindow(contextx.WindowConfig{
		MaxTokens:      200,
		ReservedTokens: 40,
		Strategy:       contextx.TrimOldest,
	})
	history := make([]provider.Message, 0, 62)
	history = append(history, provider.Message{Role: "system", Content: "keep system"})
	for i := 0; i < 60; i++ {
		history = append(history, provider.Message{
			Role:    "user",
			Content: strings.Repeat("old history line ", 40),
		})
	}
	history = append(history, provider.Message{
		Role:    "user",
		Content: "CURRENT_IMAGE",
		ContentParts: []provider.ContentPart{{
			Type:  "image",
			Image: &provider.ImagePart{FilePath: "/tmp/upload.png", Detail: "low"},
		}},
	})

	control := append([]provider.Message(nil), history[:len(history)-1]...)
	control = append(control, provider.Message{Role: "user", Content: "CURRENT_REQUEST"})
	trimmedControl := a.fitContextWindow(control)
	if len(trimmedControl) >= len(control) {
		t.Fatalf("text control was not trimmed: %d", len(trimmedControl))
	}

	got := a.fitContextWindow(history)
	if len(got) >= len(history) {
		t.Fatalf("multipart history was not trimmed: kept %d/%d", len(got), len(history))
	}
	last := got[len(got)-1]
	if last.Content != "CURRENT_IMAGE" || len(last.ContentParts) != 1 || last.ContentParts[0].Image == nil || last.ContentParts[0].Image.FilePath != "/tmp/upload.png" {
		t.Fatalf("multipart tail lost: %#v", last)
	}
	if !messageTextContains(got, "keep system") {
		t.Fatal("critical system message was dropped")
	}
}

func messageTextContains(messages []provider.Message, part string) bool {
	for _, msg := range messages {
		if strings.Contains(msg.Content, part) {
			return true
		}
	}
	return false
}

func TestFitContextWindowKeepsImageBetweenTrimmedTurns(t *testing.T) {
	a, _ := contextTestAgent(t)
	a.contextWin = contextx.NewContextWindow(contextx.WindowConfig{
		MaxTokens:      180,
		ReservedTokens: 20,
		Strategy:       contextx.TrimOldest,
	})
	messages := []provider.Message{
		{Role: "system", Content: "keep system"},
		{Role: "user", Content: strings.Repeat("stale ", 400)},
		{Role: "user", Content: "look", ContentParts: []provider.ContentPart{{Type: "image", Image: &provider.ImagePart{FilePath: "/tmp/mid.png"}}}},
		{Role: "assistant", Content: strings.Repeat("old reply ", 400)},
		{Role: "user", Content: "CURRENT_REQUEST"},
	}
	got := a.fitContextWindow(messages)
	if len(got) >= len(messages) {
		t.Fatalf("expected trim, kept %d", len(got))
	}
	var sawImage bool
	var imageAt int
	var requestAt = -1
	for i, msg := range got {
		if len(msg.ContentParts) > 0 {
			sawImage = true
			imageAt = i
			if msg.ContentParts[0].Image.FilePath != "/tmp/mid.png" {
				t.Fatalf("image path = %#v", msg.ContentParts)
			}
		}
		if msg.Content == "CURRENT_REQUEST" {
			requestAt = i
		}
	}
	if !sawImage || requestAt < 0 || imageAt > requestAt {
		t.Fatalf("image/request order lost: image=%v at %d request at %d messages=%#v", sawImage, imageAt, requestAt, got)
	}
}
