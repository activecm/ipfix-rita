# The Stitching Package

The stitching package implements the core of IPFIX-RITA. It handles
piecing together flows into overall sessions.

Flows contain connection information in the unidirectional form of "host A talked to host B".
Each flow records how many bytes and packets were sent from host A to host B and how long the connection lasted.

However, sessions, as consumed by RITA, contain connection information in the bidirectional
form of "host A talked to host B and host B talked to host A". Each session
records how may bytes and packets were sent from each host and how long the overall connection lasted.

The stitching package matches flows together to create sessions.

Flows are generally matched using the 6 tuple of:

    - IP address A
    - IP address B
    - Port A
    - Port B
    - Transport Protocol
    - Netflow/ IPFIX exporting device (IP address)

This 6 tuple is contained within `session.AggregateQuery` objects.

## The Stitching Manager

The stitching manager is responsible for converting a
stream/ array of flows into a stream/ array of sessions.  

The stitching manager handles coordinating the work for a set of parallel stitching workers.

The main stitching algorithm is explained in the following flowchart. You will likely need to zoom in on the SVG file.

![Stitching Manager Algorithm](Stitching%20Manager%20Algorithm.svg)

The Stitching Manager contains the main processing loop. On each iteration, the manager consumes a new `input.Flow` object, distributes it to an appropriate stitcher, and checks if the stitching matcher needs flushed. If the stitching matcher needs flushed, the stitchers are flushed and paused, and the matcher is flushed. Then the loop begins again.

Note that the above flow chart represents the situation when `numStitchers` is set to 2.

The selectSticher function must assign flows with the same 6-tuple to the same stitcher. Additionally, if a flow comes in with the flipped version of the same 6-tuple, it must be assigned to the same stitcher. This is needed to prevent the parallel stitchers from squashing each other's work. This is carried out with a technique known as "Hash Partitioning". (See this [Medium post](https://medium.com/@Pranaykc/understanding-partitioning-in-distributed-systems-4ac3c8010fae) for a discussion of the technique in the context of distributed systems.)

## The Stitcher

Each stitcher works in tandem with the matcher to find appropriate matches for flows and transform them into session aggregates.

The stitcher algorithm is explained by the following flowchart. You will likely need to zoom in on the SVG file.

![Stitcher Algorithm](Stitcher%20Algorithm.svg)

Note that `session.Aggregate` objects may be half-filled, representing an individual flow, or they may be merged to represent multiple flows.

## The Matcher

The matcher is responsible for maintaining an index on the
session aggregates the program has seen so far. Each
stitcher queries the Matcher for matching session aggregates. The matches are filtered and the best match is found. If no match is found, the stitcher inserts the unmatched aggregate into the Matcher. Otherwise, the stitcher merges the matched aggregate into the newly arrived session aggregate. If this completes the session, the stitcher removes the matched aggregate from the Matcher. If the merge did not complete the session (two flows from host A to host B were merged), the stitcher updates the matched entry in the Matcher with the newly merged aggregate.

The Matcher must assign into each `session.AggregateQuery`'s `MatcherID` field in the
`Insert()` method in order to support duplicate `session.AggregateQuery` objects. Otherwise, the `Remove` and `Update` methods will not be able to disambiguate between duplicate records.

The Matcher must be thread safe. Ideally, the Matcher will not block for each operation to finish. If the Matcher blocks for each operation, the parallel speed up from running multiple stitchers will be negated.

The Matcher can be parallelized as each stitcher works on its own set of keys (`session.AggregateQuery`'s). This guarantee is made by the partition method described above. Go's sync.Map fits this use case well and is used in the `matching.rammatch` package.

An alternative implementation had been written using MongoDB for matching, but it was removed as the syscall's needed to communicate with MongoDB took too long.

The Matcher may "gunk up" with unmatched `session.AggregateQuery`'s as time goes on. For this reason, the Matcher must support a `ShouldFlush()` and `Flush()` method. These methods purge the Matcher of these unmatched entries. Heuristics may be used to determine which entries to flush.
