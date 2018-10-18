# IPFIX-RITA Collector

#### A Logstash Based IPFIX/ Netflow v9 Collector for RITA-IPFIX

We make use of the open source netflow/ ipfix plugin for Logstash in order
to collect network data.

In the current set up, Logstash collects IPFIX/ Netflow v9 records, parses them,
and stores them in MongoDB for processing by the Converter.

### Helpful links

Logstash Netflow Codec: https://www.elastic.co/guide/en/logstash/current/plugins-codecs-netflow.html

Logstash Netflow Codec Github Page: https://github.com/logstash-plugins/logstash-codec-netflow

Logstash MongoDB Output Plugin: https://www.elastic.co/guide/en/logstash/current/plugins-outputs-mongodb.html

Logstash MongoDB Output Plugin Github Page: https://github.com/logstash-plugins/logstash-output-mongodb

### Benchmarking Scripts

Several python scripts have been written to study the performance of the collector.
These scripts send netflow v9 and IPFIX records as produced by various devices and programs.
They first send the template needed to decode the data, then they send the same set of flows
to the collector repeatedly. These scripts are based off of the benchmarking scripts in the
Logstash Netflow Codec Github Repository.

#### Quick Notes

The collector expects an environment variable `INPUT_WORKERS` to be defined as
it controls how many threads are used to read from the input queue. By default,
this value is set to 4.
