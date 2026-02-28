package executor

import (
	"github.com/builtwithtofu/sigil/core/decorator"
	"github.com/builtwithtofu/sigil/core/planfmt"
)

func localTestTransports(ids ...string) []planfmt.Transport {
	transports := []planfmt.Transport{{ID: "local", Decorator: "local"}}
	for _, id := range ids {
		transports = append(transports, planfmt.Transport{ID: id, Decorator: "local"})
	}
	return transports
}

func scopedLocalSessionFactory(transportID string) (decorator.Session, error) {
	base := decorator.NewLocalSession()
	if normalizedTransportID(transportID) == "local" {
		return base, nil
	}

	return &transportScopedSession{id: transportID, session: base}, nil
}
