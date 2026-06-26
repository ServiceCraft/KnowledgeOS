package eval

import (
	_ "embed"
	"encoding/json"
)

//go:embed fixtures.json
var fixturesJSON []byte

// DefaultCases returns the built-in in-domain and out-of-domain/provocative
// eval sets. The in-domain questions assume a populated, indexed knowledge base
// for the target tenant.
func DefaultCases() ([]Case, error) {
	var cases []Case
	if err := json.Unmarshal(fixturesJSON, &cases); err != nil {
		return nil, err
	}
	return cases, nil
}
