package http

import (
	"context"
	"errors"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/go-spirit/spirit/component"
	"github.com/go-spirit/spirit/doc"
	"github.com/go-spirit/spirit/mail"
	"github.com/go-spirit/spirit/message"
	"github.com/go-spirit/spirit/protocol"
	"github.com/go-spirit/spirit/worker"
)

type HTTPComponent struct {
	opts component.Options

	router *gin.Engine
}

type ctxHttpComponentKey struct{}

type httpCacheItem struct {
	c    *gin.Context
	ctx  context.Context
	done chan struct{}
}

func init() {
	component.RegisterComponent("http-raw", NewHTTPComponent)
	doc.RegisterDocumenter("http-raw", &HTTPComponent{})

}

func NewHTTPComponent(opts ...component.Option) (srv component.Component, err error) {
	s := &HTTPComponent{}

	s.init(opts...)

	srv = s

	return
}

func (p *HTTPComponent) init(opts ...component.Option) {

	for _, o := range opts {
		o(&p.opts)
	}

	debug := p.opts.Config.GetBoolean("debug", false)
	if !debug {
		gin.SetMode("release")
	}

	rootPath := p.opts.Config.GetString("path", "/")

	router := gin.New()
	router.Use(gin.Recovery())
	router.POST(path.Join(rootPath, "message"), p.serve)

	p.router = router

	return
}

func (p *HTTPComponent) serve(c *gin.Context) {

	var err error

	defer func() {
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		}
	}()

	strWait := c.DefaultQuery("wait", "false")
	strTimeout := c.DefaultQuery("timeout", "0s")

	wait, err := strconv.ParseBool(strWait)
	if err != nil {
		return
	}

	timeout, err := time.ParseDuration(strTimeout)
	if err != nil {
		return
	}

	payload := &protocol.Payload{}

	err = c.ShouldBind(payload)

	if err != nil {
		return
	}

	port, err := payload.GetGraph().CurrentPort()

	if err != nil {
		return
	}

	session := mail.NewSession()

	session.WithPayload(payload)
	session.WithFromTo("", port.GetUrl())

	var ctx context.Context
	var cancel context.CancelFunc
	var doneChan chan struct{}

	if wait {

		ctx, cancel = context.WithTimeout(context.Background(), timeout)
		defer cancel()

		doneChan = make(chan struct{})
		defer close(doneChan)

		session.WithValue(ctxHttpComponentKey{}, &httpCacheItem{c, ctx, doneChan})

	} else {
		session.WithValue(ctxHttpComponentKey{}, &httpCacheItem{c, nil, nil})
	}

	err = p.opts.Postman.Post(
		message.NewUserMessage(session),
	)

	if err != nil {
		logrus.WithField("component", "http-raw").WithError(err).Errorln("post user message failure")
		return
	}

	if !wait {
		c.JSON(http.StatusOK, struct{}{})
		return
	}

	for {
		select {
		case <-doneChan:
			{
				return
			}
		case <-ctx.Done():
			{
				c.JSON(http.StatusRequestTimeout, gin.H{"message": "request timeout"})
				return
			}
		}
	}

}

func (p *HTTPComponent) Handler() worker.HandlerFunc {
	return func(session mail.Session) (err error) {

		item, ok := session.Value(ctxHttpComponentKey{}).(*httpCacheItem)
		if !ok {
			err = errors.New("http component handler could not get response object")
			return
		}

		if item.done == nil || item.ctx == nil {
			return
		}

		payload, ok := session.Payload().(*protocol.Payload)
		if !ok {
			err = errors.New("could not convert session payload to *protocol.Payload")
			return
		}

		if item.ctx.Err() != nil {
			return
		}

		item.c.JSON(http.StatusOK, payload)

		item.done <- struct{}{}

		return
	}
}

func (p *HTTPComponent) Start() error {
	go p.router.Run()
	return nil
}

func (p *HTTPComponent) Stop() error {
	return nil
}

func (p *HTTPComponent) Document() doc.Document {
	document := doc.Document{
		Title: "Raw HTTP message service",
		Description: `we could send protocol.Payload to this http service for internal use,
		there are two params for url query, [wait=false] and [timeout=0s], if the request graph
		where resend the response to this component handler, you should let wait=true and timeout > 0s,
		then you will get http response`,
	}

	return document
}
