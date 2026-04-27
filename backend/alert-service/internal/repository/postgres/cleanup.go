package postgres

import (
	stderrors "errors"

	"github.com/pkg/errors"
)

func closeResource(resource interface{ Close() error }, errp *error, message string) {
	if closeErr := resource.Close(); closeErr != nil {
		*errp = stderrors.Join(*errp, errors.Wrap(closeErr, message))
	}
}
