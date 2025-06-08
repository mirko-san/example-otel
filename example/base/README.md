# base

Example without OTel

## usage

### sever

```
# default port :3030
$ go run server/main.go
```

To override the port (e.g. `:3031`):

```
$ EXAMPLE_SERVER_PORT=3031
$ go run server/main.go
```

### client

```
# Send a request to http://localhost:3030/hello
$ go run client/main.go
```

```
# Customize the request URL
$ go run client/main.go --server http://localhost:3030/httpbin/status/404
```
