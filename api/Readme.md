# Protos

to generate use a protoc version that is atleast >= 3.12.3 and make sure you have `protoc-gen-go` plugin installed at `${GOPATH}/bin/`

```bash
protoc -I. -I /usr/local/include/google/protobuf/ --go_out "." --go_opt paths=source_relative --plugin=protoc-gen-go="${GOPATH}/bin/protoc-gen-go" api/*.proto
```
