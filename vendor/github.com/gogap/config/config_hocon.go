package config

import (
	"github.com/go-akka/configuration"
)

type HOCONConfiguration struct {
	*configuration.Config
}

func NewHOCONConfiguration(conf *configuration.Config) Configuration {
	return &HOCONConfiguration{
		conf,
	}
}

func (p *HOCONConfiguration) GetConfig(path string) Configuration {
	conf := p.Config.GetConfig(path)
	if conf == nil {
		return nil
	}
	return &HOCONConfiguration{conf}
}

func (p *HOCONConfiguration) WithFallback(fallback Configuration) Configuration {
	if fallback == nil {
		return p
	}

	p.Config = p.Config.WithFallback(fallback.(*HOCONConfiguration).Config)
	return p
}

func (p *HOCONConfiguration) Keys() []string {
	return p.Config.Root().GetObject().GetKeys()
}

type HOCONConfigProvider struct {
}

func (p *HOCONConfigProvider) LoadConfig(filename string) Configuration {
	conf := configuration.LoadConfig(filename)
	return NewHOCONConfiguration(conf)
}

func (p *HOCONConfigProvider) ParseString(cfgStr string) Configuration {
	conf := configuration.ParseString(cfgStr)
	return NewHOCONConfiguration(conf)
}
