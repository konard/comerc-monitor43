---
name: "x-storage-s3"
description: "Применяй когда работаешь с S3-совместимым хранилищем через `../adapters/storage/minio`: выбор между `Put/Get/Delete`, `GetFileHeader`, presigned URL, multipart upload и `storage` error helpers"
compatibility: ../adapters
---

# S3-совместимое хранилище

## Когда применять

Используй этот скилл, когда работаешь с MinIO, Yandex Cloud Storage, AWS S3 или другим S3-compatible storage через `storage/minio`.

Не применяй для:
- локальной файловой системы
- tiny utility files, которые не требуют отдельного storage adapter

## Workflow

### 1. Выбери конструктор

Простой путь:

```go
storage, err := minio.NewDefault(cfg)
if err != nil {
    return err
}
defer func() {
    if err := storage.Close(); err != nil {
        // обработка
    }
}()
```

Стартовый конфиг:

```go
cfg := minio.Config{
    Endpoint:  "localhost:9000",
    AccessKey: "minioadmin",
    SecretKey: "minioadmin",
    Secure:    false,
}
```

### 2. Выбери операцию по задаче

| Задача | Операция |
|---|---|
| сохранить объект | `Put` |
| прочитать объект | `Get` |
| проверить наличие | `Exists` |
| удалить | `Delete` |
| определить тип файла без полного скачивания | `GetFileHeader` |
| дать клиенту временный прямой доступ | `GetPresignedURL` |
| загрузить большой файл частями | multipart API |
| перечислить объекты | `List` |

#### Когда выбирать Presigned URL

- backend не должен проксировать трафик
- клиент может ходить в storage напрямую
- доступ нужно ограничить по времени

#### Когда выбирать Multipart

- файл большой
- загрузка идёт кусками
- нужен контролируемый abort/retry

Для файлов заметно больше `5MB` multipart обычно безопаснее, чем один большой `Put`.

Всегда проектируй error path с `AbortMultipartUpload`.

### 3. Обрабатывай `Get` и multipart аккуратно

Для `Get`:
- всегда закрывай `io.ReadCloser`
- ошибку `Close()` не игнорируй

Для multipart:
- при падении части вызывай `AbortMultipartUpload`
- собирай `UploadedPart` и только потом делай `CompleteMultipartUpload`

## Типовые паттерны

### `Put`

```go
err := storage.Put(ctx, bucket, key, bytes.NewReader(data), &storage.PutOptions{
    ContentType: "application/json",
    Metadata: map[string]string{
        "source": "api",
    },
})
```

### `Get`

```go
reader, info, err := storage.Get(ctx, bucket, key)
if err != nil {
    return err
}

data, readErr := io.ReadAll(reader)
closeErr := reader.Close()
if readErr != nil {
    return readErr
}
if closeErr != nil {
    return fmt.Errorf("failed to close object reader: %w", closeErr)
}

_ = info
_ = data
```

### `Exists` и `Delete`

```go
exists, err := storage.Exists(ctx, bucket, key)
if err != nil {
    return err
}
if !exists {
    return nil
}

if err := storage.Delete(ctx, bucket, key); err != nil {
    return err
}
```

### `List`

```go
result, err := storage.List(ctx, bucket, &storage.ListOptions{
    Prefix:    "photos/2026/",
    Recursive: true,
    MaxKeys:   1000,
})
if err != nil {
    return err
}

for _, obj := range result.Objects {
    _ = obj.Key
}
```

### `GetFileHeader`

Используй для определения типа файла без скачивания всего объекта:

```go
header, err := storage.GetFileHeader(ctx, bucket, key)
fileType := http.DetectContentType(header)
```

### Presigned URL

Используй, когда клиент должен читать или писать объект напрямую, минуя ваш backend:

```go
url, err := storage.GetPresignedURL(ctx, bucket, key, &storage.PresignedURLOptions{
    Method: "GET",
    Expiry: 15 * time.Minute,
})
```

### Multipart

Используй для больших объектов и потоковой загрузки:

```go
upload, err := storage.CreateMultipartUpload(ctx, bucket, key, nil)
if err != nil {
    return err
}

parts := make([]storage.UploadedPart, 0, len(chunks))
for i, chunk := range chunks {
    part, err := storage.UploadPart(ctx, bucket, key, upload.UploadID, int32(i+1), bytes.NewReader(chunk))
    if err != nil {
        if abortErr := storage.AbortMultipartUpload(ctx, bucket, key, upload.UploadID); abortErr != nil {
            return errors.Join(err, abortErr)
        }
        return err
    }
    parts = append(parts, *part)
}

_, err = storage.CompleteMultipartUpload(ctx, bucket, key, upload.UploadID, &storage.CompleteMultipartUploadOptions{
    Parts: parts,
})
if err != nil {
    if abortErr := storage.AbortMultipartUpload(ctx, bucket, key, upload.UploadID); abortErr != nil {
        return errors.Join(err, abortErr)
    }
    return err
}
```

## Ошибки

Если проверяешь семантику ошибки, используй хелперы из `storage`:

```go
if storage.IsNotFound(err) {
    // объект не найден
}
```

## Практические правила

- если bucket пустой, адаптер может использовать `DefaultBucket`
- все операции поддерживают `context.Context`
- tracing и логирование уже встроены в адаптер

## Не делай

- не проксируй через backend большие клиентские скачивания, если достаточно presigned URL
- не забывай `AbortMultipartUpload` на error path
- не проверяй not-found через текст ошибки, если доступен helper
