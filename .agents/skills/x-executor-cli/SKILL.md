---
name: "x-executor-cli"
description: "Применяй когда нужно обернуть внешний CLI в адаптер или инфраструктурный компонент: `executor/cli.New`, `Start`, `Execute(ctx, ...)`, локальный запуск, SSH и проверка доступности команды"
compatibility: ../adapters
---

# CLI Executor

## Когда применять

Используй этот скилл, когда код должен запускать внешний исполняемый файл:
- `ffmpeg`, `ffprobe`, `convert`, `gsutil`, `aws`
- удалённую команду через SSH
- инфраструктурный adapter layer, а не бизнес-логику

Не применяй для:
- прямого `exec.Command` в service/domain коде
- shell pipelines в пользовательском коде без отдельного адаптера

## Workflow

### 1. Определи, это локальный CLI или SSH

Локальный вариант:

```go
cfg := cli.Config{Command: "ffmpeg"}
executor := cli.New(cfg, nil, nil)
```

SSH-вариант:

```go
cfg := cli.Config{
    Command: "docker",
    SSH: cli.SSHConfig{
        Host:    "remote.example.com",
        User:    "deploy",
        KeyPath: "/home/deploy/.ssh/id_rsa",
    },
}
executor := cli.New(cfg, nil, nil)
```

### 2. Сразу проверь доступность команды через `Start()`

```go
if err := executor.Start(); err != nil {
    return fmt.Errorf("failed to start ffmpeg executor: %w", err)
}
```

Что проверяет `Start()` в `../adapters/executor/cli`:
- наличие команды в `PATH`
- наличие `ssh`, если настроен SSH режим
- наличие `sshpass`, если используется SSH по паролю

### 3. Выполняй команду через `Execute(ctx, args...)`

```go
err := executor.Execute(ctx,
    "-i", "input.mp4",
    "-c:v", "libx264",
    "-y", "output.mp4",
)
if err != nil {
    return fmt.Errorf("ffmpeg not run: %w", err)
}
```

### 4. Закрывай executor и обрабатывай `Close()`

Если executor реализует `Close()`, регистрируй cleanup сразу после `New(...)`.

## Практические правила

- `stdout` и `stderr` передавай явно, если не подходит вывод в стандартные потоки процесса
- один executor обычно соответствует одной конкретной команде
- `context.Context` обязателен для таймаута и отмены
- если компонент использует несколько CLI, оформляй их как отдельные зависимости, а не как один “универсальный” executor

## Тестирование

Зависимость на executor подменяй интерфейсом потребителя:

```go
type cliExecutor interface {
    Start() error
    Execute(ctx context.Context, args ...string) error
    Close() error
}
```

## Не делай

- не вызывай `Start()` лениво в середине бизнес-операции, если можно провалиться раньше
- не игнорируй `Close()` и ошибки закрытия
- не тащи `exec.Command` напрямую в service-слой
