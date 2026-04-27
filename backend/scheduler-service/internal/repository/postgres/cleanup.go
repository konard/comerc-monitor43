package postgres

import (
	"database/sql"
	stderrors "errors"
	"fmt"
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
		wrappedErr := fmt.Errorf("%s: %w", message, closeErr)
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
		*errp = stderrors.Join(*errp, fmt.Errorf("%s: %w", message, rollbackErr))
	}
}
