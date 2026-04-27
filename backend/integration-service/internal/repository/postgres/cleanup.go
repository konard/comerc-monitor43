package postgres

import (
	stderrors "errors"

	pkgerrors "github.com/pkg/errors"
)

type closableResource interface {
	Close() error
}

func closeResource(resource closableResource, errp *error, message string) {
	if resource == nil {
		return
	}

	if closeErr := resource.Close(); closeErr != nil {
		wrappedErr := pkgerrors.Wrap(closeErr, message)
		if *errp == nil {
			*errp = wrappedErr
			return
		}

		*errp = stderrors.Join(*errp, wrappedErr)
	}
}
