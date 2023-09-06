package helpers

import "github.com/mircearoata/pubgrub-go/lib"

type CachingSource struct {
	lib.Source
	cache map[string][]lib.PackageVersion
}

func NewCachingSource(source lib.Source) *CachingSource {
	return &CachingSource{
		Source: source,
		cache:  map[string][]lib.PackageVersion{},
	}
}

func (s *CachingSource) GetPackageVersions(pkg string) ([]lib.PackageVersion, error) {
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
