package aggregator

import (
	"fmt"

	"ripple/internal/model"
)

// CoalescedNotification represents the coalesced notification message output.
type CoalescedNotification struct {
	Verb        string           `json:"verb"`
	SummaryText string           `json:"summary_text"`
	Subject     string           `json:"subject"`
	Body        string           `json:"body"`
	TotalCount  int              `json:"total_count"`
	Activities  []model.Activity `json:"activities"`
}

// Coalescer formats aggregated notification batches into human-friendly coalesced strings.
type Coalescer struct {
	templateEngine *TemplateEngine
}

func NewCoalescer() *Coalescer {
	return &Coalescer{
		templateEngine: NewTemplateEngine(),
	}
}

// Coalesce takes a slice of activities and produces a human-friendly summary text, subject, and body.
func (c *Coalescer) Coalesce(activities []model.Activity, customSubjectTmpl, customBodyTmpl string) *CoalescedNotification {
	if len(activities) == 0 {
		return &CoalescedNotification{
			SummaryText: "",
			TotalCount:  0,
			Activities:  nil,
		}
	}

	count := len(activities)
	firstActor := activities[0].ActorID
	verb := activities[0].Verb
	objectID := activities[0].ObjectID
	targetID := activities[0].TargetID

	var summary string
	switch count {
	case 1:
		summary = fmt.Sprintf("%s %ss your post", firstActor, verb)
		if verb == "like" {
			summary = fmt.Sprintf("%s liked your post", firstActor)
		} else if verb == "comment" {
			summary = fmt.Sprintf("%s commented on your post", firstActor)
		}
	case 2:
		secondActor := activities[1].ActorID
		summary = fmt.Sprintf("%s and %s %ss your post", firstActor, secondActor, verb)
		if verb == "like" {
			summary = fmt.Sprintf("%s and %s liked your post", firstActor, secondActor)
		} else if verb == "comment" {
			summary = fmt.Sprintf("%s and %s commented on your post", firstActor, secondActor)
		}
	default:
		otherCount := count - 1
		summary = fmt.Sprintf("%s and %d others %ss your post", firstActor, otherCount, verb)
		if verb == "like" {
			summary = fmt.Sprintf("%s and %d others liked your post", firstActor, otherCount)
		} else if verb == "comment" {
			summary = fmt.Sprintf("%s and %d others commented on your post", firstActor, otherCount)
		}
	}

	tmplCtx := &TemplateContext{
		ActorName:  firstActor,
		Count:      count,
		OtherCount: count - 1,
		Verb:       verb,
		ObjectID:   objectID,
		TargetID:   targetID,
	}

	subject := summary
	if customSubjectTmpl != "" {
		subject = c.templateEngine.Render(customSubjectTmpl, tmplCtx)
	}

	body := summary
	if customBodyTmpl != "" {
		body = c.templateEngine.Render(customBodyTmpl, tmplCtx)
	}

	return &CoalescedNotification{
		Verb:        verb,
		SummaryText: summary,
		Subject:     subject,
		Body:        body,
		TotalCount:  count,
		Activities:  activities,
	}
}
