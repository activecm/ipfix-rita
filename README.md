# IPFIX-RITA

IPFIX-RITA is a system for processing IPFIX/ Netflow v9 records for use with
[RITA](https://github.com/activecm/rita).


NOTICE: IPFIX-RITA is still in beta, it has only been tested on a limited number of devices.

# Installing IPFIX-RITA

### Preliminaries

First, ensure you have a working installation of RITA and MongoDB. For the best performance,
it is suggested that IPFIX-RITA is installed on a machine separate from RITA and MongoDB.
However, if performance is not a great concern, it is often practical to install everything
in one place.

**You must ensure the RITA database can be contacted on an IP address other than
localhost**. This can be done by editing the `bindIP` setting in `/etc/mongod.conf`.
The installer will prompt you to ensure this change is made before continuing
on.  If you intend to install IPFIX-RITA on the same machine as RITA
and MongoDB, please add the IP address suggested by the installer.

#### How to [Install RITA](https://github.com/activecm/rita#automatic-installation)

\
The IPFIX-RITA installer should run on most Linux distributions provided **Docker (min v17.06+)**
and **docker-compose (min v1.17)** are installed.
#### How to [Install Docker](https://docs.docker.com/install/)
#### How to [Install docker-compose](https://docs.docker.com/compose/install/).

### Installing

Download the latest archive from the [releases page](https://github.com/activecm/ipfix-rita/releases),
unpack it, and run the installation script with administrator privileges.

#### Download latest archive
```
$ wget $(curl --silent "https://api.github.com/repos/activecm/ipfix-rita/releases/latest" \
| grep '"browser_download_url":' | cut -d \" -f 4 ) -O ipfix-rita.tgz
```

#### Upack the installer
```
$ tar -zxf ipfix-rita.tgz
```

#### Run the installer
```
$ sudo $(ls -d */ | grep "ipfix-rita" | head -1)install-ipfix-rita.sh
```

You will be prompted for configuration details regarding the RITA database
connection and the names of the resulting datasets. Further configuration options
can be set in `/etc/ipfix-rita/converter/converter.yaml`.

By default, **IPFIX-RITA will run at start up unless it is stopped**. For more information
see [Additional Info](docs/Additional%20Info.md). Full documentation for IPFIX-RITA can be
found in the [docs](docs/) folder.

# IPFIX/ Netflow v9 Compatibility

This is an incomplete list of devices which produce compatible IPFIX/ Netflow v9 records.
More devices will be added as they are tested.

Please select the most basic version of IPFIX/Netflow v9 when setting up your router for
use with IPFIX-RITA.

|              | IPFIX | Netflow v9 |       Notes      |
|--------------|-------|------------|------------------|
|   Cisco ASA  |       |     ✔      |                  |
| Cisco ASR 9k |       |     ✔      |                  |
|   SonicWall  |       |     ✔      |                  |
|     YAF      |   ✔   |            | Use `--uniflow`  |

## What Do I Do If My Router Isn't On the List?

We need your help to expand the list of supported routers. Please help us by running
the software, logging the errors and traffic, and sending us the results. If you are
not comfortable emailing log files please contact us at support@activecountermeasures.com

Please see [Adding Support For Additional Routers](docs/Router%20Support.md) for more
information on gathering the data needed to get your device supported.
