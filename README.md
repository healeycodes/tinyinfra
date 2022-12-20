[![Go](https://github.com/healeycodes/tinyinfra/actions/workflows/go.yml/badge.svg)](https://github.com/healeycodes/tinyinfra/actions/workflows/go.yml)

# 🏗 tinyinfra

A little learning project. A key/value store (get/set) and queue (send/receive/delete).

Built with Go's standard library and [GORM](https://gorm.io/).

- GET **/user/new**
  - (returns `token` to be used via Bearer authentication for all other endpoints)
  
- POST **/kv/set** `{"key": "some_key", "value": "some_value", "ttl": 1671543399714}`
  - (`ttl` is optional)
- GET **/kv/get** `{"key": "some_key"}`
  - (returns `key`, `value`, `ttl`)
  
- POST **/queue/send** `{"namespace": "some_namespace", "message": "some_message"}`
- GET **/queue/receive** `{"namespace": "some_namespace", "visibilityTimeout": 20000}`
  - (returns `namespace`, `message`, `id`)
- POST **/queue/delete** `{"namespace": "some_namespace", "id": 1}`

## Tests

Integration tests (a few tests per endpoints) run with `go test ./...`

End-to-end tests (fairly simple) `python e2e.py`

## Dev

`go run .`
