# The Blockchain Bar
A complete blockchain project written in Go

## Install
```
go mod vendor
go install ./cmd/...
```

## Usage
### List all possible commands, arguments and configurations
```
tbb help
```

### Run TBB blockchain connected to the test network
```
tbb run --datadir=~/.tbb
```

### Run TBB blockchain in isolation on localhost
```
tbb run --datadir=~/.tbb --bootstrap=""
```

### Create a new account
```
tbb wallet new-account --datadir=~/.tbb 
```

## HTTP Usage
### List all balances
```
curl -X GET http://localhost:8080/balances/list -H 'Content-Type: application/json'
```

### Send and sign a new TX
```
curl --location --request POST 'http://localhost:8080/tx/add' \
--header 'Content-Type: application/json' \
--data-raw '{
	"from": "0x22ba1f80452e6220c7cc6ea2d1e3eeddac5f694a",
	"to": "0x6fdc0d8d15ae6b4ebf45c52fd2aafbcbb19a65c8",
	"value": 100,
	"pwd": "indeed-worried-action-wear"
}'
```

## Compile
To local OS:
```
go install ./cmd/...
```

To cross-compile:
```
xgo --targets=linux/amd64 ./cmd/tbb
```

## Tests
Run all tests with verbosity but one at a time, without timeout, to avoid ports collisions:
```
go test -v -p=1 -timeout=0 ./...
```

**Note:** Majority are integration tests and take time. Expect the test suite to finish in ~30 mins. 
