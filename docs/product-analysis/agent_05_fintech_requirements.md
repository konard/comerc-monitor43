# Агент 5: Fintech требования

## Требования к надёжности

### SLA стандарты

| Availability | Yearly Downtime | Monthly Downtime | Weekly Downtime | Daily Downtime |
|--------------|-----------------|------------------|-----------------|----------------|
| **99%** | 3.65 дней | 7.2 часов | 1.68 часов | 14.4 минут |
| **99.9%** | 8.76 часов | 43.8 минут | 10.1 минут | 1.44 минут |
| **99.99%** | 52.56 минут | 4.38 минут | 1 минута | 8.66 секунд |
| **99.999%** | 5.26 минут | 26.3 секунд | 6 секунд | 0.87 секунд |

### Расчёт downtime
- **99%** = 1% downtime (1 - 99/100)
- **99.9%** = 0.1% downtime (три девятки)
- **99.99%** = 0.01% downtime (четыре девятки)
- **99.999%** = 0.001% downtime (пять девяток)

### Для fintech ожидается
- **Критические системы (платежи, переводы)**: 99.99% - 99.999%
- **Мобильный банкинг**: 99.9% - 99.99%
- **Информационные сервисы**: 99% - 99.9%

## Регуляторные требования ЦБ РФ

### К доступности
- **Переход на отечественное ПО**: К 2024 году критическая информационная инфраструктура должна перейти на российское программное обеспечение
- **Цифровой рубль**: К июлю 2025 года все крупные банки должны обеспечить работу с цифровым рублём
- **Технологический суверенитет**:
  - 2022: минимум 50% отечественного ПО в системах
  - 2024: переход на российское ПО для критической инфраструктуры
  - 2025: переход на отечественное IT оборудование

### К мониторингу
- **Май 2024**: ЦБ РФ опубликовал руководство по порядку регуляторного контроля и мониторинга соблюдения кредитными и некредитными организациями планов перехода на российские ИКТ-решения
- Требования к непрерывности бизнес-процессов
- Мониторинг аварийных ситуаций в реальном времени

### К безопасности
- **СТО БР ИБББ** (Стандарт организации Банка России "Информационная безопасность банковских систем бизнеса") - базовый стандарт информационной безопасности
- **PCI DSS** - для платёжных систем (требование 10: логирование и мониторинг)
- Требования к защите персональных данных (152-ФЗ)

## Особенности мониторинга в fintech

### API monitoring

#### Критичные метрики
- **Request latency** (P50, P95, P99 percentiles)
- **Error rate** - процент неудачных запросов
- **Throughput** - количество транзакций в секунду (TPS)
- **Availability** - доступность API
- **Transaction success rate** - успешность транзакций

#### Практики
- **Unified Asset Governance**: активная регистрация + пассивный анализ + активное сканирование
- **Shadow API Detection**: обнаружение неизвестных API (кейс: банк нашёл 200+ неизвестных API за 3 месяца)
- **Протоколы**: REST, gRPC, GraphQL
- **Real-time asset tracking**: обнаружение новых API в течение 1 минуты

#### Производительность
- **High Concurrency**: поддержка 1M+ QPS
- **Low Latency**: накладные расходы безопасности ≤5%; средняя задержка <3ms
- **Accuracy**: точность идентификации ≥95%; ложные срабатывания ≤0.1%

### Transaction monitoring

#### Real-time Transaction Monitoring
- **Instant Analysis**: предопределённые правила + ML алгоритмы
- **Pattern Detection**: отклонения от нормального поведения (необычные суммы, высокая частота)
- **Behavioral Baselines**: ML-базлайны для интерфейсов, систем и пользователей
- **3D Profiling**: профили по измерениям интерфейс/система/пользователь

#### Security & Fraud Prevention
- **Multi-layered Protection**: OAuth 2.0/JWT + RBAC/ABAC
- **Rate Limiting**: контроль частоты для защиты от скрейпинга и автоматических атак
- **Dynamic Masking**: маскирование чувствительных данных в реальном времени
- **Automated Blocking**: <60 мин MTTR для блокировки угроз

#### Reconciliation
- **Unique transaction IDs** для сверки внутренних записей с записями провайдеров
- **Automated reconciliation systems**
- **Structured logging** для быстрой идентификации проблем

## Incident Response Metrics

| Метрика | Описание | Target для fintech |
|---------|----------|-------------------|
| **MTTD** | Mean Time To Detect | <5 минут |
| **MTTA** | Mean Time To Acknowledge | <10 минут |
| **MTTC** | Mean Time To Contain | <30 минут |
| **MTTR** | Mean Time To Recover | <60 минут |

### Technologies to Improve Metrics
- **SIEM** (Security Information and Event Management)
- **SOAR** (Security Orchestration, Automation, and Response)
- **Threat intelligence**
- **Real-time monitoring systems**

## PCI DSS Requirement 10: Logging and Monitoring

### Ключевые требования
- **10.1**: Реализовать меры логирования для связывания всех доступов с пользователями
- **10.2**: Автоматизированные аудит-трейлы для реконструкции событий
- **10.5**: Защита аудит-трейлов от изменения
- **10.6**: Ежедневный обзор логов (daily log review)
- **10.7**: Хранение логов минимум **1 год**, с **3 месяцами** онлайн-доступа

### Modern Implementation (PCI DSS 4.0.1)
- Centralized logging
- Enhanced monitoring for containerized environments (Kubernetes)
- Cloud-native logging tools (Azure Monitor)
- Time synchronization across all systems

## Чек-лист требований для нашего продукта

### 1. SLA и availability
- [ ] Поддержка определения SLO/SLA с уровнями 99%, 99.9%, 99.99%, 99.999%
- [ ] Автоматический расчёт error budget
- [ ] Отслеживание downtime по различным периодам (день/неделя/месяц/год)

### 2. API monitoring
- [ ] Мониторинг REST, gRPC, GraphQL API
- [ ] Измерение latency (P50, P95, P99)
- [ ] Отслеживание error rate и throughput
- [ ] Обнаружение shadow API
- [ ] Real-time alerts при нарушениях

### 3. Transaction monitoring
- [ ] Отслеживание транзакций от инициации до завершения
- [ ] Reconciliation с внешними системами
- [ ] Unique transaction IDs для трассировки
- [ ] Мониторинг success rate по типам транзакций

### 4. Security & Compliance
- [ ] Аудит-трейлы для всех административных действий
- [ ] Daily log review (автоматический)
- [ ] Хранение логов от 3 месяцев до 1 года
- [ ] Защита логов от изменения
- [ ] Time synchronization

### 5. Incident Response
- [ ] MTTD мониторинг
- [ ] MTTR tracking
- [ ] Automated alerts
- [ ] Integration с SIEM/SOAR
- [ ] Post-incident review templates

### 6. Performance Requirements
- [ ] Поддержка 1M+ QPS (если применимо)
- [ ] Накладные расходы мониторинга ≤5%
- [ ] Latency мониторинга <3ms
- [ ] Точность детекции ≥95%

## Отличия от e-commerce

| Аспект | E-commerce | Fintech |
|--------|------------|---------|
| **SLA ожидания** | 99.9% (general) - 99.99% (peak) | 99.99% - 99.999% (critical systems) |
| **Критичные метрики** | Availability, Page Load Time, Conversion Rate, Checkout Success | Transaction Accuracy, Success Rate, Fraud Detection, Compliance |
| **Regulatory** | Consumer protection (GDPR, local privacy laws) | Heavy financial regulation (PCI DSS, Central Bank requirements, AML/KYC) |
| **Data sensitivity** | Customer behavior, purchase history | Financial data, personal identifiers, transaction records |
| **Failure impact** | Revenue loss, reputation damage | Financial loss + regulatory penalties + legal liability |
| **Monitoring focus** | User experience, uptime, performance | Security, compliance, transaction integrity, audit trails |
| **Log retention** | Days to months | 1 year (3 months available online per PCI DSS) |
| **Real-time requirements** | Important for UX | Critical for fraud detection and transaction monitoring |
| **Incident response MTTR** | Hours acceptable | Minutes required (<60 min target) |

## Источники

- [SLA Availability Standards & Downtime Calculations](https://www.amazonaws.cn/en-us/uptime/)
- [Fintech API Monitoring Best Practices](https://wallarm.com/)
- [Fintech vs E-commerce Monitoring Differences](https://www.imda.gov.sg/)
- [PCI DSS Requirement 10 Logging and Monitoring](https://www.pcisecuritystandards.org/)
- [SRE Reliability Practices for Financial Services](https://sre.google/)
- [Incident Response Metrics MTTD MTTR](https://www.ibm.com/topics/mean-time-to-detect)
- [Central Bank of Russia Digital Ruble 2024-2025](https://www.cbr.ru/)
- [UPI Payment Monitoring NPCI](https://www.npci.org.in/)
