// Package group stores photo fingerprints in NuclaDB and gathers photos
// that look alike into groups.
package group

import (
	"context"

	"github.com/Rakshit-gen/lookalike/internal/library"
	"github.com/Rakshit-gen/lookalike/internal/nucla"
	"github.com/Rakshit-gen/lookalike/internal/photo"
)

// Tenant is the NuclaDB tenant for a library. Each folder gets its own.
func Tenant(l *library.Library) string { return "lookalike-" + library.Key(l.Root) }

// Push writes the fingerprints Refresh produced and deletes the vectors of
// photos that are gone.
func Push(ctx context.Context, db *nucla.Client, l *library.Library, ch library.Changes) error {
	t := Tenant(l)
	if err := db.EnsureTenant(ctx, t, photo.Dim); err != nil {
		return err
	}
	vs := make([]nucla.Vector, 0, len(ch.Fingerprinted))
	for _, e := range ch.Fingerprinted {
		vs = append(vs, nucla.Vector{ID: e.ID(), Values: e.Vector, Metadata: map[string]string{"path": e.Path}})
	}
	if err := db.Upsert(ctx, t, vs); err != nil {
		return err
	}
	for _, e := range ch.Removed {
		if err := db.Delete(ctx, t, e.ID()); err != nil {
			return err
		}
	}
	return nil
}
