package httpapi

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/coder/quartz"

	mf "github.com/coder/agentapi/lib/msgfmt"
	st "github.com/coder/agentapi/lib/screentracker"
	"github.com/coder/agentapi/lib/transcript"
	"github.com/coder/agentapi/lib/util"
	"github.com/danielgtaylor/huma/v2"
)

type EventType string

const (
	EventTypeMessageUpdate EventType = "message_update"
	EventTypeStatusChange  EventType = "status_change"
	EventTypeScreenUpdate  EventType = "screen_update"
	EventTypeError         EventType = "agent_error"
	EventTypeTimeline      EventType = "timeline_event"
)

type AgentStatus string

const (
	AgentStatusRunning AgentStatus = "running"
	AgentStatusStable  AgentStatus = "stable"
)

var AgentStatusValues = []AgentStatus{
	AgentStatusStable,
	AgentStatusRunning,
}

func (a AgentStatus) Schema(r huma.Registry) *huma.Schema {
	return util.OpenAPISchema(r, "AgentStatus", AgentStatusValues)
}

type MessageUpdateBody struct {
	Id      int                 `json:"id" doc:"Unique identifier for the message. This identifier also represents the order of the message in the conversation history."`
	Role    st.ConversationRole `json:"role" doc:"Role of the message author"`
	Message string              `json:"message" doc:"Message content. The message is formatted as it appears in the agent's terminal session, meaning that, by default, it consists of lines of text with 80 characters per line."`
	Time    time.Time           `json:"time" doc:"Timestamp of the message"`
}

type StatusChangeBody struct {
	Status    AgentStatus  `json:"status" doc:"Agent status"`
	AgentType mf.AgentType `json:"agent_type" doc:"Type of the agent being used by the server."`
}

type ScreenUpdateBody struct {
	Screen string `json:"screen"`
}

type ErrorBody struct {
	Message string        `json:"message" doc:"Error message"`
	Level   st.ErrorLevel `json:"level" doc:"Error level"`
	Time    time.Time     `json:"time" doc:"Timestamp when the error occurred"`
}

type TimelineKind string

var TimelineKindValues = []TimelineKind{
	TimelineKind(transcript.KindThinking),
	TimelineKind(transcript.KindText),
	TimelineKind(transcript.KindToolCall),
	TimelineKind(transcript.KindToolResult),
	TimelineKind(transcript.KindSystem),
}

func (t TimelineKind) Schema(r huma.Registry) *huma.Schema {
	return util.OpenAPISchema(r, "TimelineKind", TimelineKindValues)
}

// TimelineEventBody is a structured event captured from the agent's
// transcript files: thinking, text, tool calls, tool results.
type TimelineEventBody struct {
	Id        int             `json:"id" doc:"Unique identifier for the timeline event. Also represents the order of the event."`
	Kind      TimelineKind    `json:"kind" doc:"Kind of the event"`
	Role      string          `json:"role,omitempty" doc:"Author of the event: assistant, user, or system"`
	Time      time.Time       `json:"time" doc:"Timestamp of the event"`
	SessionId string          `json:"session_id,omitempty" doc:"Agent-native session identifier"`
	Content   string          `json:"content,omitempty" doc:"Event content: thinking text, message text, or tool result output"`
	ToolName  string          `json:"tool_name,omitempty" doc:"Tool name (tool_call events only)"`
	ToolInput json.RawMessage `json:"tool_input,omitempty" doc:"Tool input as raw JSON (tool_call events only)"`
	ToolUseId string          `json:"tool_use_id,omitempty" doc:"Identifier joining a tool_call with its tool_result"`
	SourceId  string          `json:"source_id,omitempty" doc:"Transcript-native identifier of the source entry"`
}

func toTimelineEventBody(ev transcript.TimelineEvent) TimelineEventBody {
	return TimelineEventBody{
		Id:        ev.Id,
		Kind:      TimelineKind(ev.Kind),
		Role:      ev.Role,
		Time:      ev.Time,
		SessionId: ev.SessionId,
		Content:   ev.Content,
		ToolName:  ev.ToolName,
		ToolInput: ev.ToolInput,
		ToolUseId: ev.ToolUseId,
		SourceId:  ev.SourceId,
	}
}

type Event struct {
	Type    EventType
	Payload any
}

type EventEmitter struct {
	mu                  sync.Mutex
	messages            []st.ConversationMessage
	status              AgentStatus
	agentType           mf.AgentType
	chans               map[int]chan Event
	chanIdx             int
	subscriptionBufSize uint
	screen              string
	errors              []ErrorBody
	timeline            []TimelineEventBody
	timelineNextId      int
	clock               quartz.Clock
}

func convertStatus(status st.ConversationStatus) (AgentStatus, error) {
	switch status {
	case st.ConversationStatusInitializing:
		return AgentStatusRunning, nil
	case st.ConversationStatusStable:
		return AgentStatusStable, nil
	case st.ConversationStatusChanging:
		return AgentStatusRunning, nil
	default:
		return "", fmt.Errorf("unknown conversation status: %s", status)
	}
}

const defaultSubscriptionBufSize uint = 1024

// maxStoredErrors caps the number of errors retained for late subscribers.
const maxStoredErrors = 100

// maxStoredTimelineEvents caps the in-memory timeline; the oldest events are
// dropped first. Tool results dominate event size, so this bounds worst-case
// memory to tens of MB.
const maxStoredTimelineEvents = 10000

// maxTimelineReplay caps how many recent timeline events are replayed to a
// late SSE subscriber. The full history stays available via GET /timeline,
// and monotonic ids let clients detect the gap.
const maxTimelineReplay = 500

type EventEmitterOption func(*EventEmitter)

func WithSubscriptionBufSize(size uint) EventEmitterOption {
	return func(e *EventEmitter) {
		if size == 0 {
			e.subscriptionBufSize = defaultSubscriptionBufSize
		} else {
			e.subscriptionBufSize = size
		}
	}
}

func WithAgentType(agentType mf.AgentType) EventEmitterOption {
	return func(e *EventEmitter) {
		e.agentType = agentType
	}
}

func WithClock(clock quartz.Clock) EventEmitterOption {
	return func(e *EventEmitter) {
		e.clock = clock
	}
}

func NewEventEmitter(opts ...EventEmitterOption) *EventEmitter {
	e := &EventEmitter{
		messages:            make([]st.ConversationMessage, 0),
		status:              AgentStatusRunning,
		chans:               make(map[int]chan Event),
		subscriptionBufSize: defaultSubscriptionBufSize,
	}
	for _, opt := range opts {
		opt(e)
	}
	if e.clock == nil {
		e.clock = quartz.NewReal()
	}
	return e
}

// Assumes the caller holds the lock.
func (e *EventEmitter) notifyChannels(eventType EventType, payload any) {
	chanIds := make([]int, 0, len(e.chans))
	for chanId := range e.chans {
		chanIds = append(chanIds, chanId)
	}
	for _, chanId := range chanIds {
		ch := e.chans[chanId]
		event := Event{
			Type:    eventType,
			Payload: payload,
		}

		select {
		case ch <- event:
		default:
			// If the channel is full, close it.
			// Listeners must actively drain the channel.
			e.unsubscribeInner(chanId)
		}
	}
}

// EmitMessages assumes that only the last message can change or new messages can be added.
// If a new message is injected between existing messages (identified by Id), the behavior is undefined.
func (e *EventEmitter) EmitMessages(newMessages []st.ConversationMessage) {
	e.mu.Lock()
	defer e.mu.Unlock()

	maxLength := max(len(e.messages), len(newMessages))
	for i := range maxLength {
		var oldMsg st.ConversationMessage
		var newMsg st.ConversationMessage
		if i < len(e.messages) {
			oldMsg = e.messages[i]
		}
		if i < len(newMessages) {
			newMsg = newMessages[i]
		}
		if oldMsg != newMsg {
			if i >= len(newMessages) {
				continue
			}
			e.notifyChannels(EventTypeMessageUpdate, MessageUpdateBody{
				Id:      newMessages[i].Id,
				Role:    newMessages[i].Role,
				Message: newMessages[i].Message,
				Time:    newMessages[i].Time,
			})
		}
	}

	e.messages = newMessages
}

func (e *EventEmitter) EmitStatus(newStatus st.ConversationStatus) {
	e.mu.Lock()
	defer e.mu.Unlock()

	newAgentStatus, err := convertStatus(newStatus)
	if err != nil {
		return
	}
	if e.status == newAgentStatus {
		return
	}

	e.notifyChannels(EventTypeStatusChange, StatusChangeBody{Status: newAgentStatus, AgentType: e.agentType})
	e.status = newAgentStatus
}

func (e *EventEmitter) EmitScreen(newScreen string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.screen == newScreen {
		return
	}

	e.notifyChannels(EventTypeScreenUpdate, ScreenUpdateBody{Screen: strings.TrimRight(newScreen, mf.WhiteSpaceChars)})
	e.screen = newScreen
}

func (e *EventEmitter) EmitError(message string, level st.ErrorLevel) {
	e.mu.Lock()
	defer e.mu.Unlock()

	errorBody := ErrorBody{
		Message: message,
		Level:   level,
		Time:    e.clock.Now(),
	}

	// Store the error so new subscribers can receive recent errors.
	e.errors = append(e.errors, errorBody)
	if len(e.errors) > maxStoredErrors {
		e.errors = e.errors[len(e.errors)-maxStoredErrors:]
	}

	e.notifyChannels(EventTypeError, errorBody)
}

// EmitTimelineEvent stores a normalized transcript event and broadcasts it
// to subscribers. Ids are assigned here, monotonically.
func (e *EventEmitter) EmitTimelineEvent(ev transcript.TimelineEvent) {
	e.mu.Lock()
	defer e.mu.Unlock()

	ev.Id = e.timelineNextId
	e.timelineNextId++

	body := toTimelineEventBody(ev)
	e.timeline = append(e.timeline, body)
	if len(e.timeline) > maxStoredTimelineEvents {
		e.timeline = e.timeline[len(e.timeline)-maxStoredTimelineEvents:]
	}

	e.notifyChannels(EventTypeTimeline, body)
}

// ClearTimeline drops all stored timeline events. The id counter keeps
// increasing so clients polling with since_id never miss events emitted
// after a clear.
func (e *EventEmitter) ClearTimeline() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.timeline = nil
}

// Timeline returns stored timeline events with id > sinceId, optionally
// filtered by kind (empty kind matches all).
func (e *EventEmitter) Timeline(sinceId int, kind TimelineKind) []TimelineEventBody {
	e.mu.Lock()
	defer e.mu.Unlock()

	events := make([]TimelineEventBody, 0, len(e.timeline))
	for _, ev := range e.timeline {
		if ev.Id <= sinceId {
			continue
		}
		if kind != "" && ev.Kind != kind {
			continue
		}
		events = append(events, ev)
	}
	return events
}

// Assumes the caller holds the lock.
func (e *EventEmitter) currentStateAsEvents() []Event {
	events := make([]Event, 0, len(e.messages)+2)
	for _, msg := range e.messages {
		events = append(events, Event{
			Type:    EventTypeMessageUpdate,
			Payload: MessageUpdateBody{Id: msg.Id, Role: msg.Role, Message: msg.Message, Time: msg.Time},
		})
	}
	events = append(events, Event{
		Type:    EventTypeStatusChange,
		Payload: StatusChangeBody{Status: e.status, AgentType: e.agentType},
	})
	events = append(events, Event{
		Type:    EventTypeScreenUpdate,
		Payload: ScreenUpdateBody{Screen: strings.TrimRight(e.screen, mf.WhiteSpaceChars)},
	})

	// Include all error events
	for _, err := range e.errors {
		events = append(events, Event{
			Type:    EventTypeError,
			Payload: err,
		})
	}

	// Replay only the most recent timeline events; the full history is
	// available via GET /timeline.
	timeline := e.timeline
	if len(timeline) > maxTimelineReplay {
		timeline = timeline[len(timeline)-maxTimelineReplay:]
	}
	for _, ev := range timeline {
		events = append(events, Event{
			Type:    EventTypeTimeline,
			Payload: ev,
		})
	}

	return events
}

// Subscribe returns:
// - a subscription ID that can be used to unsubscribe.
// - a channel for receiving events.
// - a list of events that allow to recreate the state of the conversation right before the subscription was created.
func (e *EventEmitter) Subscribe() (int, <-chan Event, []Event) {
	e.mu.Lock()
	defer e.mu.Unlock()
	stateEvents := e.currentStateAsEvents()

	// Once a channel becomes full, it will be closed.
	ch := make(chan Event, e.subscriptionBufSize)
	e.chans[e.chanIdx] = ch
	e.chanIdx++
	return e.chanIdx - 1, ch, stateEvents
}

// Assumes the caller holds the lock.
func (e *EventEmitter) unsubscribeInner(chanId int) {
	close(e.chans[chanId])
	delete(e.chans, chanId)
}

func (e *EventEmitter) Unsubscribe(chanId int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.unsubscribeInner(chanId)
}
