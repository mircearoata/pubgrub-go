package helpers

import "github.com/mircearoata/pubgrub-go"

type CachingSource struct {
	pubgrub.Source
	cache map[string][]pubgrub.PackageVersion
}

func NewCachingSource(source pubgrub.Source) *CachingSource {
	return &CachingSource{
		Source: source,
		cache:  map[string][]pubgrub.PackageVersion{},
	}
}

func (s *CachingSource) GetPackageVersions(pkg string) ([]pubgrub.PackageVersion, error) {
	if v, ok := s.cache[pkg]; ok {
		return v, nil
	}
	result, err := s.Source.GetPackageVersions(pkg)
	if err != nil {
		return nil, err
	}
	s.cache[pkg] = result
	return result, nil
}
