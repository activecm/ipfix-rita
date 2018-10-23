# IPFIX-RITA

IPFIX-RITA is a system for processing IPFIX/ Netflow v9 records for use with
[RITA](https://github.com/activecm/rita).

# Structure

IPFIX-RITA is made up of four components. These are the

- Collector (Logstash)
  - Transforms IPFIX/ Netflow v9 records into records compatible with the Converter
- Buffer (MongoDB)
  - Used to buffer records created by the collector until they are read by the Converter
- Converter (Written with Go)
  - Converts unidirectional flow data into bidirectional connection records for use with RITA
- RITA database (MongoDB)
  - Holds data processed by the Converter

# Running IPFIX-RITA

IPFIX-RITA is managed by `/opt/ipfix-rita/bin/ipfix-rita`. This script relays
commands to `docker-compose` and finally, `docker`.

IPFIX-RITA will start automatically after installation.
In order to stop the system, run `ipfix-rita stop`. This will shut down
IPFIX-RITA and prevent the program from starting when the system boots up.
To bring the system back up, run `ipfix-rita up -d`.

To view the IPFIX-RITA logs, run `ipfix-rita logs`.

When IPFIX-RITA receives IPFIX or Netflow v9 records timestamped with the
current date, it will begin writing records into the resulting RITA dataset.
Every night at 5 minutes past midnight local time, the dataset will be closed,
and it will become eligible for analysis by RITA. The resulting datasets will
have names of the form `DBRoot-YYYY-MM-DD` where `DBRoot` is set during
installation or configured in `/etc/ipfix-rita/converter/converter.yaml`.

# Compatibility

This is an incomplete list of devices which produce compatible IPFIX/ Netflow v9 records. More devices will be added as they are tested.

Please select the most basic version of IPFIX/ Netflow v9 when setting up your router for use with IPFIX-RITA.

|              | IPFIX | Netflow v9 |       Notes      |
|--------------|-------|------------|------------------|
|   Cisco ASA  |       |     ✔      |                  |
| Cisco ASR 9k |       |     ✔      |                  |
|   SonicWall  |   ✔   |            |                  |
|     YAF      |   ✔   |            | Use `--uniflow`  |


# Installing IPFIX-RITA

### Preliminaries

First, ensure you have a working installation of RITA and MongoDB. For the best performance,
it is suggested that IPFIX-RITA is installed on a machine separate from RITA and MongoDB. However, if performance isn't a great concern, it is often practical to install everything in one place.

Either way, you must ensure the RITA database can be contacted on an IP address other than
localhost. This can be done by editing the `bindIP` setting in `/etc/mongod.conf`.
The installer will prompt you to ensure this change is made before continuing
on.  If you intend to install IPFIX-RITA on the same machine as RITA
and MongoDB, please add the IP address suggested by the installer.

### Running the Installer

Download the latest archive from the [releases page](https://github.com/activecm/ipfix-rita/releases), unpack it, and run the
installation script with administrator privileges.

The installer should run on most modern Linux distributions as long as
Docker and docker-compose are installed. The minimum supported versions of
Docker and docker-compose are 17.06 and 1.17, respectively.

The installer will install Docker images for the components listed above, and
create two new folders.

`/opt/ipfix-rita` contains the controller script for IPFIX-RITA and supporting
code. The main script will be located at `/opt/ipfix-rita/bin/ipfix-rita`.

`/etc/ipfix-rita` contains the configuration files needed to run IPFIX-RITA.

As the installer runs, it will prompt for configuration details regarding the RITA database
connection and the names of the resulting datasets. Further configuration options
can be set in `/etc/ipfix-rita/converter/converter.yaml` such as the CIDR
ranges for local networks (by default all class A, B, and C networks are considered local).

Finally, the installer will initialize and start the system. IPFIX-RITA
will begin listening for IPFIX and Netflow v9 traffic on UDP port 2055 on
the Docker host.

By default, IPFIX-RITA will run at start up unless it is stopped.

# Uninstalling IPFIX-RITA

```
/opt/ipfix-rita/bin/ipfix-rita down -v
sudo rm -rf /opt/ipfix-rita /etc/ipfix-rita
```

---

#### Developer Notes

The converter may be built outside of Docker using the `Makefile` in the
`converter/` directory. Before running the converter ensure you have a config
file installed in `/etc/ipfix-rita/converter/converter.yaml`. This may be done
one of three ways. First, you could manually copy `runtime/etc/converter/converter.yaml`
to `/etc/ipfix-rita/converter/converter.yaml`. Alternatively, you could run `make install`
to install the converter software natively. Or, finally, you could run the release installer.

Once the configuration file has been installed, the converter executable will be able to run on its own. Additionally, `runtime/bin/ipfix-rita` should work to control the dockerized system as a whole.
If you'd like to make a development build of the dockerized system run `runtime/bin/ipfix-rita build`.

The `dev-scripts/make-release` script is used to produce a release tarball.
