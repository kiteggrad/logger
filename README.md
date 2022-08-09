# logger

Обёртка над https://github.com/uber-go/zap для упрощения инициализации и использования zap логгера

## Quick Start

```go
log, err := logger.New(logger.Config{})
if err != nil {
    panic(err)
}
log.SetLevel("info") // по умолчанию "debug"
logger.SetGlobal(log)
```