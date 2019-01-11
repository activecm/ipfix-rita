# Developer Notes

### Structure

IPFIX-RITA is made up of four components. These are the

- Collector (Logstash)
  - Transforms IPFix/Netflow v9/Netflow v5 records into records compatible with the Converter
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

### Generating a Release

The `dev-scripts/make-release` script is used to produce a release tarball.

First ensure the code is ready for release by testing it with `runtime/bin/ipfix-rita`.

Next, run `dev-scripts/make-release`. This script will build each of the
necessary docker images and create an installer tarball.
The resulting tarball will contain the docker images, the files in the
runtime directory, the documentation files, and an installer script.
Note that this will tag the resulting docker images with current version of the
software. We may want to add a `--test` flag to `make-release`
to avoid this in the future.

You will be asked if you would like to publish the resulting release. Enter
`no` as this release is for testing purposes only.

Unpack the resulting tarball and run through the installer. Make sure the
software installs cleanly on both fresh systems and systems with previous
versions of the software.

Once the installer has been tested. edit the [VERSION](../VERSION) file
such that it contains the version number for the release you would like to
publish. After making the change, make a new git commit.
Do NOT push this commit up. The `make-release` script will handle this during
the publish step.

After updating the version, run `dev-scripts/make-release` again. The script
should run quickly as the resulting docker images should not have changed.
When asked whether you would like to publish the release, enter `yes`.
The script will then tag the latest commit with the version in the VERSION
file and push commit up with the new tag.

The script will then ask for your quay.io credentials. After entering
your username and password, the script will then publish the newly built
docker images to quay.io.

The `make-release` script will then exit. Now, you must go to the Github page
for the project and make a new release. Set the referenced tag for the release
to the version you entered into the VERSION file. Add a small write up for
the new version and attach the resulting tarball to the release.

### Replaying a PCAP with Netflow Data

While any PCAP can be converted into IPFIX records using YAF (see [Generating Data.md](./Generating%20Data.md)), PCAPs which contain raw Netflow v5/v9/IPFix
data may be replayed directly using the [replay-pcap.py script](../dev-scripts/replay-pcap.py).

The script requires **python3** and **scapy**. **scapy** can be installed by running `pip3 install scapy`.

```
./replay-pcap.py --help

This script reads in a [pcap-file], extracts the UDP packets sent to
[old-dest-ip] on [old-dest-port], and sends the data in the packets to
[new-dest-ip] on [new-dest-port].

Usage:
./replay-pcap.py pcap-file old-dest-ip old-dest-port new-dest-ip new-dest-port
```

If the PCAP was not captured on the same day as it was replayed, database rotation
must be disabled in the converter.

### Disabling Database Rotation

The converter automatically rotates the resulting RITA databases on a day by day basis.
In order to carry this out, the converter rejects any flows which are not timestamped
within the current day (Note: there is a small grace period in which flows from the
previous day are not rejected).

This presents a problem when processing data stored in a PCAP file.

Records stored in PCAP files can be converted into IPFIX flows using YAF. In this case, the timestamps
in the PCAP can be aligned to the current date using the [align_pcap_to_today.sh](../dev-scripts/align_pcap_to_today.sh) script before YAF performs the conversion.

However, this alignment can't be performed when replaying raw
Netflow v5/v9/IPFix data as recorded in a PCAP.

Restarting IPFix-RITA with the following commands prevents this problem from
occurring by disabling the database rotation.
```
sudo ipfix-rita rm -fs converter
sudo ipfix-rita -f /opt/ipfix-rita/lib/docker-compose/no-rotate.yaml up -d converter
```

IPFix-RITA will then place each stitched session into a database matching its
closing timestamp. These databases will not be ready for analysis until IPFix-RITA
is stopped i.e. the `ImportFinished` flag is not set on the created MetaDatabase
records until the program exits.
