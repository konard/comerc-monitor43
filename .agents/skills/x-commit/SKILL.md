---
name: "x-commit"
description: "Применяй только по явной команде пользователя вроде «закоммить», «сделай коммит», «/commit»: проанализируй diff, составь commit message `type: subject` + body и выполни локальный git commit без push"
---

# Локальный коммит

> **Этот скилл запускается только по явной команде пользователя.**
> Никогда не вызывай его самостоятельно.
>
> Команда может быть разовой («закоммить») или blanket («коммить после каждой подфазы»).
> Blanket-команда действует до конца сессии или до явной отмены.
>
> **Формат строго:** `type: subject`.
> Scope в скобках не использовать, даже если он есть в истории проекта.
>
> **Делегируй выполнение этого скилла субагенту с самой младшей (дешёвой) доступной моделью.**

## Когда применять

Используй этот скилл, когда пользователь прямо просит:
- закоммитить текущие изменения
- составить commit message и сделать локальный commit
- выполнить `/commit`

Не применяй для:
- обычного описания diff без коммита
- push, rebase, amend, squash и других git-операций

## Workflow

### 1. Собери состояние рабочего дерева

```bash
git status
git diff --cached --stat
```

Режимы:
- если есть staged-изменения, коммить только их и не подтягивай unstaged "заодно"
- если staged пуст, коммить unstaged-изменения, но добавлять только конкретные файлы этого коммита
- если рабочее дерево грязное, сначала отдели релевантные файлы от чужих или побочных изменений

Для анализа diff:

```bash
git diff --cached   # если staged есть
git diff            # если staged пуст
```

### 2. Сформируй commit message

Формат:

```text
<type>: <subject>

<body>
```

Типы:
- `feat` — новая функциональность
- `fix` — исправление ошибки
- `hotfix` — срочное исправление production/staging инцидента
- `docs` — документация
- `refactor` — переписывание без изменения поведения
- `test` — тесты
- `style` — форматирование, lint
- `perf` — оптимизация
- `chore` — инфраструктурные и служебные изменения

Правила:
- `subject` на английском, в нижнем регистре, без точки
- `subject` короткий, обычно 3-5 слов
- `body` обязателен
- `body` тоже на английском
- `body` объясняет что и зачем изменено, а не пересказывает diff построчно
- обычное исправление — это `fix`; `hotfix` не используй без реального incident context

Примеры:

```text
feat: add rabbitmq retry config

Add explicit retry queue settings for subscriber startup.
Keep the topology contract visible in configuration.
```

```text
fix: handle empty room id

Return a validation error before calling the repository.
Prevent nil dereference in the service flow.
```

```text
hotfix: restore s3 endpoint config

Restore the production endpoint parsing after the rollout regression.
Prevent startup failure in the file upload path.
```

### 3. Создай коммит

```bash
git add <конкретные файлы>   # только если staged пуст
git commit -m "<type>: <subject>" -m "<body>"
```

### 4. Проверь результат

```bash
git log --oneline -3
git status
```

В ответе пользователю покажи hash коммита и итоговое сообщение.

## Ограничения

- не пушить в `origin`
- не использовать `git add -A` и `git add .`
- не использовать `--no-verify`
- если pre-commit хук упал, сначала исправь проблему, потом создай новый commit
- не коммитить секреты и локальные `.env`
