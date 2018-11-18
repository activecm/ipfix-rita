# Developer Notes

### Structure

IPFIX-RITA is made up of four components. These are the

- Collector (Logstash)
  - Transforms IPFIX/Netflow v9/Netflow v5 records into records compatible with the Converter
- Buffer (MongoDB)
  - Used to buffer records created by the collector until they are read by the Converter
- Converter (Written with Go)
  - Converts unidirectional flow data into bidirectional connection records for use with RITA
- RITA database (MongoDB)
  - Holds data processed by the Converter

### Building the Converter

Once the configuration file has been installed, the converter executable will be able to run
on its own. 

The converter may be built outside of Docker using the `Makefile` in the
`converter/` directory. Before running the converter ensure you have a config
file at `/etc/ipfix-rita/converter/converter.yaml`. This may be done one of three ways.

1. manually copy `runtime/etc/converter/converter.yaml` to `/etc/ipfix-rita/converter/converter.yaml`.
2. run `make install` to install the converter software natively
3. run the release installer.

### Additional Notes
To control the dockerized syster as a whole use `runtime/bin/ipfix-rita`.

If you'd like to make a development build of the dockerized system run
`runtime/bin/ipfix-rita build`.

The `dev-scripts/make-release` script is used to produce a release tarball.
