package agent

import (
	"strings"
	"testing"

	"github.com/yurika0211/luckyagent-runtime/provider"
	"github.com/yurika0211/luckyagent-runtime/tool"
)

func TestAppendLatestComputerObservationKeepsOnlyNewestFrame(t *testing.T) {
	messages := []provider.Message{
		{Role: "user", Content: "request"},
		{Role: "user", Content: "[Computer Observation old]", ContentParts: []provider.ContentPart{{Type: "image", Image: &provider.ImagePart{FilePath: "/tmp/old.png"}}}},
	}
	executed := []executedToolCall{{
		ToolCall: provider.ToolCall{Name: "computer_act"},
		Observations: []tool.Observation{
			{Kind: "image", FrameID: "frame-1", FilePath: "/tmp/frame-1.png", MimeType: "image/png"},
			{Kind: "image", FrameID: "frame-2", FilePath: "/tmp/frame-2.png", MimeType: "image/png"},
		},
	}}

	got := appendLatestComputerObservation(messages, executed)
	var observations []provider.Message
	for _, msg := range got {
		if isTransientComputerObservation(msg) {
			observations = append(observations, msg)
		}
	}
	if len(observations) != 1 {
		t.Fatalf("expected one transient observation, got %d: %#v", len(observations), got)
	}
	if observations[0].Content != "[Computer Observation frame-2]" {
		t.Fatalf("expected newest frame label, got %q", observations[0].Content)
	}
	if observations[0].ContentParts[0].Image.FilePath != "/tmp/frame-2.png" {
		t.Fatalf("expected newest frame path, got %#v", observations[0].ContentParts)
	}
}

func TestRepeatedObservationKeepsTheFrameAlreadyShown(t *testing.T) {
	messages := []provider.Message{
		{Role: "tool", Name: "computer_act", Content: "clicked"},
		{Role: "user", Content: "[Computer Observation frame-2]", ContentParts: []provider.ContentPart{{Type: "image", Image: &provider.ImagePart{FilePath: "/tmp/frame-2.png"}}}},
	}
	repeated := []executedToolCall{{
		ToolCall:     provider.ToolCall{Name: "computer_observe", Arguments: `{"format":"image"}`},
		Observations: []tool.Observation{{Kind: "image", FrameID: "frame-3", FilePath: "/tmp/frame-3.png", MimeType: "image/png"}},
	}}
	got := appendLatestComputerObservation(messages, repeated)
	var observations []provider.Message
	for _, msg := range got {
		if isTransientComputerObservation(msg) {
			observations = append(observations, msg)
		}
	}
	if len(observations) != 1 || observations[0].Content != "[Computer Observation frame-2]" {
		t.Fatalf("repeated observation replaced the current frame: %#v", got)
	}
	if !strings.Contains(got[len(got)-1].Content, "[computer observation reused]") {
		t.Fatalf("missing reuse guidance: %#v", got)
	}

	observed := []provider.Message{
		{Role: "tool", Name: "computer_observe", Content: "observed"},
		{Role: "user", Content: "[Computer Observation frame-2]", ContentParts: []provider.ContentPart{{Type: "image", Image: &provider.ImagePart{FilePath: "/tmp/frame-2.png"}}}},
	}
	switched := appendLatestComputerObservation(observed, []executedToolCall{{
		ToolCall:     provider.ToolCall{Name: "computer_observe", Arguments: `{"window":"Feishu"}`},
		Observations: []tool.Observation{{Kind: "image", FrameID: "frame-4", FilePath: "/tmp/frame-4.png", MimeType: "image/png"}},
	}})
	observations = nil
	for _, msg := range switched {
		if isTransientComputerObservation(msg) {
			observations = append(observations, msg)
		}
	}
	if len(observations) != 1 || observations[0].Content != "[Computer Observation frame-4]" {
		t.Fatalf("a different target did not replace the frame: %#v", switched)
	}
}

func TestRemoveTransientComputerObservationsPreservesUserImages(t *testing.T) {
	messages := []provider.Message{
		{Role: "user", Content: "[Computer Observation frame-1]", ContentParts: []provider.ContentPart{{Type: "image", Image: &provider.ImagePart{FilePath: "/tmp/frame.png"}}}},
		{Role: "user", Content: "uploaded image", ContentParts: []provider.ContentPart{{Type: "image", Image: &provider.ImagePart{FilePath: "/tmp/upload.png"}}}},
	}
	got := removeTransientComputerObservations(messages)
	if len(got) != 1 || got[0].Content != "uploaded image" {
		t.Fatalf("unexpected retained messages: %#v", got)
	}
}

func TestImageReadDoesNotEmitComputerObservation(t *testing.T) {
	for _, name := range []string{"image_read", "computer_observe", "computer_act"} {
		t.Run(name, func(t *testing.T) {
			events := make(chan ChatEvent, 1)
			emitChatObservationEvents(events, executedToolCall{
				ToolCall: provider.ToolCall{Name: name},
				Observations: []tool.Observation{{
					Kind: "image", FilePath: "/tmp/input.png", MimeType: "image/png",
				}},
			})
			if name == "image_read" {
				if len(events) != 0 {
					t.Fatal("reading an image must not send it back as a computer screenshot")
				}
				return
			}
			if len(events) != 1 {
				t.Fatal("computer screenshots must still be emitted")
			}
			event := <-events
			if event.Type != ChatEventObservation || event.Observation.FilePath != "/tmp/input.png" {
				t.Fatalf("unexpected computer event: %+v", event)
			}
		})
	}
}
