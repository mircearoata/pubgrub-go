package helpers

import (
	"sync"

	"github.com/mircearoata/pubgrub-go/pubgrub"
)

type cacheInstance struct {
	Versions []pubgrub.PackageVersion
	Error    error
	Waiter   chan bool
}

type CachingSource struct {
	pubgrub.Source
	cache sync.Map
}

func NewCachingSource(source pubgrub.Source) *CachingSource {
	return &CachingSource{
		Source: source,
		cache:  sync.Map{},
	}
}

func (s *CachingSource) GetPackageVersions(pkg string) ([]pubgrub.PackageVersion, error) {
	actual, loaded := s.cache.LoadOrStore(pkg, &cacheInstance{
		Versions: nil,
		Error:    nil,
		Waiter:   make(chan bool),
	})

	instance := actual.(*cacheInstance)

	if loaded {
		<-instance.Waiter
		return instance.Versions, instance.Error
	}

	defer func() {
		close(instance.Waiter)
	}()

	result, err := s.Source.GetPackageVersions(pkg)
	instance.Versions = result
	instance.Error = err

	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	return result, nil
}
