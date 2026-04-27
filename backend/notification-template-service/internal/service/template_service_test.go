package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/raul/monitor/backend/notification-template-service/internal/config"
	"github.com/raul/monitor/backend/notification-template-service/internal/model"
)

var ErrTemplateNotFound = errors.New("template not found")

// MockTemplateRepository мок репозитория для тестов.
type MockTemplateRepository struct {
	mu        sync.Mutex
	templates map[string]*model.Template
	createErr error
	getErr    error
	updateErr error
	deleteErr error
}

func NewMockTemplateRepository() *MockTemplateRepository {
	return &MockTemplateRepository{
		templates: make(map[string]*model.Template),
	}
}

func (m *MockTemplateRepository) Create(ctx context.Context, tmpl *model.Template) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	tmpl.ID = uuid.New().String()
	m.templates[tmpl.ID] = tmpl
	return nil
}

func (m *MockTemplateRepository) GetByID(ctx context.Context, id string) (*model.Template, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	tmpl, ok := m.templates[id]
	if !ok {
		return nil, ErrTemplateNotFound
	}
	snapshot := *tmpl
	return &snapshot, nil
}

func (m *MockTemplateRepository) GetDefault(ctx context.Context, userID string, channel model.TemplateChannel, tmplType model.TemplateType) (*model.Template, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, tmpl := range m.templates {
		if tmpl.UserID == userID && tmpl.Channel == channel && tmpl.Type == tmplType && tmpl.IsDefault {
			snapshot := *tmpl
			return &snapshot, nil
		}
	}
	return nil, ErrTemplateNotFound
}

func (m *MockTemplateRepository) List(ctx context.Context, filter ListFilter) ([]*model.Template, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*model.Template, 0, len(m.templates))
	for _, tmpl := range m.templates {
		result = append(result, tmpl)
	}
	return result, len(result), nil
}

func (m *MockTemplateRepository) Update(ctx context.Context, tmpl *model.Template) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.templates[tmpl.ID] = tmpl
	return nil
}

func (m *MockTemplateRepository) Delete(ctx context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.templates, id)
	return nil
}

func (m *MockTemplateRepository) SetDefault(ctx context.Context, id, userID string) error {
	return nil
}

func (m *MockTemplateRepository) Exists(ctx context.Context, id string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.templates[id]
	return ok, nil
}

func TestTemplateService_CreateTemplate(t *testing.T) {
	logger := zerolog.Nop()
	cfg := &config.Config{MaxTemplateSize: 64 * 1024}

	tests := []struct {
		name    string
		req     *CreateTemplateRequest
		setup   func(*MockTemplateRepository)
		wantErr bool
	}{
		{
			name: "valid template",
			req: &CreateTemplateRequest{
				UserID:  "user-1",
				Name:    "Test Template",
				Body:    "Monitor {{ .monitor_name }} is {{ .status }}",
				Channel: model.ChannelEmail,
				Type:    model.TypeMonitorDown,
			},
			setup:   func(m *MockTemplateRepository) {},
			wantErr: false,
		},
		{
			name: "invalid template - empty name",
			req: &CreateTemplateRequest{
				UserID:  "user-1",
				Name:    "",
				Body:    "Monitor {{ .monitor_name }} is {{ .status }}",
				Channel: model.ChannelEmail,
				Type:    model.TypeMonitorDown,
			},
			setup:   func(m *MockTemplateRepository) {},
			wantErr: true,
		},
		{
			name: "repository error",
			req: &CreateTemplateRequest{
				UserID:  "user-1",
				Name:    "Test Template",
				Body:    "Monitor {{ .monitor_name }} is {{ .status }}",
				Channel: model.ChannelEmail,
				Type:    model.TypeMonitorDown,
			},
			setup: func(m *MockTemplateRepository) {
				m.createErr = errors.New("database error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockTemplateRepository()
			tt.setup(repo)

			svc := NewTemplateService(repo, &logger, cfg)
			_, err := svc.CreateTemplate(context.Background(), tt.req)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateTemplate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTemplateService_DeleteTemplate(t *testing.T) {
	logger := zerolog.Nop()
	cfg := &config.Config{}

	tests := []struct {
		name    string
		id      string
		setup   func(*MockTemplateRepository)
		wantErr bool
		errMsg  string
	}{
		{
			name: "delete user template",
			id:   "test-id",
			setup: func(m *MockTemplateRepository) {
				m.templates["test-id"] = &model.Template{
					ID:       "test-id",
					Name:     "Test",
					Body:     "Test",
					Channel:  model.ChannelEmail,
					Type:     model.TypeMonitorDown,
					IsSystem: false,
				}
			},
			wantErr: false,
		},
		{
			name:    "template not found",
			id:      "non-existent",
			setup:   func(m *MockTemplateRepository) {},
			wantErr: true,
		},
		{
			name: "cannot delete system template",
			id:   "system-id",
			setup: func(m *MockTemplateRepository) {
				m.templates["system-id"] = &model.Template{
					ID:       "system-id",
					Name:     "System Template",
					Body:     "Test",
					Channel:  model.ChannelEmail,
					Type:     model.TypeMonitorDown,
					IsSystem: true,
				}
			},
			wantErr: true,
			errMsg:  "cannot be deleted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockTemplateRepository()
			tt.setup(repo)

			svc := NewTemplateService(repo, &logger, cfg)
			err := svc.DeleteTemplate(context.Background(), tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteTemplate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !containsString(err.Error(), tt.errMsg) {
					t.Errorf("DeleteTemplate() error = %v, should contain %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestTemplateService_CloneTemplate(t *testing.T) {
	logger := zerolog.Nop()
	cfg := &config.Config{}

	t.Run("clone template successfully", func(t *testing.T) {
		repo := NewMockTemplateRepository()
		original := &model.Template{
			ID:      "original-id",
			Name:    "Original",
			Body:    "Monitor {{ .monitor_name }} is {{ .status }}",
			Channel: model.ChannelEmail,
			Type:    model.TypeMonitorDown,
			Format:  model.FormatHTML,
		}
		repo.templates["original-id"] = original

		svc := NewTemplateService(repo, &logger, cfg)
		clone, err := svc.CloneTemplate(context.Background(), "original-id", "user-1", "Clone")

		if err != nil {
			t.Fatalf("CloneTemplate() error = %v", err)
		}

		if clone.Name != "Clone" {
			t.Errorf("CloneTemplate() name = %v, want %v", clone.Name, "Clone")
		}

		if clone.ParentID == nil || *clone.ParentID != "original-id" {
			t.Errorf("CloneTemplate() parentID = %v, want %v", clone.ParentID, "original-id")
		}

		if clone.IsSystem {
			t.Error("CloneTemplate() clone should not be system")
		}
	})
}

func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}
