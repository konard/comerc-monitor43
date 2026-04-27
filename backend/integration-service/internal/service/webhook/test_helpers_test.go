package webhook

import (
	"context"
	"testing"
)

func stopWebhookDeliveryService(tb testing.TB, svc *WebhookDeliveryService) {
	tb.Helper()

	tb.Cleanup(func() {
		if err := svc.Stop(context.Background()); err != nil {
			tb.Errorf("stop webhook delivery service: %v", err)
		}
	})
}
