package approvals

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
)

type State string

const (
	StateDraft    State = "draft"    // İçerik yalnızca oluşturan kullanıcı tarafından düzenlenebilir.
	StatePending  State = "pending"  // Onay bekleyen içerik.
	StateApproved State = "approved" // Onaylanmış ve öğrenciye sunulabilir içerik.
	StateRejected State = "rejected" // Reddedilmiş, düzeltme bekleyen içerik.
)

var States = []State{StateDraft, StatePending, StateApproved, StateRejected}

type Review struct {
	State        State      `gorm:"size:20;not null;default:'draft';index" json:"state"`
	ReviewedByID *uuid.UUID `gorm:"type:char(36);index" json:"reviewed_by_id"`
	ReviewedAt   *time.Time `json:"reviewed_at"`
	Note         string     `gorm:"type:text" json:"note"`
}

type Decision struct {
	State      State      `json:"state"`
	ReviewerID *uuid.UUID `json:"reviewer_id"`
	ReviewedAt *time.Time `json:"reviewed_at"`
	Note       string     `json:"note"`
}

type Action string

const (
	ActionSubmit  Action = "submit"  // Onaya gönderir.
	ActionApprove Action = "approve" // Onaylar.
	ActionReject  Action = "reject"  // Reddeder.
)

func (state State) IsValid() bool {
	for _, candidate := range States {
		if state == candidate {
			return true
		}
	}
	return false
}

func (state State) IsPublished() bool { return state == StateApproved }

func CanTransition(from State, to State) bool {
	switch from {
	case StateDraft:
		return to == StatePending || to == StateApproved
	case StatePending:
		return to == StateApproved || to == StateRejected || to == StateDraft
	case StateApproved:
		return to == StateDraft || to == StatePending
	case StateRejected:
		return to == StateDraft || to == StatePending
	default:
		return false
	}
}

func Draft() Decision { return Decision{State: StateDraft} }

func Submit() Decision { return Decision{State: StatePending} }

func Approve(reviewerID uuid.UUID, note string, at time.Time) Decision {
	return Decision{State: StateApproved, ReviewerID: &reviewerID, ReviewedAt: &at, Note: strings.TrimSpace(note)}
}

func Reject(reviewerID uuid.UUID, note string, at time.Time) Decision {
	return Decision{State: StateRejected, ReviewerID: &reviewerID, ReviewedAt: &at, Note: strings.TrimSpace(note)}
}

func Decide(action Action, reviewerID uuid.UUID, note string, at time.Time, from State) (Decision, error) {
	if !from.IsValid() {
		return Decision{}, core.ErrValidation
	}
	switch action {
	case ActionSubmit:
		target := StatePending
		if !CanTransition(from, target) {
			return Decision{}, core.ErrValidation
		}
		return Decision{State: target, Note: strings.TrimSpace(note)}, nil
	case ActionApprove:
		if !CanTransition(from, StateApproved) {
			return Decision{}, core.ErrValidation
		}
		return Approve(reviewerID, note, at), nil
	case ActionReject:
		if !CanTransition(from, StateRejected) {
			return Decision{}, core.ErrValidation
		}
		return Reject(reviewerID, note, at), nil
	default:
		return Decision{}, core.ErrValidation
	}
}

func Validate(decision Decision) error {
	if !decision.State.IsValid() {
		return core.ErrValidation
	}
	return nil
}

func Apply(review *Review, decision Decision) error {
	if err := Validate(decision); err != nil {
		return err
	}
	review.State = decision.State
	review.ReviewedByID = decision.ReviewerID
	review.ReviewedAt = decision.ReviewedAt
	review.Note = decision.Note
	return nil
}
