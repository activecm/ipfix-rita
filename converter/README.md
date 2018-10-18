# IPFIX-RITA Converter

#### An IPFIX/ Netflow v9 Flow Sticher and Converter for RITA

Netflow v9 and IPFIX records generally come in as 1-way network flows. Each
record describes a connection from one computer to another, but not the other
way back. This is quite different than the connection records provided by
Bro IDS as Bro IDS reports metrics on both sides of any given connection.

In order to use network flow data with RITA, we must first stitch the data
together. This is done by matching flows against each other by inspecting
the source and destination IP addresses and ports, the transport protocol,
and the exporter which recorded the flow.

#### Program Structure

Main file: `boot.go`

Main command: `commands/convert.go`

Main components:
- An interface for consuming data: `input/reader.go`
    - Implementation: `input/mgologstash/reader.go`
        - Requires a buffer class conforming to `input/mgologstash/buffer.go`
            - Implementation: `input/mgologstash/id_bulk_buffer.go`
- An interface for holding network flow data: `input/flow.go`
    - Implementation: `input/mgologstash/flow.go`
        - This is where data is being sanitized on input
    - Mock: `input/flow_mock.go`
- The stitching manager: `stitching/manager.go`
    - Should have the flow matcher DI'd into into it via constructor, but does not. Interface: `stitching/matching/matcher.go`
        - Implementation: `stitching/matching/rammatch/ram.go`
    - Partitions input data stream to multiple stitchers: `stitching/sticher.go`
    - The stitching subsystem produces sessions from flows: `stitching/session/session.go`
- An interface writing out processed data: `output/writer.go`
    - Implementation: `output/rita/streaming/dates/rita_dates.go`

#### Testing the Converter

`go test` is used throughout the project. `make test` will perform the following commands:

```
dep ensure -v
# check if docker is installed and warn if it is not
go test -v -p=1 ./...
```

Note the `-p=1`, this ensures the packages are tested sequentially.

The tests in the Converter software make use of the DBTest package. The DBTest
package lets test writers spin up an instances of MongoDB via Docker and
connect to them for integration testing. If you would like to skip the integration
tests, use the `-short` flag with `go test`.
