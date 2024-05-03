# ingrid osrm demo

run with `go run`:

```
go run github.com/sirdivealot/ingrid@latest
```

otherwise `git clone` and `go build .` or `go run .`

- env `HOST` controls host binding, e.g. `HOST=0.0.0.0`
- env `PORT` specifies port binding, e.g. `PORT=8080`

## structure

```
├── api         # generated from spec.yaml
├── ext
│   └── osrm    # osrm client
├── handlers.go # api handler implementation
├── main.go     # server entrypoint
├── spec.yaml   # OpenAPI 3 spec
└── tools.go    # go generate
```

## implementation notes

- rate limited to 1 req/sec
- CORS allowed for all origins with max-age
- `go generate` for codegen
