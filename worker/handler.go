package worker

import (
	"github.com/go-spirit/spirit/mail"
)

type SubscriberFunc func(mail.Session, error)

type HandlerFunc func(mail.Session) error

func (p HandlerFunc) Run(session mail.Session) error {
	return p(session)
}

func (p *HandlerFunc) Next(next HandlerFunc, subscribers ...SubscriberFunc) *HandlerFunc {
	var h HandlerFunc = func(session mail.Session) (err error) {
		err = (*p)(session)

		if err != nil {
			return
		}

		err = next(session)
		(*p).publish(session, err, subscribers...)

		if err != nil {
			return
		}

		return
	}

	return &h
}

func (p *HandlerFunc) Subscribe(subscribers ...SubscriberFunc) *HandlerFunc {
	var h HandlerFunc = func(session mail.Session) (err error) {
		err = (*p)(session)
		(*p).publish(session, err, subscribers...)

		return
	}

	return &h
}

func (p *HandlerFunc) publish(session mail.Session, err error, subscribers ...SubscriberFunc) {

	if len(subscribers) == 0 {
		return
	}

	go func(session mail.Session, err error, subscribers ...SubscriberFunc) {
		for _, s := range subscribers {
			if session != nil {
				s(session.Fork(), err)
			} else {
				s(nil, err)
			}
		}
	}(session, err, subscribers...)
}
