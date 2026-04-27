package channels

import (
	stderrors "errors"
	"fmt"
	"net/http"
)

func closeResponseBody(resp *http.Response, errp *error, message string) {
	if resp == nil || resp.Body == nil {
		return
	}

	if closeErr := resp.Body.Close(); closeErr != nil {
		wrappedErr := fmt.Errorf("%s: %w", message, closeErr)
		if *errp == nil {
			*errp = wrappedErr
			return
		}

		*errp = stderrors.Join(*errp, wrappedErr)
	}
}
