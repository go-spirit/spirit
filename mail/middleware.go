package mail

type InboundMiddlewareFunc func(Message) error

func (p *InboundMiddlewareFunc) Next(next InboundMiddlewareFunc) *InboundMiddlewareFunc {
	var h InboundMiddlewareFunc = func(m Message) (err error) {
		err = (*p)(m)
		if err != nil {
			return
		}

		err = next(m)
		if err != nil {
			return
		}

		return
	}

	return &h
}
