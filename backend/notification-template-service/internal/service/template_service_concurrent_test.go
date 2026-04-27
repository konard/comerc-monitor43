package service

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/rs/zerolog"

	"github.com/raul/monitor/backend/notification-template-service/internal/config"
	"github.com/raul/monitor/backend/notification-template-service/internal/model"
)

// TestTemplateService_ConcurrentCreates проверяет конкурентное создание шаблонов.
func TestTemplateService_ConcurrentCreates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	logger := zerolog.Nop()
	cfg := &config.Config{MaxTemplateSize: 100000}
	repo := NewMockTemplateRepository()
	svc := NewTemplateService(repo, &logger, cfg)

	ctx := context.Background()
	numGoroutines := 10

	var wg sync.WaitGroup
	errs := make(chan error, numGoroutines)

	// Запускаем несколько goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			req := &CreateTemplateRequest{
				UserID:  "user-concurrent",
				Name:    fmt.Sprintf("Concurrent Template %d", idx),
				Channel: model.ChannelEmail,
				Type:    model.TypeMonitorDown,
				Body:    fmt.Sprintf("Body %d", idx),
			}

			_, err := svc.CreateTemplate(ctx, req)
			if err != nil {
				errs <- err
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	// Проверяем, что не было ошибок
	for err := range errs {
		if err != nil {
			t.Errorf("concurrent creation failed: %v", err)
		}
	}

	// Проверяем, что все шаблоны созданы
	templates, total, err := svc.ListTemplates(ctx, ListFilter{
		UserID: "user-concurrent",
	})
	if err != nil {
		t.Fatalf("failed to list templates: %v", err)
	}

	if total != numGoroutines {
		t.Errorf("expected %d templates, got %d", numGoroutines, total)
	}

	if len(templates) != numGoroutines {
		t.Errorf("expected %d templates, got %d", numGoroutines, len(templates))
	}
}

// TestTemplateService_ConcurrentUpdates проверяет конкурентное обновление шаблонов.
func TestTemplateService_ConcurrentUpdates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	logger := zerolog.Nop()
	cfg := &config.Config{MaxTemplateSize: 100000}
	repo := NewMockTemplateRepository()
	svc := NewTemplateService(repo, &logger, cfg)

	ctx := context.Background()

	// Создаём шаблон
	createReq := &CreateTemplateRequest{
		UserID:  "user-concurrent-update",
		Name:    "Concurrent Update Test",
		Channel: model.ChannelEmail,
		Type:    model.TypeMonitorDown,
		Body:    "Original body",
	}

	created, err := svc.CreateTemplate(ctx, createReq)
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	numGoroutines := 5
	var wg sync.WaitGroup

	// Конкурентно обновляем шаблон
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			updateReq := &UpdateTemplateRequest{
				Name: fmt.Sprintf("Updated Name %d", idx),
				Body: fmt.Sprintf("Updated Body %d", idx),
			}

			_, err := svc.UpdateTemplate(ctx, created.ID, updateReq)
			if err != nil {
				t.Errorf("concurrent update %d failed: %v", idx, err)
			}
		}(i)
	}

	wg.Wait()

	// Проверяем, что шаблон обновлён (хоть и последним обновлением)
	updated, err := svc.GetTemplate(ctx, created.ID)
	if err != nil {
		t.Fatalf("failed to get updated template: %v", err)
	}

	if updated.Name == "Concurrent Update Test" {
		t.Error("template should have been updated at least once")
	}
}

// TestTemplateService_ConcurrentReads проверяет конкурентное чтение шаблонов.
func TestTemplateService_ConcurrentReads(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	logger := zerolog.Nop()
	cfg := &config.Config{MaxTemplateSize: 100000}
	repo := NewMockTemplateRepository()
	svc := NewTemplateService(repo, &logger, cfg)

	ctx := context.Background()

	// Создаём несколько шаблонов
	numTemplates := 5
	for i := 0; i < numTemplates; i++ {
		req := &CreateTemplateRequest{
			UserID:  "user-concurrent-read",
			Name:    fmt.Sprintf("Read Test %d", i),
			Channel: model.ChannelEmail,
			Type:    model.TypeMonitorDown,
			Body:    fmt.Sprintf("Body %d", i),
		}
		if _, err := svc.CreateTemplate(ctx, req); err != nil {
			t.Fatalf("failed to create template %d: %v", i, err)
		}
	}

	// Получаем список всех шаблонов
	allTemplates, _, err := svc.ListTemplates(ctx, ListFilter{
		UserID: "user-concurrent-read",
	})
	if err != nil {
		t.Fatalf("failed to list templates: %v", err)
	}

	numGoroutines := 10
	var wg sync.WaitGroup

	// Конкурентно читаем каждый шаблон
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			// Читаем случайный шаблон
			tmplIdx := idx % len(allTemplates)
			_, err := svc.GetTemplate(ctx, allTemplates[tmplIdx].ID)
			if err != nil {
				t.Errorf("concurrent read %d failed: %v", idx, err)
			}
		}(i)
	}

	wg.Wait()
}
