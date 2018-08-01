# RITADiff

RITADiff is a fork of [MongoDiff](https://github.com/activecm/mongo-diff).
MongoDiff checks whether two MongoDB databases are the same (including indexes).

RITADiff is used to check whether the results of IPFIX-RITA match those
produced by Bro IDS and `rita import`.

RITADiff only checks the `conn` collection in each database and
only checks the fields that are absolutely required by RITA. Currently,
these fields are:

- ts
- id_orig_h
- id_orig_p
- id_resp_h
- id_resp_p
- proto
- duration
- local_orig
- local_resp
- orig_pkts
- orig_ip_bytes
- resp_pkts
- resp_ip_bytes

## Usage
`rita-diff.py <uri> <database 1> <database 2> 2>unmatched-records`

RITADiff exits with status 0 if the databases are the same and 1 if they are not.

Stdout displays a human readable status information and a report on
how many records matched in each collection.

Stderr displays a line separated list of records in `<database 1>`
which were not found in `<database 2>`

## Dependencies

- Python3
- PyMongo

## Notes

RITADiff will usually use the `ts` index when searching the `conn` collection.

IPFIX-RITA ouputs records into different date separated databases based
on the *CLOSE* timestamp for each connection. Bro IDS outputs records into
different date separated folders based on the *OPEN* timestamp for each collection.
In order to avoid complications from this discrepancy, test with datasets that
span a single day. `editcap` can be used to pare down a pcap file to a single day.
However, remember to specify `TZ=UTC editcap ...` to specify the timezone as UTC
for editcap since Bro IDS and IPFIX use unix timestamps in UTC. 
