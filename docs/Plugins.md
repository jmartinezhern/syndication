# Plugins

## Types

### API

### Content

## Initialization

```go
func Init(c Context) {
  c.onNewEntries().do(func(c Context) {
    c.entries = entries
  })
}

func Exit() {

}
```
