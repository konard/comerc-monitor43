package postgres

import (
	"database/sql"
	stderrors "errors"

	pkgerrors "github.com/pkg/errors"
)

type closableResource interface {
	Close() error
}

type rollbackableTx interface {
	Rollback() error
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

func rollbackOnError(tx rollbackableTx, errp *error, message string) {
	if tx == nil || *errp == nil {
		return
	}

	if rollbackErr := tx.Rollback(); rollbackErr != nil && !stderrors.Is(rollbackErr, sql.ErrTxDone) {
		*errp = stderrors.Join(*errp, pkgerrors.Wrap(rollbackErr, message))
	}
}
