# example-otel

## requirements

- go 1.22.5

## usage

```
$ cp .envrc.sample .envrc
$ source .envrc
```

### sever

```
# default port :3030
$ go run cmd/server/main.go
```

### client

```
# request to http://localhost:3030/hello
$ go run cmd/client/main.go
```

```
# customise URL
$ go run cmd/client/main.go --server http://localhost:3030/httpbin/status/404
```
