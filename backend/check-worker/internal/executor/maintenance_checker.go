package executor

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MaintenanceWindow представляет информацию об окне обслуживания
type MaintenanceWindow struct {
	ID              string
	MonitorID       string
	StartTime       time.Time
	EndTime         time.Time
	PauseMonitoring bool
	IsGlobal        bool
}

// IsActive проверяет, активно ли окно обслуживания сейчас
func (w *MaintenanceWindow) IsActive() bool {
	now := time.Now()
	return now.Equal(w.StartTime) || (now.After(w.StartTime) && now.Before(w.EndTime))
}

// AffectsMonitor проверяет, влияет ли окно на монитор
func (w *MaintenanceWindow) AffectsMonitor(monitorID string) bool {
	if !w.IsActive() {
		return false
	}

	// Глобальные окна влияют на все мониторы
	if w.IsGlobal {
		return true
	}

	// Проверяем конкретный монитор
	return w.MonitorID == monitorID
}

// MaintenanceCache кэширует активные окна обслуживания
type MaintenanceCache struct {
	mu       sync.RWMutex
	windows  []MaintenanceWindow
	lastSync time.Time
	ttl      time.Duration
}

// NewMaintenanceCache создаёт новый кэш
func NewMaintenanceCache(ttl time.Duration) *MaintenanceCache {
	return &MaintenanceCache{
		windows: make([]MaintenanceWindow, 0),
		ttl:     ttl,
	}
}

// ShouldRefresh проверяет, нужно ли обновить кэш
func (c *MaintenanceCache) ShouldRefresh() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.lastSync.IsZero() {
		return true
	}

	return time.Since(c.lastSync) > c.ttl
}

// Update обновляет кэш новыми данными
func (c *MaintenanceCache) Update(windows []MaintenanceWindow) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.windows = windows
	c.lastSync = time.Now()
}

// IsMonitorInMaintenance проверяет, находится ли монитор на обслуживании
func (c *MaintenanceCache) IsMonitorInMaintenance(monitorID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, window := range c.windows {
		if window.AffectsMonitor(monitorID) {
			return true
		}
	}

	return false
}

// GetActiveWindowForMonitor возвращает активное окно для монитора
func (c *MaintenanceCache) GetActiveWindowForMonitor(monitorID string) *MaintenanceWindow {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, window := range c.windows {
		if window.AffectsMonitor(monitorID) {
			return &window
		}
	}

	return nil
}

// MaintenanceChecker проверяет окна обслуживания
type MaintenanceChecker struct {
	cache      *MaintenanceCache
	client     MaintenanceClient
	refreshMux sync.Mutex
}

// MaintenanceClient предоставляет интерфейс для получения maintenance windows
type MaintenanceClient interface {
	GetActiveMaintenanceWindows(ctx context.Context) ([]MaintenanceWindow, error)
}

// NewMaintenanceChecker создаёт новый checker
func NewMaintenanceChecker(client MaintenanceClient, cacheTTL time.Duration) *MaintenanceChecker {
	return &MaintenanceChecker{
		cache:  NewMaintenanceCache(cacheTTL),
		client: client,
	}
}

// ShouldSkipCheck проверяет, следует ли пропустить проверку из-за maintenance
func (mc *MaintenanceChecker) ShouldSkipCheck(ctx context.Context, monitorID string) (bool, *MaintenanceWindow, error) {
	// Проверяем кэш
	if mc.cache.ShouldRefresh() {
		refreshErr := mc.refreshCache(ctx)
		_ = refreshErr // продолжаем с устаревшими данными, если обновление не удалось
	}

	// Проверяем наличие активного окна
	if window := mc.cache.GetActiveWindowForMonitor(monitorID); window != nil {
		return true, window, nil
	}

	return false, nil, nil
}

// refreshCache обновляет кэш с защитой от одновременных обновлений
func (mc *MaintenanceChecker) refreshCache(ctx context.Context) error {
	// Проверяем, не обновляется ли кэш уже
	if !mc.refreshMux.TryLock() {
		return nil // Уже обновляется другим goroutine
	}
	defer mc.refreshMux.Unlock()

	// Ещё раз проверяем, не устарел ли кэш (мог обновиться пока ждали блокировку)
	if !mc.cache.ShouldRefresh() {
		return nil
	}

	windows, err := mc.client.GetActiveMaintenanceWindows(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch maintenance windows: %w", err)
	}

	mc.cache.Update(windows)
	return nil
}

// ForceRefresh принудительно обновляет кэш
func (mc *MaintenanceChecker) ForceRefresh(ctx context.Context) error {
	mc.refreshMux.Lock()
	defer mc.refreshMux.Unlock()

	windows, err := mc.client.GetActiveMaintenanceWindows(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch maintenance windows: %w", err)
	}

	mc.cache.Update(windows)
	return nil
}
