# Product Context

## Why This Project Exists

Мониторинг критически важных сервисов — фундаментальная потребность любой инфраструктуры. Существующие решения либо дорогие (DataDog, New Relic), либо сложные в настройке (Prometheus+Grafana stack). Monitor Service provides:

- **Simple setup** — Docker Compose для локальной разработки
- **Modern stack** — gRPC, event-driven architecture
- **Extensible** — Легко добавлять новые каналы алертинга
- **Self-hosted** — Полный контроль над данными

## Problems It Solves

1. **Downtime Detection** - Быстрое обнаружение недоступности сервисов
2. **Alert Fatigue** - Flapping detection, rate limiting, acknowledge/mute
3. **Multi-channel Notifications** - Telegram, Email, Webhook из коробки
4. **Incident Tracking** - История проверок и инцидентов
5. **Team Access Control** - OAuth интеграция для управления доступом

## User Experience Goals

- **Developer Experience** - Простота добавления новых мониторов через gRPC API
- **Ops Experience** - Понятные статусы, actionable алерты
- **Team Experience** - Совместная работа с алертами (acknowledge, escalate)

## Success Metrics

- **Reliability** - 99.9% uptime самого monitor-service
- **Performance** - Проверки < 1s для большинства эндпоинтов
- **Alert Accuracy** - Minimize false positives через flapping detection
- **Time to Alert** - < 30s от детекции до доставки уведомления
