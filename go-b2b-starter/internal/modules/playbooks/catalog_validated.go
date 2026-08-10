package playbooks

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	playbooksServices "github.com/moasq/go-b2b-starter/internal/modules/playbooks/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/playbooks/domain"
)

// ErrCatalogValidationFailed is returned when the Go catalog and the seeded
// modules.playbooks rows drift apart. The server fails fast on it at boot so
// catalog.go and the migration seed cannot silently diverge.
var ErrCatalogValidationFailed = errors.New("playbook catalog validation failed")

// CatalogValidated matches the Go catalog (catalog.go) against the seeded
// modules.playbooks rows. It is the runtime counterpart to catalog_test.go:
// the test asserts the Go side in isolation, this check compares it with the
// actual DB seed. Any mismatch — missing/unknown vertical key, name/vertical
// drift, or guiones (including pasos) differing — fails fast.
func CatalogValidated(ctx context.Context, repo domain.PlaybookRepository) error {
	rows, err := repo.ListActive(ctx)
	if err != nil {
		return fmt.Errorf("catalog validation: load seeded playbooks: %w", err)
	}

	seeded := make(map[string]*domain.Playbook, len(rows))
	for _, row := range rows {
		seeded[row.Key] = row
	}

	catalog := Catalog()
	catalogByKey := make(map[string]domain.Playbook, len(catalog))
	for _, pb := range catalog {
		catalogByKey[pb.Key] = pb

		row, ok := seeded[pb.Key]
		if !ok {
			return fmt.Errorf("%w: playbook %q missing from DB seed", ErrCatalogValidationFailed, pb.Key)
		}
		if row.Vertical != pb.Vertical {
			return fmt.Errorf("%w: vertical drift for %q (catalog=%q db=%q)", ErrCatalogValidationFailed, pb.Key, pb.Vertical, row.Vertical)
		}
		if row.Name != pb.Name {
			return fmt.Errorf("%w: name drift for %q (catalog=%q db=%q)", ErrCatalogValidationFailed, pb.Key, pb.Name, row.Name)
		}

		expected, err := playbooksServices.ParsePayload(pb.Payload)
		if err != nil {
			return fmt.Errorf("%w: catalog %q payload invalid: %v", ErrCatalogValidationFailed, pb.Key, err)
		}
		actual, err := playbooksServices.ParsePayload(row.Payload)
		if err != nil {
			return fmt.Errorf("%w: seeded %q payload invalid: %v", ErrCatalogValidationFailed, pb.Key, err)
		}
		if !reflect.DeepEqual(expected.Guiones, actual.Guiones) {
			return fmt.Errorf("%w: guiones drift for %q (incl. pasos)", ErrCatalogValidationFailed, pb.Key)
		}
	}

	for key := range seeded {
		if _, ok := catalogByKey[key]; !ok {
			return fmt.Errorf("%w: seeded playbook %q missing from catalog", ErrCatalogValidationFailed, key)
		}
	}
	return nil
}
