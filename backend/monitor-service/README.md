# Monitor Service

Сервис мониторинга HTTP/HTTPS эндпоинтов с поддержкой окон технического обслуживания.

## Возможности

### Мониторинг
- Создание и управление HTTP/HTTPS мониторами
- Настраиваемые интервалы проверки (30с - 1ч)
- Проверка status code, response time, body pattern
- Рабочие часы и дни
- Grace period для определения DOWN статуса
- DEGRADED статус для медленных ответов

### Окна технического обслуживания (Maintenance Windows)
- Создание окон обслуживания для подавления алертов во время работ
- Поддержка разовых и повторяющихся окон (ежедневно, еженедельно, ежемесячно)
- Глобальные окна (для всех мониторов) и для конкретных мониторов
- Настройка паузы мониторинга и подавления алертов
- Автоматическая активация и завершение окон по расписанию
- Безопасный режим (safe mode) для критических проверок во время обслуживания

## API

### Maintenance Windows API

**CreateMaintenanceWindow** - Создание нового окна обслуживания

**UpdateMaintenanceWindow** - Обновление параметров окна

**ListMaintenanceWindows** - Получение списка окон

**CancelMaintenanceWindow** - Отмена активного или запланированного окна

**GetMaintenanceWindowHistory** - Получение истории окон

**DeleteMaintenanceWindow** - Удаление окна

## Статусы окон

- `SCHEDULED` - Запланировано
- `ACTIVE` - Активно
- `COMPLETED` - Завершено
- `CANCELLED` - Отменено
- `ORPHANED` - Осиротело

## Конфигурация

```bash
MAINTENANCE_MIN_DURATION_MINUTES=1
MAINTENANCE_MAX_DURATION_HOURS=24
MAINTENANCE_MAX_WINDOWS_FREE=5
MAINTENANCE_MAX_WINDOWS_PRO=20
MAINTENANCE_MAX_WINDOWS_ENTERPRISE=100
MAINTENANCE_MAX_MONITORS_PER_WINDOW=50
```

## Интеграции

- **Monitor Service**: Пауза и подавление алертов
- **Alert Service**: Mark alerts as suppressed
- **Reporting Service**: Исключение из SLA расчётов
