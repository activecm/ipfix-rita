# IPFIX-RITA

IPFIX-RITA is a system for processing IPFIX/ Netflow v9 records for use with
[RITA](https://github.com/activecm/rita).

# Structure

IPFIX-RITA is made up of four components. These are the

- Collector (Logstash)
- Buffer (MongoDB)
- Converter (Written with Go)
- RITA database (MongoDB)

# Running IPFIX-RITA

The Collector, Buffer, and Converter are managed by Docker.
In order to start the system, run `docker-compose up -d` inside the `docker/`
folder. This will start Logstash, MongoDB, and the IPFIX-RITA Converter.
To stop the system, run `docker-compose stop`. To view the IPFIX-RITA logs,
run `docker-compose logs`.

The UDP port 2055 on the host machine will be mapped to the Collector (Logstash).
You may send your IPFIX/ Netflow v9 traffic here.

By default, the system uses the same MongoDB instance to buffer the incoming
data and store the resulting RITA datasets. This database is local to the
machine running IPFIX-RITA and is accessible on the host machine on TCP port `27017`.
This means RITA will run out of the box with a machine running IPFIX-RITA.

You may send the resulting RITA datasets to a different database by editing
the file at `docker/etc/converter/converter.yaml`. Edit the database
settings under `Output`. By default the resulting RITA datasets will be named
`IPFIX-{Date}` where `{Date}` is the date corresponding to all of the records
held within the dataset. You may change the prefix from `IPFIX` to something
of your choosing by editing the `DBRoot` field.

## Collector
The collector receives IPFIX and Netflow v9 data, decodes it and places it
in the buffer. Currently, Logstash (loaded with the Netflow and MongoDB plugins)
is used as the collector. The Logstash collector requires a large UDP buffer
in order to prevent the dropping of packets. Set your UDP receive buffer with
`sudo sysctl -w net.core.rmem_max=$((1024*1024*64))` before running IPFIX-RITA.

The collector is multithreaded. In order to make use of multiple worker threads,
edit the line `- INPUT_WORKERS=4` under `logstash` in `docker/docker-compose.yaml`.
This number should generally be set to the number of CPUs available on the system
minus one.

## The Buffer
In order to prevent the loss of packets, data is buffered into MongoDB between
the Collector and the Converter. By default a local instance of MongoDB is used.
However, this can be changed by editing `docker/etc/collector/logstash/mongo.conf`
and `docker/etc/converter/converter.yaml`. The logstash configuration may be
changed with respect to the options set out [here](https://www.elastic.co/guide/en/logstash/current/plugins-outputs-mongodb.html).
Any changes to the `uri`, `collection`, or `database` fields must be
reflected in the `Input` settings in `converter.yaml`.

The local MongoDB instance is likely to work well enough for most installations.
However, when processing more than 15K flows/ second, you may want to use 
a dedicated MongoDB cluster. This may change in the future.

## Converter
The converter reads in network flow data, matches flows together, and outputs
records in RITA compatible datasets. As a part of this process, the converter
marks which hosts are part of the local network. By default the class A, B, and C
IPv4 networks are marked local. However, this can be changed by editing the
`LocalNetworks` list in `converter.yaml`. IPv6 subnets may
also be added to this list.

This repo holds tools related to converting and importing IPFIX (Netflow V10) and Netflow V9 records into a mongodb database in a format RITA can understand.
