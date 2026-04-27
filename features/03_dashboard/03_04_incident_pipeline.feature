@epic=03_dashboard
@user_story=03_04_incident_pipeline
# Description: Event-driven обработка incident.detected/incident.resolved и устойчивость consumer

Feature: Incident pipeline в dashboard
  Как система мониторинга
  Я хочу детерминированно отражать события инцидентов в dashboard
  Чтобы пользователь видел актуальный статус и длительность инцидентов

  @implemented
  @integration
  @use_case=uc_03_04_01
  Scenario: Событие incident.detected создаёт активную запись инцидента
    Given в системе есть монитор
    When публикуется событие incident.detected
    Then в incidents появляется активный инцидент
    And поле started_at инцидента заполнено
    And поле ended_at инцидента пустое

  @implemented
  @integration
  @use_case=uc_03_04_02
  Scenario: Событие incident.resolved закрывает инцидент с длительностью
    Given в системе есть монитор
    And опубликовано событие incident.detected
    And в incidents появляется активный инцидент
    When публикуется событие incident.resolved через "1" секунду
    Then инцидент в incidents отмечен как разрешённый
    And поле ended_at инцидента заполнено
    And поле duration_seconds инцидента не отрицательное

  @implemented
  @integration
  @use_case=uc_03_04_03
  Scenario: Инциденты двух мониторов трекаются независимо
    Given в системе есть инцидент "A"
    And в системе есть инцидент "B"
    When публикуется событие incident.detected для инцидента "A"
    And публикуется событие incident.detected для инцидента "B"
    And в incidents оба инцидента "A" и "B" имеют статус "ACTIVE"
    And публикуется событие incident.resolved для инцидента "A"
    Then инцидент "A" имеет статус "RESOLVED"
    And инцидент "B" остаётся в статусе "ACTIVE"

  @implemented
  @integration
  @use_case=uc_03_04_04
  Scenario: incident.resolved для несуществующего инцидента не останавливает consumer
    Given в системе есть инцидент "unknown"
    And в системе есть инцидент "valid"
    When публикуется событие incident.resolved для инцидента "unknown"
    And публикуется событие incident.detected для инцидента "valid"
    Then инцидент "valid" имеет статус "ACTIVE"
    And фантомная запись для инцидента "unknown" отсутствует
