package detector

// Detector defines the contract for all schema detectors.
type Detector interface {
	Name() string
	Detect(uri string, content []byte) (schemaURL string, detected bool, err error)
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

// Run iterates through the detectors until one successfully claims the file.
func (c *Chain) Run(uri string, content []byte) (string, bool, error) {
	for _, d := range c.detectors {
		schemaUrl, detected, err := d.Detect(uri, content)

		if err != nil {
			// TODO: loggin
			continue
		}

		if detected {
			return schemaUrl, true, nil
		}
	}

	// No detector claimed this file.
	return "", false, nil
}
