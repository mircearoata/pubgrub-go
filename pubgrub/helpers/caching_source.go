package helpers

import (
	"github.com/mircearoata/pubgrub-go/pubgrub"
)

type cacheInstance struct {
	Versions []pubgrub.PackageVersion
	Error    error
	Waiter   chan bool
}

type CachingSource struct {
	pubgrub.Source
	cache map[string]*cacheInstance
}

func NewCachingSource(source pubgrub.Source) *CachingSource {
	return &CachingSource{
		Source: source,
		cache:  make(map[string]*cacheInstance),
	}
}

func (s *CachingSource) GetPackageVersions(pkg string) ([]pubgrub.PackageVersion, error) {
	if v, ok := s.cache[pkg]; ok {
		<-v.Waiter
		return v.Versions, v.Error
	}

	s.cache[pkg] = &cacheInstance{
		Versions: nil,
		Error:    nil,
		Waiter:   make(chan bool),
	}

	defer func() {
		close(s.cache[pkg].Waiter)
	}()

	result, err := s.Source.GetPackageVersions(pkg)
	s.cache[pkg].Versions = result
	s.cache[pkg].Error = err

	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	return result, nil
}
