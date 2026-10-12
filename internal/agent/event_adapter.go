package agent

import (
	"github.com/yurika0211/luckyagent/internal/provider"
	"github.com/yurika0211/luckyagent/internal/turn"
)

// ToTurnChatEvent converts the legacy in-process event into the neutral DTO
// used by gateway adapters. It is intentionally explicit so provider usage
// accounting and local computer paths do not leak through the boundary.
func ToTurnChatEvent(event ChatEvent) turn.ChatEvent {
	return turn.ChatEvent{
		Type:        turn.EventType(event.Type),
		Content:     event.Content,
		TaskID:      event.TaskID,
		Name:        event.Name,
		Args:        event.Args,
		Result:      event.Result,
		Round:       event.Round,
		Observation: toTurnObservation(event.Observation),
		Approval:    toTurnApproval(event.Approval),
		Compact:     toTurnCompact(event.Compact),
		Usage:       provider.ToTurnTokenUsage(event.Usage),
		CreatedAt:   event.CreatedAt,
		Err:         event.Err,
	}
}

func toTurnObservation(value *ObservationEvent) *turn.ObservationEvent {
	if value == nil {
		return nil
	}
	return &turn.ObservationEvent{
		FrameID:      value.FrameID,
		FilePath:     value.FilePath,
		MimeType:     value.MimeType,
		Width:        value.Width,
		Height:       value.Height,
		ScaleFactor:  value.ScaleFactor,
		DisplayID:    value.DisplayID,
		ActiveWindow: value.ActiveWindow,
	}
}

func toTurnApproval(value *ApprovalEvent) *turn.ApprovalEvent {
	if value == nil {
		return nil
	}
	fields := make(map[string]string, len(value.Fields))
	for key, item := range value.Fields {
		fields[key] = item
	}
	return &turn.ApprovalEvent{
		RequestID: value.RequestID,
		Tool:      value.Tool,
		Action:    value.Action,
		Reason:    value.Reason,
		FrameID:   value.FrameID,
		Kind:      value.Kind,
		Prompt:    value.Prompt,
		SessionID: value.SessionID,
		Secure:    value.Secure,
		Fields:    fields,
	}
}

func toTurnCompact(value *CompactProgress) *turn.CompactProgress {
	if value == nil {
		return nil
	}
	return &turn.CompactProgress{
		Phase:               value.Phase,
		Trigger:             value.Trigger,
		Message:             value.Message,
		BoundaryID:          value.BoundaryID,
		SummarySource:       value.SummarySource,
		Chunk:               value.Chunk,
		Chunks:              value.Chunks,
		PreTokenEstimate:    value.PreTokenEstimate,
		PostTokenEstimate:   value.PostTokenEstimate,
		DroppedMessages:     value.DroppedMessages,
		RetainedMessages:    value.RetainedMessages,
		RestoredAttachments: value.RestoredAttachments,
		Model:               value.Model,
		Error:               value.Error,
	}
}
