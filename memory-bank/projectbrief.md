# Project Brief

**Project:** Monitor Service
**Type:** Distributed Monitoring & Alerting Platform
**Status:** Active Development (2026-03-27)
**Architecture:** Microservices with gRPC

## Core Purpose

Система мониторинга HTTP/HTTPS эндпоинтов с алертингом и OAuth авторизацией.

## Key Capabilities

1. **Monitoring Service** - Периодические проверки доступности эндпоинтов
2. **Alert Service** - Управление алертами и каналами оповещений
3. **Auth Service** - OAuth авторизация и управление пользователями
4. **API Gateway** - Единая точка входа для всех сервисов

## Current Scope

- Мониторинг статуса (UP, DOWN, DEGRADED)
- Расчет uptime и детекция инцидентов
- Алертинг через Telegram, Email, Webhook
- Trigger логика (consecutive failures, flapping detection)
- Rate limiting и alert storm protection
- Escalation по приоритетам каналов
- OAuth 2.0 интеграция

## Out of Scope

- Мониторинг не-HTTP протоколов (TCP, ICMP, etc.)
- Synthetic transactions
- APM (Application Performance Monitoring)
- Distributed tracing beyond Jaeger integration
- Система биллинга (proto существует, но не реализовано)
