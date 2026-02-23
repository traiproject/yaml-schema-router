// Package detector defines the core interface and evaluation chain for identifying file schemas.
package detector

import "log"

// Detector defines the contract for all schema detectors.
type Detector interface {
	Name() string
	Detect(uri string, content []byte) (schemaURLs []string, err error)
}

// Chain manages a sequence of Detectors.
type Chain struct {
	detectors []Detector
}

// NewChain creates a new Chain of Responsibility.
func NewChain(detectors ...Detector) *Chain {
	return &Chain{
		detectors: detectors,
	}
}

// Run iterates through all detectors and aggregates every claimed file schema.
func (c *Chain) Run(uri string, content []byte) (schemaURLs []string, err error) {
	var allURLs []string

	for _, d := range c.detectors {
		urls, err := d.Detect(uri, content)
		if err != nil {
			log.Printf("[%s] Error during detection: %v", d.Name(), err)
			continue
		}

		if len(urls) > 0 {
			allURLs = append(allURLs, urls...)
		}
	}

	return allURLs, nil
}
