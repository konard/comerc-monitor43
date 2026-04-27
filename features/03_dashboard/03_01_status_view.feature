@epic=03_dashboard
@user_story=03_01_status_view
# Description: Просмотр статуса всех мониторов на дашборде

Feature: Просмотр статуса мониторов
  Как пользователь
  Я хочу видеть статус всех своих мониторов на одном экране
  Чтобы быстро оценивать состояние всех отслеживаемых сервисов

  @use_case=uc_03_01_01
  Scenario: Отображение структуры дашборда
    Given пользователь авторизован в системе
    When пользователь открывает дашборд
    Then отображается структура страницы:
      | элемент      | описание                         |
      | Sidebar      | Навигация по разделам            |
      | Main content | Список мониторов со статусами    |
    And sidebar содержит разделы:
      | раздел           |
      | Мониторы         |
      | Инциденты        |
      | Настройки        |
      | Биллинг          |

  @use_case=uc_03_01_14a
  @validation
  Scenario: Ошибка загрузки дашборда a
    Given пользователь авторизован в системе
    And сервер мониторинга недоступен
    When пользователь открывает дашборд
    Then возвращается ошибка с кодом "DASHBOARD_LOAD_FAILED"
    And ответ содержит структуру:
      | error.code       | DASHBOARD_LOAD_FAILED |
      | error.message    | Failed to load dashboard data |
      | error.request_id | <uuid> |
    And ошибка логируется в audit log

  @use_case=uc_03_01_14b
  @security
  Scenario: Неавторизованный доступ к дашборду
    Given пользователь не авторизован в системе
    When пользователь пытается открыть дашборд
    Then возвращается ошибка с кодом "UNAUTHORIZED"
    And ответ содержит структуру:
      | error.code       | UNAUTHORIZED |
      | error.message    | Authorization required |
      | error.request_id | <uuid> |
    And попытка доступа логируется в audit log как unauthorized

  @use_case=uc_03_01_14c
  @security
  Scenario: Недостаточно прав для просмотра мониторов
    Given пользователь авторизован с ролью "viewer"
    And пользователь пытается получить доступ к мониторам другого тенанта
    When пользователь запрашивает дашборд
    Then возвращается ошибка с кодом "INSUFFICIENT_PERMISSIONS"
    And ответ содержит структуру:
      | error.code       | INSUFFICIENT_PERMISSIONS |
      | error.message    | Insufficient permissions to access resource |
      | error.details    | {"required_permission": "monitors:read", "tenant_id": "<tenant_id>"} |
      | error.request_id | <uuid> |
    And попытка логируется в audit log как access_denied

  @use_case=uc_03_01_02
  Scenario: Audit logging при просмотре дашборда
    Given пользователь авторизован в системе
    When пользователь открывает дашборд
    Then действие логируется в audit log:
      | action       | dashboard_view |
      | user_id      | <user_id> |
      | tenant_id    | <tenant_id> |
      | success      | true |
      | filters      | {} |
    And log содержит timestamp и IP адрес

  @use_case=uc_03_01_03
  Scenario: Кросс-тенантная изоляция данных
    Given пользователь тенанта "A" авторизован в системе
    And тенант "B" имеет мониторы
    When пользователь тенанта "A" запрашивает дашборд
    Then отображаются только мониторы тенанта "A"
    And мониторы тенанта "B" не доступны в ответе
    And запрос не содержит данных тенанта "B" в response
    And попытка доступа изолируется на уровне сервиса

  @use_case=uc_03_01_04
  Scenario: Перехват WebSocket соединения
    Given пользователь авторизован в системе
    And открывает дашборд с WebSocket соединением
    When злоумышленник пытается перехватить WebSocket token
    Then сервер валидирует WebSocket connection token
    And при неверном token соединение отклоняется
    And попытка перехвата логируется в audit log как security_breach
    And IP адрес злоумышленника заносится в blacklist

  # Integration: Displays real-time monitor status from Epic 01 (Monitoring)
  # Integration: Shows active alerts from Epic 02 (Alerting)
  # See: Epic 01, uc_01_01_01; Epic 02, uc_02_02_11
  @use_case=uc_03_01_05
  Scenario: Просмотр всех статусов на дашборде
    Given пользователь имеет мониторы:
      | name         | url                          | status   | uptime_percent |
      | API Service  | https://api.example.com       | UP       | 99.9           |
      | Web Site     | https://example.com           | DOWN     | 95.2           |
      | Database     | https://db.example.com        | UP       | 100.0          |
      | Cache        | https://cache.example.com     | DEGRADED | 98.5           |
    When пользователь просматривает дашборд
    Then пользователь видит "4" монитора
    And каждый монитор отображается с визуальными индикаторами:
      | монитор     | статус   | цвет      | иконка | progress_bar |
      | API Service | UP       | Зеленый   | ✓      | 99.9%        |
      | Web Site    | DOWN     | Красный   | ✗      | 95.2%        |
      | Database    | UP       | Зеленый   | ✓      | 100.0%       |
      | Cache       | DEGRADED | Желтый    | ⚠     | 98.5%        |
    And DOWN мониторы отображаются первыми в списке
    And рассчитывается общий uptime процент

  @use_case=uc_03_01_06
  Scenario: Пагинация при большом количестве мониторов
    Given пользователь имеет "120" мониторов
    When пользователь просматривает дашборд
    Then отображается первая страница с "50" мониторами
    And доступна навигация пагинации:
      | опция         |
      | 50 per page   |
      | 100 per page  |
      | 200 per page  |
    When пользователь выбирает "100 per page"
    Then отображается "100" мониторов на странице
    And номер текущей страницы отображается в UI

  @use_case=uc_03_01_13a
  @boundary
  Scenario: Граница page_size пагинации (максимум)
    Given пользователь имеет "1000" мониторов
    When пользователь запрашивает страницу с size="500"
    Then возвращается ошибка с кодом "INVALID_PAGE_SIZE"
    And ответ содержит структуру:
      | error.code       | INVALID_PAGE_SIZE |
      | error.message    | Page size exceeds maximum |
      | error.details    | {"requested": 500, "max": 200} |
      | error.request_id | <uuid> |
    And автоматически применяется page_size="200"
    And действие логируется в audit log

  @use_case=uc_03_01_07
  Scenario: Фильтрация мониторов по статусу
    Given пользователь имеет мониторы с различными статусами:
      | name        | status   |
      | Service A   | UP       |
      | Service B   | DOWN     |
      | Service C   | DEGRADED |
      | Service D   | UP       |
      | Service E   | PAUSED   |
    When пользователь применяет фильтр "Только проблемы"
    Then отображаются только мониторы со статусом "DOWN" или "DEGRADED"
    And мониторы со статусом "PAUSED" исключаются из результатов
    And выбранные фильтры сохраняются на сервере для пользователя

  @use_case=uc_03_01_15a
  @validation
  Scenario: Неверный параметр фильтрации a
    Given пользователь авторизован в системе
    And пользователь имеет мониторы
    When пользователь применяет фильтр с неверным параметром "invalid_status"
    Then возвращается ошибка с кодом "INVALID_FILTER_PARAMETER"
    And ответ содержит структуру:
      | error.code       | INVALID_FILTER_PARAMETER |
      | error.message    | Invalid filter parameter |
      | error.details    | {"parameter": "status", "allowed_values": ["UP", "DOWN", "DEGRADED", "PAUSED"]} |
      | error.request_id | <uuid> |
    And действие логируется в audit log как failed

  @use_case=uc_03_01_15b
  @boundary
  Scenario: Максимальная длина фильтра
    Given пользователь авторизован в системе
    And пользователь имеет мониторы с различными тегами
    When пользователь применяет фильтр с длиной "1000" символов
    Then фильтр обрезается до "500" символов
    And возвращается предупреждение "FILTER_TOO_LONG"
    And ответ содержит структуру:
      | error.code       | FILTER_TOO_LONG |
      | error.message    | Filter exceeds maximum length |
      | error.details    | {"max_length": 500, "actual_length": 1000} |
      | error.request_id | <uuid> |
    And действие логируется в audit log как failed

  @use_case=uc_03_01_15c
  @boundary
  Scenario: Пустой результат при фильтрации
    Given пользователь имеет мониторы:
      | name        | status   | tags           |
      | API Prod    | DOWN     | production,api |
      | API Dev     | UP       | development,api|
      | Web Prod    | DEGRADED | production,web |
    When пользователь применяет фильтры:
      | фильтр  | значение    |
      | статус  | UP          |
      | теги    | production  |
    Then возвращается пустой список мониторов
    And возвращается предупреждение "NO_MONITORS_MATCHING_FILTERS"
    And ответ содержит структуру:
      | error.code       | NO_MONITORS_MATCHING_FILTERS |
      | error.message    | No monitors matching the specified filters |
      | error.request_id | <uuid> |
    And предлагается изменить критерии фильтрации
    And результат кэшируется

  @use_case=uc_03_01_08
  Scenario: Audit logging при изменении фильтров
    Given пользователь авторизован в системе
    When пользователь применяет фильтры:
      | фильтр  | значение    |
      | статус  | DOWN        |
      | теги    | production  |
    Then действие логируется в audit log:
      | action       | filters_changed |
      | user_id      | <user_id> |
      | filters      | {"status": ["DOWN"], "tags": ["production"]} |
      | success      | true |
    And log содержит предыдущие и новые значения фильтров

  @use_case=uc_03_01_09
  Scenario: Сортировка мониторов с приоритетом проблемных
    Given пользователь имеет мониторы с различными статусами:
      | name        | status   |
      | Service A   | UP       |
      | Service B   | DOWN     |
      | Service C   | DEGRADED |
      | Service D   | UP       |
    When пользователь просматривает дашборд
    Then мониторы упорядочены по умолчанию:
      | приоритет | статус   |
      | 1         | DOWN     |
      | 2         | DEGRADED |
      | 3         | UP       |
      | 4         | PAUSED   |
    And "Service B" (DOWN) отображается первым
    And "Service C" (DEGRADED) отображается вторым

  @use_case=uc_03_01_10
  Scenario: Отображение времени последней проверки
    Given пользователь имеет монитор "API Service"
    And последняя проверка была "5 minutes ago"
    When пользователь просматривает дашборд
    Then отображается время последней проверки для каждого монитора
    And если проверка была более "5 minutes" назад, показывается предупреждение
    And время хранится в UTC, отображается в timezone пользователя

  @use_case=uc_03_01_11
  Scenario: Комбинированная фильтрация
    Given пользователь имеет мониторы с различными статусами и тегами:
      | name        | status   | tags           |
      | API Prod    | DOWN     | production,api |
      | API Dev     | UP       | development,api|
      | Web Prod    | DEGRADED | production,web |
      | Web Dev     | UP       | development,web|
    When пользователь применяет фильтры:
      | фильтр  | значение    |
      | статус  | DOWN, DEGRADED |
      | теги    | production    |
    Then отображаются только мониторы с производственных окружений с проблемами
    And сохраняется комбинация примененных фильтров на сервере

  @use_case=uc_03_01_12
  Scenario: Поиск мониторов по названию
    Given пользователь имеет мониторы:
      | name              |
      | API Service       |
      | Web API           |
      | Database Service  |
    When пользователь вводит в поиск "API"
    Then отображаются только мониторы содержащие "API" в названии
    And результаты поиска обновляются в реальном времени

  @use_case=uc_03_01_16a
  @validation
  Scenario: Неверный поисковый запрос a
    Given пользователь авторизован в системе
    And пользователь имеет мониторы
    When пользователь вводит поисковый запрос с недопустимыми символами "@#$%"
    Then возвращается ошибка с кодом "INVALID_SEARCH_QUERY"
    And ответ содержит структуру:
      | error.code       | INVALID_SEARCH_QUERY |
      | error.message    | Search query contains invalid characters |
      | error.details    | {"query": "@#$%", "allowed_pattern": "^[a-zA-Z0-9\\s\\-._]+$"} |
      | error.request_id | <uuid> |
    And действие логируется в audit log как failed

  @use_case=uc_03_01_16b
  @boundary
  Scenario: Максимальная длина поискового запроса
    Given пользователь авторизован в системе
    And пользователь имеет мониторы
    When пользователь вводит поисковый запрос длиной "1000" символов
    Then запрос обрезается до "200" символов
    And возвращается предупреждение "SEARCH_QUERY_TOO_LONG"
    And ответ содержит структуру:
      | error.code       | SEARCH_QUERY_TOO_LONG |
      | error.message    | Search query exceeds maximum length |
      | error.details    | {"max_length": 200, "actual_length": 1000} |
      | error.request_id | <uuid> |
    And действие логируется в audit log как failed

  @use_case=uc_03_01_13
  Scenario: Визуальное отображение статуса PAUSED
    Given пользователь имеет монитор "Maintenance Service" со статусом "PAUSED"
    When пользователь просматривает дашборд
    Then монитор отображается с особым визуальным стилем:
      | статус   | цвет      | иконка | progress_bar | описание                   |
      | PAUSED   | Серый     | ⏸     | 100%         | Приостановлен пользователем |
    And монитор учитывается как 100% uptime в статистике
    And монитор исключается из downtime расчетов

  @use_case=uc_03_01_17a
  @boundary
  Scenario: Ровно 100 мониторов на дашборде
    Given пользователь имеет подписку "Pro" с лимитом "100" мониторов
    And пользователь создал ровно "100" мониторов
    When пользователь открывает дашборд
    Then отображаются все "100" мониторов
    And пагинация отображается корректно
    And фильтрация работает без ошибок
    And сортировка применяется ко всем мониторам
    And метрики рассчитываются корректно

  @use_case=uc_03_01_17b
  @boundary
  Scenario: Ровно 1000 мониторов (performance boundary)
    Given пользователь имеет подписку "Enterprise" с безлимитными мониторами
    And пользователь создал "1000" мониторов
    When пользователь открывает дашборд
    Then первая страница загружается не более чем за "5" секунд
    And применяется пагинация с "50" мониторов на странице
    And доступна навигация по страницам
    And общее количество мониторов отображается корректно
    And фильтрация и сортировка работают с приемлемой производительностью

  @use_case=uc_03_01_17c
  @performance
  Scenario: Сложная комбинация фильтров с большим dataset
    Given пользователь имеет "1000" мониторов
    And мониторы имеют различные статусы и теги
    When пользователь применяет фильтры:
      | фильтр  | значение           |
      | статус  | DOWN, DEGRADED     |
      | теги    | production,api,critical |
      | поиск   | API                |
    Then результаты фильтрации загружаются не более чем за "10" секунд
    And пагинация применяется корректно
    And метрики рассчитываются только для отфильтрованных мониторов
    And фильтр сохраняется на сервере

  @use_case=uc_03_01_17d
  @performance
  Scenario: Экспорт дашборда с большим количеством мониторов
    Given пользователь имеет "1000" мониторов
    And пользователь имеет права на экспорт данных
    When пользователь запрашивает экспорт всех мониторов в формате "CSV"
    Then система генерирует файл асинхронно
    And пользователь получает уведомление когда файл готов
    And файл содержит все "1000" мониторов
    And размер файла не превышает "10" MB
    And ссылка на файл доступна "24" часа
    And экспорт логируется в audit log

  @use_case=uc_03_01_17e
  @integration
  Scenario: Конкурентная загрузка дашборда несколькими пользователями
    Given тенант имеет "10" пользователей
    And все пользователи одновременно открывают дашборд
    And каждый пользователь запрашивает список мониторов
    Then все запросы обрабатываются успешно
    And время ответа не превышает "5" секунд для каждого пользователя
    And данные изолированы для каждого пользователя
    And race conditions отсутствуют
    And система масштабируется горизонтально

  @use_case=uc_03_01_17f
  @integration
  Scenario: WebSocket сервер падает во время подключения
    Given пользователь авторизован в системе
    And пользователь открывает дашборд
    When WebSocket сервер падает во время установления соединения
    Then возвращается ошибка с кодом "WEBSOCKET_CONNECTION_FAILED"
    And ответ содержит структуру:
      | error.code       | WEBSOCKET_CONNECTION_FAILED |
      | error.message    | WebSocket server unavailable |
      | error.details    | {"reason": "server_crash", "retry_after": "30s"} |
      | error.request_id | <uuid> |
    And система автоматически пытается переподключиться через "30" секунд
    And ошибка логируется в audit log

  @use_case=uc_03_01_17g
  @integration
  Scenario: Таймаут запроса данных дашборда a
    Given пользователь авторизован в системе
    And пользователь имеет "1000" мониторов
    When пользователь открывает дашборд
    And запрос к базе данных превышает "30" секунд
    Then возвращается ошибка с кодом "DASHBOARD_QUERY_TIMEOUT"
    And ответ содержит структуру:
      | error.code       | DASHBOARD_QUERY_TIMEOUT |
      | error.message    | Dashboard data query timeout |
      | error.details    | {"timeout": "30s", "query_duration": "<actual>"} |
      | error.request_id | <uuid> |
    And предлагается уменьшить количество мониторов или применить фильтры
    And timeout логируется в audit log

  @use_case=uc_03_01_17h
  @integration
  Scenario: Недоступность cache сервиса для дашборда
    Given пользователь авторизован в системе
    And пользователь открывает дашборд
    When cache сервис недоступен
    Then данные загружаются напрямую из базы данных
    And время ответа увеличивается
    And возвращается предупреждение "CACHE_UNAVAILABLE"
    And ответ содержит структуру:
      | error.code       | CACHE_UNAVAILABLE |
      | error.message    | Cache service unavailable, using direct database access |
      | error.details    | {"mode": "degraded_performance"} |
      | error.request_id | <uuid> |
    And ошибка логируется в audit log как cache_unavailable
    And система продолжает работать с degraded performance

  @use_case=uc_03_01_17i
  @state_transition
  Scenario: Изменение статуса монитора во время просмотра дашборда
    Given пользователь просматривает дашборд с монитором "API Service" со статусом "UP"
    And WebSocket соединение активно
    When статус монитора меняется на "DOWN"
    Then дашборд автоматически обновляется через WebSocket
    And монитор "API Service" переупорядочивается в начало списка
    And визуальные индикаторы обновляются (красный цвет)
    And отображается уведомление об изменении статуса
    And timestamp последней проверки обновляется

  @use_case=uc_03_01_17j
  @state_transition
  Scenario: Real-time обновление при добавлении нового монитора
    Given пользователь просматривает дашборд с "50" мониторами
    And WebSocket соединение активно
    When пользователь создает новый монитор "New Service"
    Then дашборд автоматически обновляется
    And новый монитор появляется в списке
    And счетчик мониторов обновляется
    And пользователь получает уведомление о создании монитора

  @use_case=uc_03_01_17k
  @state_transition
  Scenario: Real-time обновление при удалении монитора
    Given пользователь просматривает дашборд
    And на дашборде отображается монитор "Old Service"
    And WebSocket соединение активно
    When монитор "Old Service" удаляется
    Then дашборд автоматически обновляется
    And монитор удаляется из списка
    And счетчик мониторов обновляется
    And пользователь получает уведомление об удалении

  @use_case=uc_03_01_17l
  @performance
  Scenario: Обработка большого количества real-time обновлений
    Given пользователь имеет "100" мониторов
    And все мониторы обновляются каждую "1 second"
    And WebSocket соединение активно
    When поступает более "50" обновлений в секунду
    Then обновления группируются в пакеты
    And UI обновляется не чаще чем "20 times per second"
    And никакие обновления не теряются
    And визуальные изменения отображаются плавно
    And производительность UI остается приемлемой

  @use_case=uc_03_01_17m
  @accessibility
  Scenario: Навигация по дашборду с клавиатуры
    Given пользователь авторизован в системе
    And пользователь использует только клавиатуру для навигации
    When пользователь открывает дашборд
    Then все интерактивные элементы доступны через Tab
    And порядок Tab следует логической структуре страницы
    And фокус виден на всех элементах
    And Enter/Space активируют фокусированные элементы
    And shortcut keys доступны для основных действий
    And фокус не попадает в trap

  @use_case=uc_03_01_17n
  @accessibility
  Scenario: Поддержка screen reader на дашборде
    Given пользователь с нарушениями зрения использует screen reader
    And пользователь авторизован в системе
    When пользователь открывает дашборд
    Then всем элементам назначены ARIA labels
    And статусы мониторов озвучиваются текстом ("UP", "DOWN", "DEGRADED")
    And цвета дополняются текстовыми метками
    And иконки имеют текстовые альтернативы
    And таблицы имеют корректные заголовки
    And динамические обновления анонсируются через ARIA live regions

  @use_case=uc_03_01_17o
  @accessibility
  Scenario: Цветовой контраст и визуальная доступность
    Given пользователь с нарушениями зрения авторизован в системе
    When пользователь открывает дашборд
    Then текстовые элементы имеют контрастность минимум 4.5:1
    And крупные тексты имеют контрастность минимум 3:1
    And статусные индикаторы используют цвета + символы
    And не используется цвет как единственный способ передачи информации
    And пользователю доступна high contrast режим
    And размер шрифта можно увеличить до 200% без потери функциональности

  @use_case=uc_03_01_17p
  @cross_device
  Scenario: Адаптивный дизайн для мобильных устройств
    Given пользователь использует мобильное устройство с экраном "375px"
    And пользователь авторизован в системе
    When пользователь открывает дашборд
    Then интерфейс адаптируется под малый экран
    And sidebar сворачивается в hamburger menu
    And таблица мониторов переключается в card view
    And критическая информация видна без горизонтального скролла
    And touch targets имеют минимум 44x44 пикселей
    And текст остается читаемым без zoom

  @use_case=uc_03_01_17q
  @cross_device
  Scenario: Адаптивный дизайн для планшетов
    Given пользователь использует планшет с экраном "768px"
    And пользователь авторизован в системе
    When пользователь открывает дашборд
    Then интерфейс оптимизирован для среднего экрана
    And sidebar может сворачиваться/разворачиваться
    And таблица мониторов адаптирует количество колонок
    And пагинация остается доступной
    And фильтры доступны в выпадающем меню
    And touch-жесты поддерживаются для навигации

  @use_case=uc_03_01_17r
  @cross_browser
  Scenario: Кросс-браузерная совместимость (Chrome)
    Given пользователь использует браузер "Chrome" последней версии
    And пользователь авторизован в системе
    When пользователь открывает дашборд
    Then все функции работают корректно
    And WebSocket соединение устанавливается
    And real-time обновления отображаются
    And фильтры и сортировка работают
    And графики отрисовываются корректно
    And ошибки не появляются в консоли браузера

  @use_case=uc_03_01_17s
  @cross_browser
  Scenario: Кросс-браузерная совместимость (Firefox)
    Given пользователь использует браузер "Firefox" последней версии
    And пользователь авторизован в системе
    When пользователь открывает дашборд
    Then все функции работают корректно
    And WebSocket соединение устанавливается
    And real-time обновления отображаются
    And фильтры и сортировка работают
    And графики отрисовываются корректно
    And стили не "плывут"

  @use_case=uc_03_01_17t
  @cross_browser
  Scenario: Кросс-браузерная совместимость (Safari)
    Given пользователь использует браузер "Safari" на macOS
    And пользователь авторизован в системе
    When пользователь открывает дашборд
    Then все функции работают корректно
    And WebSocket соединение устанавливается
    And real-time обновления отображаются
    And фильтры и сортировка работают
    And шрифты отображаются корректно
    And Safari-specific особенности учитываются

  @implemented
  @use_case=uc_03_01_18a
  Scenario: Событие monitor.paused обновляет статус монитора в dashboard
    Given в системе создан монитор через событие "monitor.created"
    When публикуется событие "monitor.paused" со статусом "PAUSED"
    Then в dashboard статус монитора становится "PAUSED"

  @implemented
  @use_case=uc_03_01_18b
  Scenario: Событие monitor.created добавляет монитор в dashboard
    When публикуется событие "monitor.created" с именем "Dashboard Sync Monitor"
    Then в dashboard появляется запись монитора с именем "Dashboard Sync Monitor"

  @implemented
  @use_case=uc_03_01_18c
  Scenario: Невалидное сообщение не ломает dashboard consumer
    Given в системе создан монитор через событие "monitor.created"
    When публикуется невалидное JSON сообщение с routing key "monitor.paused"
    And публикуется событие "monitor.resumed" со статусом "UP"
    Then в dashboard статус монитора становится "UP"

  @implemented
  @use_case=uc_03_01_18d
  Scenario Outline: Все статусы монитора отражаются в dashboard
    Given в системе создан монитор через событие "monitor.created"
    When публикуется событие "<event>" со статусом "<status>"
    Then в dashboard статус монитора становится "<status>"

    Examples:
      | event            | status   |
      | monitor.paused   | PAUSED   |
      | monitor.resumed  | UP       |
      | monitor.updated  | DOWN     |
      | monitor.updated  | DEGRADED |

  @use_case=uc_03_01_17u
  @performance
  Scenario: Throttling high-frequency обновлений
    Given пользователь имеет "100" мониторов
    And все мониторы обновляются каждую "1 second"
    And WebSocket соединение активно
    When поступает более "100" обновлений в секунду
    Then обновления группируются в пакеты
    And UI обновляется не чаще чем "30 times per second"
    And визуальные изменения применяются пакетно
    And нагрузка на CPU браузера ограничена
    And никакие данные не теряются
    And пользователь получает уведомление о высокой частоте обновлений
