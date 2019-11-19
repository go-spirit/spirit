package spirit

import (
	"github.com/gogap/config"
	// "github.com/go-spirit/spirit/mail"
	"github.com/go-spirit/spirit/worker"
)

type WorkerOptions struct {
	Url           string
	HandlerRouter worker.HandlerRouter
}

type WorkerOption func(*WorkerOptions)

func WorkerUrl(url string) WorkerOption {
	return func(w *WorkerOptions) {
		w.Url = url
	}
}

func WorkerHandlerRouter(h worker.HandlerRouter) WorkerOption {
	return func(w *WorkerOptions) {
		w.HandlerRouter = h
	}
}

type Options struct {
	config         config.Configuration
	configProvider string
}

type Option func(*Options)

func ConfigFile(filename, configProvider string) Option {
	return func(o *Options) {
		o.config = config.NewConfig(
			config.ConfigFile(filename),
			config.ConfigProviderByName(configProvider),
		)
		o.configProvider = configProvider
	}
}
