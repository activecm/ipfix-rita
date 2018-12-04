# IPFIX-RITA

IPFIX-RITA is a system for processing IPFIX/Netflow v9/Netflow v5 records for
use with [RITA](https://github.com/activecm/rita).


NOTICE: IPFIX-RITA is still in beta, it has only been tested on a limited
number of devices.

# Installing IPFIX-RITA

### Preliminaries

First, ensure you have a working installation of RITA and MongoDB. For the best
performance, it is suggested that IPFIX-RITA is installed on a machine separate
from RITA and MongoDB. However, if performance is not a great concern, it is
often practical to install everything in one place.

**You must ensure the RITA database can be contacted on an IP address other
than localhost**. This can be done by editing the `bindIP` setting in
`/etc/mongod.conf`. The installer will prompt you to ensure this change is made
before continuing on.  If you intend to install IPFIX-RITA on the same machine 
as RITA and MongoDB, please add the IP address suggested by the installer.

#### How to [Install RITA](https://github.com/activecm/rita#automatic-installation)

\
The IPFIX-RITA installer should run on most Linux distributions provided
**Docker (min v17.06+)** and **docker-compose (min v1.17)** are installed.
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
connection and the names of the resulting datasets. Further configuration
options can be set in `/etc/ipfix-rita/converter/converter.yaml`.

By default, **IPFIX-RITA will run at start up unless it is stopped**. For more 
information see [Additional Info](docs/Additional%20Info.md). Full
documentation for IPFIX-RITA can be found in the [docs](docs/) folder.

# IPFIX/Netflow v9/Netflow v5 Compatibility

This is an incomplete list of devices which produce compatible
IPFIX/Netflow v9/Netflow v5 records. More devices will be added as they are
tested.

Please select the most basic version of IPFIX/Netflow v9/Netflow v5 when
setting up your router for use with IPFIX-RITA.

|              | IPFIX | Netflow v9 | Netflow v5 |       Notes      |
|--------------|-------|------------|------------|------------------|
|   Cisco ASA  |       |     ✔      |            |                  |
| Cisco ASR 9k |       |     ✔      |            |                  |
|   SonicWall  |       |     ✔      |            |                  |
|   MikroTik   |       |            |     ✔      |                  | 
|     YAF      |   ✔   |            |            | Use `--uniflow`  |

## What Do I Do If My Router Isn't On the List?

We need your help to expand the list of supported routers. Please help us by
running the software, logging the errors and traffic, and sending us the
results. If you are not comfortable emailing log files please contact us at
support@activecountermeasures.com

Please see [Adding Support For Additional Routers](docs/Router%20Support.md) for more
information on gathering the data needed to get your device supported.

# Troubleshooting
### Testing IPFix/Netflow Records
To test that IPFix/Netflow records are arriving at you IPFIX-RITA system, run
the following on the IPFIX-RITA system:
```
$ tcpdump -qtnp 'udp port 2055'
```
Though the actual IP addresses and length will be different, you should see
lines like:
```
IP 10.0.0.5:2055 > 10.0.0.43:2055: UDP, length 212
```
arriving somewhat regularly. Press Ctrl-C to exit. If you don't get any of
these lines after a minute or so, your router may not be configured correctly
to send these records to the ipfix-rita system. Double check your router
configuration; make sure it's sending records to the IPFIX-RITA system's IP
address and to UDP port 2055.

### Ensure Docker Containers are Running
To make sure that all the docker containers are running correctly on the
IPFIX-RITA system, run the following on that system:
```
sudo docker ps
```
You should get a header line starting with "CONTAINER ID" and then at least
three lines of running ocntainers with a status on "Up (some amount of time)".
The names of these containers should start with "ipfix_rita_logstash",
"ipfix_rita_converter", and "ipfix_rita_mongodb". If you do not get these three
lines, somehting may be wrong with the docker intances, please contact
technical support.

### Checking that IPFIX-RITA is Creating Mongo Databases
To see if IPFIX-RITA is creating mongo databases, first find the container ID
for the IPFIX-RITA-mongodb contianter. It's the 12 character hex string at the
left of the ipfix_rita_mongodb... from the above docker command. Now run:
```
sudo docker exec -it **12_char_hex_id** mongo
```
You'll find yourself in a command prompt that accepts mongo commands. Type:
```
show dbs
```
Which will list the available databases. If you see:
```
IPFIX   0.000GB
admin   0.000GB
config  0.000GB
local   0.000GB
```
That means you're not yet saving data; skip to the next section to see why. If
your output also includes "Metadatabase" and "IPFIX-YYMMDD" databases, that's a
good sign. To get out of this terminal type "exit".

### Checking for Errors from IPFIX-RITA
To see if there are any error reported by IPFIX-RITA, run
```
sudo ipfix-rita logs | grep -i 'erro'
```
Any errors that show up here should be sent to technical support. Please
include a brief descript of the router or filewall that's sending the IPFix
records, as well as what type of records these are (Netflow V5, Netflow V9, or
IPFix).

