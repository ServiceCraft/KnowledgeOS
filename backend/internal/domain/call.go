package domain

import (
	"time"

	"github.com/google/uuid"
)

type Call struct {
	SyncableModel
	ExternalID  *string    `gorm:"type:text" json:"external_id,omitempty"`
	Title       string     `gorm:"type:text;not null;default:''" json:"title"`
	OccurredAt  *time.Time `json:"occurred_at,omitempty"`
	DurationSec *int       `json:"duration_sec,omitempty"`
	AudioURL    *string    `gorm:"type:text" json:"audio_url,omitempty"`
	Transcript  string     `gorm:"type:text;not null;default:''" json:"transcript"`
}

func (Call) TableName() string { return "calls" }

type QAPairCallMention struct {
	SyncableModel
	QAPairID    uuid.UUID `gorm:"type:uuid;not null" json:"qa_pair_id"`
	CallID      uuid.UUID `gorm:"type:uuid;not null" json:"call_id"`
	Snippet     string    `gorm:"type:text;not null" json:"snippet"`
	StartOffset *int      `json:"start_offset,omitempty"`
	EndOffset   *int      `json:"end_offset,omitempty"`
	StartSec    *float64  `gorm:"type:numeric(10,3)" json:"start_sec,omitempty"`
	EndSec      *float64  `gorm:"type:numeric(10,3)" json:"end_sec,omitempty"`
	Confidence  *float64  `gorm:"type:numeric(4,3)" json:"confidence,omitempty"`
}

func (QAPairCallMention) TableName() string { return "qa_pair_call_mentions" }

// QAPairCallMentionView is the row returned by GET /qa/{id}/mentions —
// a mention enriched with metadata from the joined call so the UI does not
// need a second round trip.
type QAPairCallMentionView struct {
	QAPairCallMention
	CallTitle      string     `json:"call_title"`
	CallOccurredAt *time.Time `json:"call_occurred_at,omitempty"`
}
