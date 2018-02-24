package message

import (
	"fmt"
	"github.com/go-spirit/spirit/mail"
)

type Stop struct {
	mail.SystemMessage
}

type UserMessage struct {
	mail.UserMessage

	session mail.Session
}

func NewUserMessage(session mail.Session) *UserMessage {
	return &UserMessage{
		session: session,
	}
}

func (p *UserMessage) Session() mail.Session {
	return p.session
}

func (p *UserMessage) String() string {
	if p.session != nil {
		return p.session.String()
	}

	return fmt.Sprintf("<< UserMessage: %v >>", p.session)
}

// ResumeMailbox is message sent by the actor system to resume mailbox processing.
//
// This will not be forwarded to the Receive method
type ResumeMailbox struct{}

// SuspendMailbox is message sent by the actor system to suspend mailbox processing.
//
// This will not be forwarded to the Receive method
type SuspendMailbox struct{}
