package cache

import (
	"github.com/gogap/config"
)

type Caches struct {
	conf   config.Configuration
	caches map[string]Cache
}

func NewCaches(conf config.Configuration) (caches *Caches, err error) {

	if conf == nil {
		caches = &Caches{}
		return
	}

	mapCaches := map[string]Cache{}

	for _, k := range conf.Keys() {

		driver := conf.GetString(k + ".driver")

		var newCache Cache
		newCache, err = NewCache(driver, Config(conf.GetConfig(k+".options")))
		if err != nil {
			return
		}

		mapCaches[k] = newCache
	}

	return &Caches{
		conf,
		mapCaches,
	}, nil
}

func (p *Caches) Require(name string) (cache Cache, exist bool) {
	if p.caches == nil {
		return
	}

	cache, exist = p.caches[name]

	return
}
