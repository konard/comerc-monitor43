@epic=03_dashboard
@user_story=03_03_realtime_updates
# Description: Обновление дашборда в реальном времени

Feature: Обновления в реальном времени
  Как пользователь
  Я хочу видеть изменения статуса мониторов в реальном времени
  Чтобы быстро реагировать на проблемы

  @use_case=uc_03_03_01
  @critical
  Scenario: WebSocket подключение к дашборду
    Given пользователь открывает дашборд
    Then устанавливается WebSocket соединение
    And отображается индикатор "Подключено"
    And индикатор имеет зеленый цвет
    And мониторинг доступности соединения активен

  @use_case=uc_03_03_01a
  @validation
  Scenario: Ошибка подключения WebSocket a
    Given пользователь авторизован в системе
    And открывает дашборд
    When сервер WebSocket недоступен
    Then возвращается ошибка с кодом "WEBSOCKET_CONNECTION_FAILED"
    And ответ содержит структуру:
      | error.code       | WEBSOCKET_CONNECTION_FAILED |
      | error.message    | Failed to establish WebSocket connection |
      | error.details    | {"endpoint": "wss://api.example.com/ws", "reason": "connection_refused"} |
      | error.request_id | <uuid> |
    And ошибка логируется в audit log
    And система автоматически пытается переподключиться

  @use_case=uc_03_03_01b
  @security
  Scenario: Неавторизованный WebSocket доступ
    Given пользователь не авторизован в системе
    When пользователь пытается установить WebSocket соединение
    Then WebSocket соединение отклоняется
    And возвращается ошибка с кодом "UNAUTHORIZED"
    And ответ содержит структуру:
      | error.code       | UNAUTHORIZED |
      | error.message    | Authorization required for WebSocket connection |
      | error.request_id | <uuid> |
    And попытка подключения логируется в audit log как unauthorized

  @use_case=uc_03_03_01c
  @security
  Scenario: Недостаточно прав для real-time обновлений
    Given пользователь авторизован с ролью "viewer"
    And монитор принадлежит другому тенанту
    When пользователь пытается подписаться на обновления монитора
    Then возвращается ошибка с кодом "INSUFFICIENT_PERMISSIONS"
    And ответ содержит структуру:
      | error.code       | INSUFFICIENT_PERMISSIONS |
      | error.message    | Insufficient permissions to subscribe to updates |
      | error.details    | {"monitor_id": "API Service", "required_permission": "monitor:subscribe"} |
      | error.request_id | <uuid> |
    And попытка подписки логируется в audit log как access_denied

  @use_case=uc_03_03_01d
  Scenario: Разрыв соединения
    Given WebSocket соединение активно
    And индикатор подключения отображает "Подключено"
    When происходит разрыв соединения
    Then отображается индикатор "Соединение потеряно"
    And индикатор имеет красный цвет
    And система автоматически пытается восстановить соединение
    And попытки восстановления происходят с экспоненциальной задержкой
    And пользователь может видеть количество попыток переподключения

  @use_case=uc_03_03_01e
  Scenario: Автоматическое восстановление соединения
    Given WebSocket соединение было разорвано
    And система пытается восстановить соединение
    When соединение успешно восстановлено
    Then индикатор изменяется на "Подключено"
    And производится полная перезагрузка данных мониторов
    And возвращается подтверждение "CONNECTION_RESTORED"
    And ответ содержит структуру:
      | error.code       | CONNECTION_RESTORED |
      | error.message    | WebSocket connection restored successfully |
      | error.request_id | <uuid> |
    And полное обновление данных завершено

  @use_case=uc_03_03_01f
  Scenario: Отключение обновлений при уходе со страницы
    Given WebSocket соединение активно
    When пользователь закрывает вкладку браузера
    Then WebSocket соединение корректно закрывается
    And сервер освобождает ресурсы соединения
    And завершается подписка на обновления

  @use_case=uc_03_03_01g
  @integration
  Scenario: WebSocket сервер падает во время активной сессии
    Given пользователь имеет активное WebSocket соединение
    And пользователь просматривает графики в реальном времени
    When WebSocket сервер падает
    Then соединение разрывается
    And отображается индикатор "Соединение потеряно"
    And система автоматически пытается восстановить соединение
    And попытки восстановления происходят с экспоненциальной задержкой
    And данные мониторов сохраняются в локальном cache
    And ошибка логируется в audit log

  @use_case=uc_03_03_01h
  @fallback
  Scenario: Fallback на polling при недоступности WebSocket
    Given пользователь открывает дашборд
    And WebSocket сервер недоступен
    When пользователь просматривает мониторы
    Then система автоматически переключается на polling режим
    And опрос данных происходит каждые "30" секунд
    And возвращается предупреждение "LIMITED_FUNCTIONALITY_MODE"
    And ответ содержит структуру:
      | error.code       | LIMITED_FUNCTIONALITY_MODE |
      | error.message    | WebSocket unavailable, using polling mode |
      | error.details    | {"mode": "polling", "interval_seconds": 30} |
      | error.request_id | <uuid> |
    And индикатор подключения отображает "Polling"
    And попытки восстановления WebSocket происходят в фоне
    And система возвращается к WebSocket при восстановлении

  @use_case=uc_03_03_01i
  @fallback
  Scenario: Gradual degradation при проблемах с WebSocket
    Given пользователь имеет активное WebSocket соединение
    And соединение становится нестабильным
    When происходит более "3" разрывов за минуту
    Then система автоматически переключается на polling
    And интервал polling составляет "15" секунд
    And возвращается предупреждение "CONNECTION_UNSTABLE"
    And ответ содержит структуру:
      | error.code       | CONNECTION_UNSTABLE |
      | error.message    | WebSocket connection unstable, switched to polling mode |
      | error.details    | {"mode": "polling", "interval_seconds": 15, "disconnections_per_minute": 3} |
      | error.request_id | <uuid> |
    And система продолжает попытки восстановления WebSocket
    And статистика разрывов логируется в audit log
    And пользователь может вручную выбрать режим подключения

  @use_case=uc_03_03_01j
  @security
  Scenario: Перехват WebSocket соединения злоумышленником
    Given пользователь авторизован в системе
    And имеет активное WebSocket соединение
    When злоумышленник пытается перехватить connection ID
    Then сервер валидирует WebSocket connection при каждом сообщении
    And при обнаружении перехвата соединение разрывается
    And попытка перехвата логируется как security_breach
    And пользователь получает уведомление о подозрительной активности
    And IP адрес злоумышленника блокируется

  @use_case=uc_03_03_01k
  @security
  Scenario: Защита от инъекции через WebSocket
    Given пользователь имеет активное WebSocket соединение
    When злоумышленник отправляет вредоносный payload через WebSocket
    Then сервер валидирует каждое incoming сообщение
    And при обнаружении инъекции соединение разрывается
    And попытка атаки логируется как security_breach
    And сообщение не выполняется на сервере
    And пользователь уведомляется о попытке атаки

  @use_case=uc_03_03_01l
  @offline_mode
  Scenario: Работа в offline режиме с локальным cache
    Given пользователь просматривает дашборд
    And данные мониторов закэшированы локально
    When интернет-соединение теряется
    Then отображаются последние кэшированные данные
    And показывается индикатор "Offline mode"
    And данные доступны только для чтения
    And пользователь видит timestamp последнего обновления
    And попытки reconnect происходят в фоне
    And при восстановлении соединения данные синхронизируются

  @use_case=uc_03_03_01m
  @offline_mode
  Scenario: Синхронизация данных после восстановления соединения
    Given пользователь работал в offline режиме
    And локальный кэш содержит данные "5" минутной давности
    When интернет-соединение восстанавливается
    Then происходит полная синхронизация данных
    And загружаются все изменения за период offline
    And UI обновляется с актуальными данными
    And возвращается подтверждение "DATA_SYNCHRONIZED"
    And ответ содержит структуру:
      | error.code       | DATA_SYNCHRONIZED |
      | error.message    | Data synchronized after offline mode |
      | error.details    | {"offline_duration_minutes": 5} |
      | error.request_id | <uuid> |
    And изменения применяются с сохранением пользовательских действий
    And синхронизация логируется в audit log

  @use_case=uc_03_03_01n
  @bandwidth
  Scenario: Оптимизация обновлений при медленном соединении
    Given пользователь имеет медленное соединение (< 1 Mbps)
    And WebSocket соединение активно
    When поступают обновления для "50" мониторов
    Then данные обновлений сжимаются перед отправкой
    And бинарные данные графики не отправляются автоматически
    And пользователь может запросить графики вручную
    And приоритет обновлений: критические > обычные
    And показывается индикатор "Медленное соединение"
    And пользователь может выбрать "lightweight mode"

  @use_case=uc_03_03_01o
  @bandwidth
  Scenario: Lightweight режим для bandwidth-constrained устройств
    Given пользователь использует устройство с ограниченным bandwidth
    And пользователь авторизован в системе
    When пользователь выбирает "lightweight mode"
    Then real-time обновления ограничиваются
    And графики не загружаются автоматически
    And изображения оптимизируются по размеру
    And данные передаются в сжатом формате
    And частота обновлений уменьшается до "1" раза в минуту
    And пользователь может вернуться в normal mode

  @use_case=uc_03_03_01p
  @high_frequency
  Scenario: Обработка очень высокочастотных обновлений (1000+ updates/sec)
    Given пользователь имеет монитор с интервалом проверки "100ms"
    And WebSocket соединение активно
    And поступает более "1000" обновлений в секунду
    Then обновления группируются в пакеты по "100" штук
    And UI обновляется не чаще чем "30 times per second"
    And пакеты обновлений применяются атомарно
    And нагрузка на CPU браузера ограничена
    And память UI контролируется (max 100MB)
    And никакие данные не теряются
    And показывается индикатор "Высокая частота обновлений"

  @use_case=uc_03_03_01q
  @high_frequency
  Scenario: Adaptive throttling при нестабильной частоте обновлений
    Given пользователь имеет мониторы с различными интервалами проверки
    And WebSocket соединение активно
    When частота обновлений колеблется от "10" до "1000" в секунду
    Then система адаптивно регулирует частоту UI обновлений
    And при низкой частоте обновления выполняются в реальном времени
    And при средней частоте применяется группировка по "50" штук
    And при высокой частоте применяется группировка по "200" штук
    And адаптация происходит автоматически
    And пользователь видит текущий режим обновлений
    And throttling логируется для мониторинга производительности

  @use_case=uc_03_03_01r
  Scenario: Audit logging при WebSocket подключении
    Given пользователь авторизован в системе
    When пользователь устанавливает WebSocket соединение
    Then действие логируется в audit log:
      | action       | websocket_connected |
      | user_id      | <user_id> |
      | tenant_id    | <tenant_id> |
      | connection_id| <connection_id> |
      | success      | true |
    And log содержит timestamp и IP адрес
    And подключение логируется с уникальным session ID

  @use_case=uc_03_03_01s
  Scenario: Audit logging при отключении WebSocket
    Given пользователь имеет активное WebSocket соединение
    When пользователь закрывает вкладку браузера
    Then действие логируется в audit log:
      | action       | websocket_disconnected |
      | user_id      | <user_id> |
      | tenant_id    | <tenant_id> |
      | connection_id| <connection_id> |
      | reason       | client_closed |
      | success      | true |
    And log содержит длительность сессии

  @use_case=uc_03_03_02
  Scenario: Real-time обновление статуса монитора
    Given на дашборде отображается монитор "API Service" со статусом "UP"
    And пользователь просматривает страницу дашборда
    When статус монитора меняется на "DOWN"
    Then дашборд автоматически обновляется через WebSocket
    And статус монитора изменяется на "DOWN" в UI
    And монитор выделяется красным цветом
    And показывается уведомление об изменении статуса
    And обновляется timestamp последней проверки

  @use_case=uc_03_03_02a
  Scenario: Получение обновления для DEGRADED статуса
    Given на дашборде отображается монитор "API Service" со статусом "UP"
    And для монитора настроен threshold response time = "5 seconds"
    When время ответа монитора превышает "5 seconds"
    Then дашборд автоматически обновляется через WebSocket
    And статус монитора изменяется на "DEGRADED"
    And монитор выделяется желтым цветом
    And показывается уведомление о деградации

  @use_case=uc_03_03_02b
  @state_transition
  Scenario: Изменение статуса монитора при просмотре графиков
    Given пользователь просматривает графики монитора "API Service" в реальном времени
    And монитор имеет статус "UP"
    And WebSocket соединение активно
    When статус монитора меняется на "DOWN"
    Then графики автоматически обновляются
    And добавляется новая точка данных с статусом "DOWN"
    And зона инцидента выделяется на графике красным цветом
    And пользователь получает уведомление об изменении статуса

  @use_case=uc_03_03_02c
  @state_transition
  Scenario: Переход монитора из DOWN в UP во время просмотра
    Given пользователь просматривает графики монитора "API Service"
    And монитор находится в статусе "DOWN"
    And на графике выделена зона инцидента
    When монитор восстанавливается и переходит в статус "UP"
    Then графики автоматически обновляются
    And зона инцидента завершается
    And длительность инцидента рассчитывается
    And пользователь получает уведомление о восстановлении

  @use_case=uc_03_03_02d
  @state_transition
  Scenario: Множественные изменения статуса за короткий период
    Given пользователь просматривает графики монитора "API Service"
    And WebSocket соединение активно
    When монитор последовательно меняет статусы:
      | статус перехода | timestamp     |
      | UP → DOWN       | 10:00:00      |
      | DOWN → UP       | 10:00:30      |
      | UP → DEGRADED   | 10:01:00      |
      | DEGRADED → UP   | 10:01:30      |
    Then каждое изменение отображается на графике
    And все точки данных добавляются корректно
    And зоны инцидентов выделяются соответствующими цветами
    And пользователь получает уведомления о каждом изменении

  @use_case=uc_03_03_02e
  @performance
  Scenario: High-frequency real-time обновления графиков
    Given пользователь имеет монитор "API Service" с интервалом проверки "10 seconds"
    And WebSocket соединение активно
    And пользователь просматривает графики в реальном времени
    When поступает более "50" обновлений в минуту
    Then обновления группируются в пакеты
    And UI обновляется не чаще чем "20 times per second"
    And графики обновляются плавно без лагов
    And никакие данные не теряются
    And производительность остается приемлемой

  @use_case=uc_03_03_03
  Scenario: Real-time добавление нового монитора
    Given пользователь просматривает графики в реальном времени
    And WebSocket соединение активно
    When пользователь добавляет новый монитор "Database Service"
    Then новый монитор появляется на дашборде в реальном времени
    And монитор добавляется в список без перезагрузки страницы
    And начинают поступать обновления для нового монитора
    And WebSocket подписка автоматически обновляется

  @use_case=uc_03_03_03a
  Scenario: Отображение графика response time (line chart)
    Given пользователь имеет монитор "API Service" с проверками за "24 hours"
    When пользователь открывает детальную страницу монитора
    Then отображается график response time:
      | тип        | название              |
      | line chart | Response Time (ms)     |
    And график содержит точки для каждой успешной проверки
    And ось X отображает время проверки
    And ось Y отображает время ответа в миллисекундах
    And график обновляется в реальном времени при новых проверках

  @use_case=uc_03_03_03b
  Scenario: Отображение графика uptime (bar chart)
    Given пользователь имеет монитор "API Service" с проверками за "24 hours"
    When пользователь открывает детальную страницу монитора
    Then отображается график uptime:
      | тип       | название    |
      | bar chart | Uptime %    |
    And график отображает процент uptime для каждого интервала
    And успешные проверки отображаются зеленым цветом
    And failed проверки отображаются красным цветом
    And DEGRADED проверки отображаются желтым цветом
    And график обновляется в реальном времени

  @use_case=uc_03_03_03c
  Scenario: Выбор zoom уровня для графиков
    Given пользователь просматривает графики монитора "API Service"
    And доступны zoom уровни:
      | уровень   | период      |
      | 1 hour    | 1h          |
      | 24 hours  | 24h         |
      | 7 days    | 7d          |
      | 30 days   | 30d         |
    When пользователь выбирает zoom уровень "24 hours"
    Then загружаются данные за последние "24 hours"
    And графики перерисовываются с новым периодом
    And выбранный zoom уровень сохраняется на сервере
    And при следующем открытии применяется сохраненный zoom

  @use_case=uc_03_03_03d
  Scenario: Адаптивная детализация графиков по zoom уровню
    Given пользователь просматривает графики монитора "API Service"
    When пользователь выбирает zoom уровень "1 hour"
    Then графики отображаются с максимальной детализацией
    And каждая точка данных представляет отдельную проверку
    When пользователь переключается на zoom уровень "30 days"
    Then графики отображаются с агрегацией данных
    And точки данных группируются по интервалам для производительности
    And агрегация сохраняет общую картину состояния

  @use_case=uc_03_03_03e
  Scenario: Выделение инцидентов на графике
    Given пользователь имеет монитор "API Service" с инцидентом:
      | начало      | конец        | статус   |
      | 2026-03-01 10:00 | 2026-03-01 10:15 | DOWN |
    When пользователь просматривает график за период инцидента
    Then на графике выделяется зона инцидента:
      | элемент        | значение              |
      | фон            | Красный, полупрозрачный|
      | границы        | От начала до конца     |
      | подпись        | "DOWN: 15 min"        |
    And зона инцидента кликабельна
    And при клике открываются детали инцидента

  @use_case=uc_03_03_03f
  Scenario: Выделение нескольких инцидентов на графике
    Given пользователь имеет монитор "API Service" с инцидентами:
      | начало             | конец              | статус   |
      | 2026-03-01 10:00   | 2026-03-01 10:15   | DOWN     |
      | 2026-03-03 14:30   | 2026-03-03 14:45   | DEGRADED |
      | 2026-03-05 08:00   | 2026-03-05 09:00   | DOWN     |
    When пользователь просматривает график за "7 days"
    Then на графике выделяются все зоны инцидентов
    And DOWN инциденты отображаются красным цветом
    And DEGRADED инциденты отображаются желтым цветом
    And каждая зона имеет подпись с длительностью

  @use_case=uc_03_03_03g
  Scenario: Интерактивность графиков
    Given пользователь просматривает график response time
    When пользователь наводит курсор на точку данных
    Then отображается tooltip с информацией:
      | поле          | значение                 |
      | timestamp     | 2026-03-01 10:05:30 UTC  |
      | status        | UP                       |
      | response time | 145ms                    |
      | response code | 200                      |
    And tooltip следует за курсором
    When пользователь кликает на точку данных
    Then открывается детальная информация о проверке

  @use_case=uc_03_03_03h
  Scenario: Real-time обновление графиков
    Given пользователь просматривает график монитора "API Service" в реальном времени
    And WebSocket соединение активно
    When поступает новая проверка монитора
    Then график автоматически обновляется
    And новая точка данных добавляется к графику
    And старые данные удаляются за пределами выбранного zoom уровня
    And анимация обновления плавная

  @use_case=uc_03_03_03i
  Scenario: Переключение между типами графиков
    Given пользователь просматривает детальную страницу монитора
    And отображаются оба типа графиков:
      | график         |
      | Response Time  |
      | Uptime %       |
    When пользователь переключается между графиками
    Then активный график выделяется визуально
    And неактивный график остается доступным
    And состояние каждого графика сохраняется независимо

  @use_case=uc_03_03_03j
  Scenario: Загрузка исторических данных при изменении zoom
    Given пользователь просматривает график с zoom уровнем "1 hour"
    And данные загружены за последний час
    When пользователь изменяет zoom на "7 days"
    Then отправляется запрос на сервер за данными за "7 days"
    And отображается индикатор загрузки
    And после загрузки графики перерисовываются
    And zoom уровень сохраняется на сервере

  @use_case=uc_03_03_03k
  Scenario: Обработка ошибок загрузки данных графиков
    Given пользователь запрашивает данные за период "30 days"
    When сервер возвращает ошибку
    Then отображается сообщение об ошибке
    And графики показывают последние доступные данные
    And предлагается повторить запрос
    And ошибка логируется для анализа

  @use_case=uc_03_03_03l
  Scenario: Отображение метрик на графике
    Given пользователь просматривает график response time за "24 hours"
    Then на графике отображаются вспомогательные линии:
      | метрика | описание                     |
      | P50     | Медиана времени ответа       |
      | P95     | 95-й перцентиль              |
      | P99     | 99-й перцентиль              |
    And линии имеют разные цвета
    And легенда объясняет каждую линию
    And линии можно скрыть через настройки графика

  @use_case=uc_03_03_03m
  @validation
  Scenario: Ошибка загрузки данных графиков m
    Given пользователь авторизован в системе
    And пользователь открывает детальную страницу монитора "API Service"
    When сервис графиков недоступен
    Then возвращается ошибка с кодом "CHART_DATA_LOAD_FAILED"
    And ответ содержит структуру:
      | error.code       | CHART_DATA_LOAD_FAILED |
      | error.message    | Failed to load chart data |
      | error.details    | {"monitor_id": "API Service", "chart_type": "response_time"} |
      | error.request_id | <uuid> |
    And ошибка логируется в audit log
    And предлагается повторить запрос

  @use_case=uc_03_03_03n
  @validation
  Scenario: Неверный zoom уровень графика n
    Given пользователь авторизован в системе
    And пользователь просматривает графики монитора "API Service"
    When пользователь выбирает zoom уровень "1000 years"
    Then возвращается ошибка с кодом "INVALID_ZOOM_LEVEL"
    And ответ содержит структуру:
      | error.code       | INVALID_ZOOM_LEVEL |
      | error.message    | Invalid chart zoom level |
      | error.details    | {"requested_zoom": "1000 years", "allowed_values": ["1h", "24h", "7d", "30d"]} |
      | error.request_id | <uuid> |
    And действие логируется в audit log как failed

  @use_case=uc_03_03_03o
  Scenario: Audit logging при изменении zoom уровня
    Given пользователь авторизован в системе
    And пользователь просматривает графики монитора "API Service"
    When пользователь изменяет zoom уровень на "7 days"
    Then действие логируется в audit log:
      | action       | chart_zoom_changed |
      | user_id      | <user_id> |
      | monitor_id   | API Service |
      | previous_zoom| 24h |
      | new_zoom     | 7d |
      | success      | true |
    And log содержит timestamp

  @use_case=uc_03_03_03p
  @integration
  Scenario: Таймаут загрузки данных графиков при изменении zoom p
    Given пользователь просматривает графики монитора "API Service"
    And пользователь изменяет zoom уровень на "30 days"
    When запрос данных превышает "30" секунд
    Then возвращается ошибка с кодом "CHART_DATA_LOAD_TIMEOUT"
    And ответ содержит структуру:
      | error.code       | CHART_DATA_LOAD_TIMEOUT |
      | error.message    | Chart data loading timeout |
      | error.details    | {"timeout": "30s", "zoom_level": "30d"} |
      | error.request_id | <uuid> |
    And предлагается выбрать меньший период
    And предыдущий zoom уровень восстанавливается
    And timeout логируется в audit log

  @use_case=uc_03_03_03q
  @integration
  Scenario: Недоступность cache сервиса для графиков
    Given пользователь просматривает графики монитора "API Service"
    When cache сервис недоступен
    Then данные графиков загружаются напрямую из базы данных
    And время загрузки увеличивается
    And возвращается предупреждение "CACHE_UNAVAILABLE"
    And ответ содержит структуру:
      | error.code       | CACHE_UNAVAILABLE |
      | error.message    | Cache service unavailable, loading directly from database |
      | error.details    | {"mode": "slow_loading"} |
      | error.request_id | <uuid> |
    And графики обновляются после загрузки данных
    And ошибка логируется в audit log

  @use_case=uc_03_03_03r
  @integration
  Scenario: Сервис генерации графиков недоступен
    Given пользователь просматривает детальную страницу монитора
    And отображаются графики response time и uptime
    When сервис генерации графиков недоступен
    Then графики не отображаются
    And возвращается предупреждение "CHARTS_TEMPORARILY_UNAVAILABLE"
    And ответ содержит структуру:
      | error.code       | CHARTS_TEMPORARILY_UNAVAILABLE |
      | error.message    | Chart generation service temporarily unavailable |
      | error.details    | {"monitor_data_available": true} |
      | error.request_id | <uuid> |
    And остальная информация о мониторе остается доступной
    And предлагается повторить попытку позже
    And ошибка логируется в audit log

  @use_case=uc_03_03_03s
  @integration
  Scenario: Несколько сервисов недоступны одновременно
    Given пользователь просматривает графики в реальном времени
    And недоступны сервисы:
      | сервис           |
      | WebSocket server |
      | cache service    |
      | chart service    |
    When пользователь пытается обновить данные
    Then WebSocket соединение разрывается
    And система переходит в polling режим с интервалом "30" секунд
    And графики не генерируются
    And возвращается предупреждение "SOME_FEATURES_UNAVAILABLE"
    And ответ содержит структуру:
      | error.code       | SOME_FEATURES_UNAVAILABLE |
      | error.message    | Multiple services are unavailable |
      | error.details    | {"unavailable_services": ["WebSocket server", "cache service", "chart service"]} |
      | error.request_id | <uuid> |
    And базовые данные загружаются из базы данных
    And ошибки логируются в audit log для каждого сервиса

  @use_case=uc_03_03_03t
  @boundary
  Scenario: Минимальный zoom уровень (1 hour)
    Given пользователь просматривает графики монитора "API Service"
    And доступные zoom уровни: "1h", "24h", "7d", "30d"
    When пользователь выбирает zoom уровень "1 hour"
    Then загружаются данные за последний час
    And графики отображаются с максимальной детализацией
    And каждая точка данных представляет отдельную проверку
    And графики обновляются в реальном времени
    And zoom уровень сохраняется на сервере

  @use_case=uc_03_03_03u
  @boundary
  Scenario: Максимальный zoom уровень (30 days)
    Given пользователь просматривает графики монитора "API Service"
    And доступные zoom уровни: "1h", "24h", "7d", "30d"
    When пользователь выбирает zoom уровень "30 days"
    Then загружаются данные за последние "30" дней
    And данные агрегируются по интервалам для производительности
    And точки данных группируются по часам
    And общая картина состояния сохраняется
    And графики обновляются при новых данных

  @use_case=uc_03_03_03v
  @boundary
  Scenario: Попытка выбора недопустимого zoom уровня v
    Given пользователь просматривает графики монитора "API Service"
    And доступные zoom уровни: "1h", "24h", "7d", "30d"
    When пользователь пытается выбрать zoom уровень "1 year"
    Then возвращается ошибка с кодом "INVALID_ZOOM_LEVEL"
    And ответ содержит структуру:
      | error.code       | INVALID_ZOOM_LEVEL |
      | error.message    | Invalid zoom level |
      | error.details    | {"requested": "1 year", "allowed": ["1h", "24h", "7d", "30d"]} |
      | error.request_id | <uuid> |
    And применяется предыдущий корректный zoom уровень
    And действие логируется в audit log как failed

  @use_case=uc_03_03_03w
  @boundary
  Scenario: Граница retention периода на графике
    Given пользователь имеет монитор "API Service" с данными за "30" дней
    And retention period составляет "30" дней
    When пользователь выбирает zoom уровень "30 days"
    Then отображаются все доступные данные
    And данные старше "30" дней отсутствуют
    And возвращается предупреждение "RETENTION_PERIOD_LIMIT"
    And ответ содержит структуру:
      | error.code       | RETENTION_PERIOD_LIMIT |
      | error.message    | Data older than retention period is not available |
      | error.details    | {"retention_days": 30} |
      | error.request_id | <uuid> |
    And предлагается экспортировать данные

  @use_case=uc_03_03_03x
  @boundary
  Scenario: Пустой результат при выборе zoom уровня
    Given пользователь имеет новый монитор "API Service"
    And монитор создан "5" минут назад
    When пользователь выбирает zoom уровень "24 hours"
    Then график отображается с пустыми данными
    And возвращается предупреждение "INSUFFICIENT_DATA"
    And ответ содержит структуру:
      | error.code       | INSUFFICIENT_DATA |
      | error.message    | Insufficient data to display for selected zoom level |
      | error.details    | {"requested_zoom": "24h", "available_data_minutes": 5} |
      | error.request_id | <uuid> |
    And предлагается выбрать меньший zoom уровень
    And график автоматически переключается на "1 hour"

  @use_case=uc_03_03_03y
  @performance
  Scenario: Загрузка и отрисовка большого объема данных на графике
    Given пользователь имеет монитор с "10000" проверок за "24 hours"
    When пользователь выбирает zoom уровень "24 hours"
    Then данные агрегируются для производительности
    And точки данных группируются по минутам
    And график отрисовывается не более чем за "3" секунды
    And пользователь может взаимодействовать с графиком
    And tooltips и интерактивность работают плавно

  @use_case=uc_03_03_03z
  @performance
  Scenario: Экспорт графиков для большого dataset
    Given пользователь имеет монитор с "10000" проверок за "30" дней
    And пользователь имеет права на экспорт данных
    When пользователь запрашивает экспорт графиков в формате "PNG"
    Then изображение генерируется асинхронно
    And пользователь получает уведомление когда изображение готово
    And изображение содержит все графики с выбранным zoom уровнем
    And разрешение изображения составляет "1920x1080"
    And ссылка на изображение доступна "24" часа

  @use_case=uc_03_03_03aa
  @performance
  Scenario: Одновременное просмотр графиков несколькими пользователями
    Given тенант имеет "50" пользователей
    And все пользователи одновременно просматривают графики монитора "API Service"
    And WebSocket соединения активны для всех пользователей
    Then все подключения обрабатываются успешно
    And обновления доставляются всем пользователям
    And время доставки не превышает "1" секунду
    And сервер масштабирует нагрузку
    And latency остается приемлемой
