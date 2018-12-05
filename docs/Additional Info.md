# Installer Information

The installer will install Docker images for its components (collector, buffer, converter and
RITA Database), and create the following new folders.

`/opt/ipfix-rita` contains the controller script for IPFIX-RITA and supporting
code. The main script will be located at `/opt/ipfix-rita/bin/ipfix-rita`.

`/etc/ipfix-rita` contains the configuration files needed to run IPFIX-RITA.

As the installer runs, it will prompt for configuration details regarding the RITA database
connection and the names of the resulting datasets. Further configuration options
can be set in `/etc/ipfix-rita/converter/converter.yaml` such as the CIDR
ranges for local networks (by default all class A, B, and C networks are considered local).

The installer will initialize and start the system. IPFIX-RITA will begin
listening for IPFix and Netflow v9/v5 traffic on UDP port 2055 on the Docker
host.

# Running IPFIX-RITA

IPFIX-RITA is managed by `/opt/ipfix-rita/bin/ipfix-rita`. This script relays
commands to `docker-compose` and finally, `docker`.

IPFIX-RITA will start automatically after installation.

## Stopping IPFIX-Rita

```
ipfix-rita stop
```
This will shut down IPFIX-RITA and prevent the program from starting when
the system boots up.

## Restarting IPFIX-RITA

```
ipfix-rita up -d
```
This will bring IPFIX-RITA and allow the program to start on system boot

## Viewing the IPFIX-RITA logs

```
ipfix-rita logs
```

## IPFIX-RITA logging information

When IPFIX-RITA receives IPFix or Netflow v9/v5 records timestamped with the
current date, it will begin writing records into the resulting RITA dataset.
Every night at 5 minutes past midnight local time, the dataset will be closed,
and it will become eligible for analysis by RITA. The resulting datasets will
have names of the form `DBRoot-YYYY-MM-DD` where `DBRoot` is set during
installation or configured in `/etc/ipfix-rita/converter/converter.yaml`.

# Uninstalling IPFIX-RITA

Shutdown IPFIX-RITA
```
/opt/ipfix-rita/bin/ipfix-rita down -v
```

Remove IPFIX-RITA Binary
```
sudo rm /usr/local/bin/ipfix-rita
```

Remove all IPFIX-RITA files and folders
```
sudo rm -rf /opt/ipfix-rita /etc/ipfix-rita
```

---
