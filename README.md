# IPFIX-RITA

IPFIX-RITA is a system for processing IPFix/Netflow v9/Netflow v5 records for
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
before continuing on. If you intend to install IPFIX-RITA on the same machine
as RITA and MongoDB, please add the IP address suggested by the installer.
\
NOTE: if you want multiple bind IP addresses in your MongoDB config file you
must place each of them on the same line separated by commas. For example if
you want both `10.0.0.5` and `172.20.0.1` as bind IP's (one for RITA and one
for IPFIX-RITA to access) your bind IP line should look like the following.
`  bindIP: 10.0.0.5,172.20.0.1`
\
Also if your RITA config file (`/etc/rita/config.yaml`) connects to MongoDB on
localhost you will need to change that to the same value as MongoDB is listening
on. For example if you change the bindIP in your MongoDB config file to 10.0.0.5
and you check your RITA config file and the connection string is
`mongodb://localhost:27017` you'll need to change it to ` mongodb://10.0.0.5:27017`.

#### How to [Install RITA](https://github.com/activecm/rita#automatic-installation)

\
The IPFIX-RITA installer should run on most Linux distributions provided
**Docker (min v17.06, max 18.09)** and **docker-compose (min v1.17, max v1.23)**
are installed.
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

#### Unpack the installer
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

Once this is complete you can direct your IPFix or Netflow v5/v9 logs to
your IPFIX-RITA machine that is listening for logs on **UDP port 2055**.

By default, **IPFIX-RITA will run at start up unless it is stopped**. For more
information see [Additional Info](docs/Additional%20Info.md). Full
documentation for IPFIX-RITA can be found in the [docs](docs/) folder.

# Useful Commands
Below are some useful commands to use with the IPFIX-RITA docker containers.

```
sudo ipfix-rita stop
```
This will stop all of the ipfix-rita containers, if you see a lot of errors
running it is sometimes helpful to stop the containers so you don't get a
ton of errors stored in your logs while you check other components.

```
sudo ipfix-rita restart
```
Will stop and restart the ipfix-rita containers. If the container was
already stopped it will start it back up. This can be helpful if a
connection was lost to regain that connection or to reload variables
that were changed in the config file.

```
sudo ipfix-rita log
```
Will display all the logs from the IPFIX-RITA, this includes Info, Warning,
and Error logs, it can be a lot so we recommend running it with tail. This
is useful because it allows us to see what is happening in IPFIX-RITA at
any given time

# IPFix/Netflow v9/Netflow v5 Compatibility

This is an incomplete list of devices which produce compatible
IPFix/Netflow v9/Netflow v5 records. More devices will be added as they are
tested.

Please select the most basic version of IPFix/Netflow v9/Netflow v5 when
setting up your router for use with IPFIX-RITA.

|              | IPFix | Netflow v9 | Netflow v5 |       Notes      |
|--------------|-------|------------|------------|------------------|
|   Cisco ASA  |       |     ✔      |            |                  |
| Cisco ASR 9k |       |     ✔      |            |                  |
|   SonicWall  |       |     ✔      |            |                  |
|   MikroTik   |   ✔   |     ✔      |     ✔      |                  |
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
To test that IPFix/Netflow records are arriving at your IPFIX-RITA system, run
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
to send these records to the IPFIX-RITA system. Double check your router
configuration; make sure it's sending records to the IPFIX-RITA system's IP
address and to UDP port 2055.

### Ensure Docker Containers are Running
To make sure that all the docker containers are running correctly on the
IPFIX-RITA system, run the following on that system:
```
sudo ipfix-rita ps
```
You should get a header line starting with "CONTAINER ID" and then at least
three lines of running containers with a status on "Up (some amount of time)".
The names of these containers should start with "ipfix_rita_logstash",
"ipfix_rita_converter", and "ipfix_rita_mongodb". If you do not get these three
lines, something may be wrong with the docker instances, please contact
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
it might mean that you are logging to a different database, use the
ConnectionString value under RITA-MongoDB in
/etc/ipfix-rita/converter/converter.yaml .  For example, if your file looks like this:
```
Output:
  RITA-MongoDB:
    MongoDB-Connection:
      # See https://docs.mongodb.com/manual/reference/connection-string/
      ConnectionString: mongodb://10.0.0.5:27017
      # Accepted Values: null, "SCRAM-SHA-1", "MONGODB-CR"
      AuthenticationMechanism: null
      TLS:
        Enable: false
        VerifyCertificate: false
        CAFile: null
...
```
try connecting to mongo using
```
mongo [ipaddress]:[port]
mongo 10.0.0.5:27017
```
and running the command again.  If you still see:
```
admin   0.000GB
config  0.000GB
local   0.000GB
```
That means you're not yet saving data; skip to the next section to see why. If
your output also includes "Metadatabase" and "IPFIX-YYMMDD" databases, that's a
good sign. To get out of this terminal type "exit".

### Ensuring Your RITA Machine is connected to IPFIX-RITA
To check if you have an ongoing connection with your IPFIX-RITA machine run
the following command from your RITA machine
```
netstat -an | grep :27017
```
If your IPFIX-RITA machine's address is `192.168.0.6` and your RITA machine's
addess is `10.0.0.5` you should see something like the following pop up:
```
tcp        0      0 0.0.0.0:27017           0.0.0.0:*               LISTEN     
tcp        0      0 10.0.0.5:27017          192.168.0.6:47486       ESTABLISHED
tcp        0      0 10.0.0.5:27017          192.168.0.6:47476       ESTABLISHED
```
That means RITA is connecting to IPFIX-RITA (since the IPFIX-RITA address is
listed), however if you only see 
```
tcp        0      0 0.0.0.0:27017           0.0.0.0:*               LISTEN     
```
Then your RITA box is listening for incoming data but no connection to
the IPFIX-RITA machine has been established. Double check the values in
your MongoDB config file on your RITA machine (/etc/mongod.conf by
default) as well as your RITA connection settings for IPFIX-RITA
in the IPFIX-RITA config file at /etc/ipfix-rita/converter/converter.yaml

The values for ConnectionString should be `mongodb://ip.address.for.rita:mongoPort`
AuthenticationMechanism, TLS Enable, TLS VerifyCertificate and CAFile should all be
the same as your mongod.conf file on your RITA machine. In addition, if present the
CAFile should be copied on your IPFIX-RITA machine too.

### Checking if Data is Being Sent to IPFIX-RITA
To Check if data is arriving at the IPFIX-RITA box, run
```
sudo ipfix-rita logs | grep "new data"
```
If you see
```
converter_1  | INFO[0090] reading new data from input buffer
```
You have recieved some logs, however if you are still recieving data requires
counting lines. An easy way to do this is to run
```
sudo ipfix-rita logs | grep "new data" | wc -l
```
which will return a number (say 245), that is the number of times the line shows
up in your logs. Now wait 5-10 minutes and run the command again, it should
increase. If the value didn't, then you might not be recieving logs. Double
check your router is sending IPFix/Netflow data, and if the problem persists
contact support@activecountermeasures.com

### Checking if RITA is Recieving Records
To check that RITA is not only recieving records but storing them for
threat-hunting requires us to look directly into mongo. So start by
loading mongo, if you don't have users and passwords enabled run 
```
mongo
```
on your RITA machine to attach to the local mongo instance then run
```
show dbs
```
to show all of the databases stored. Normally IPFIX-RITA uses the form
IPFIX-\[YYYY-MM-DD\] to store your databases so if you see
```
IPFIX-2019-12-24  0.002GB
IPFIX-2018-12-25  0.053GB
IPFIX-2018-12-26  0.034GB
MetaDatabase      0.000GB
admin             0.000GB
config            0.000GB
local             0.000GB
rita-bl           0.017GB
```
Type the following command to tell mongo to use the current day
```
use IPFIX-2018-12-26
```
then to see how many connections are currently stored and if we
are updating them run the following
```
db.conn.find().count()
```
and you should see a number printed like so
```
9759
```
That's the number of connections currently stored in the connection
collection of the mongo database. Since IPFIX-RITA takes a little
to process the data if you run the same command again in a short time
you'll likely still see the same number, so wait 10-15 minutes and run
the command again. You should see the number go up, if the number don't
increase then IPFIX-RITA isn't storing data in RITA. It could be caused
by a number of problems. Check the other issues in this document and if
the problem persists contact support@activecountermeasures.com

### Checking for Errors from IPFIX-RITA
To see if there are any errors reported by IPFIX-RITA, run
```
sudo ipfix-rita logs | grep -i 'erro'
```
If there are too many errors, simply run
```
sudo ipfix-rita logs --tail 20 -f | grep -i 'erro' [> error_report.txt]
```

Any errors that show up here (or the error_report.txt file) should be sent
to technical support at support@activecountermeasures.com. Please
include a brief description of the router or firewall that's sending the IPFix
records, as well as what type of records these are (Netflow v5, Netflow v9, or
IPFix).

# BUG NOTICE
The following bugs have been documented by Active Countermeasures and solutions
are in development

### IPFix-RITA Fails on Reboot
If IPFix-RITA is configured to write to a MongoDB database running on the
Docker host (as in the default configuration), IPFix-RITA may encounter
an error after a system reboot.
The following error may arise:
```
converter_1_eeb48d380f26 | ERRO[0038] could not connect to RITA MongoDB: could not connect to MongoDB (no TLS): could not connect to MongoDB: no reachable servers  stacktrace="[rita_dates.go:60 convert.go:194 convert.go:40 app.go:490 command.go:210 app.go:255 boot.go:18 proc.go:198 asm_amd64.s:2361]"
```
This is due to an error in which the MongoDB server starts before the Docker engine.
Until a fix is implemented, Active Countermeasures recommends running the following command
to resolve the issue
```
sudo systemctl restart mongod.service
```

### Log Rotation Error
It has been discovered that some flow logs will report flow start and end
times in UTC+00:00. This means if you don't live in the UTC+00:00
timezone you may see logs in MongoDB from a future date or not rotating when
you expect them to. Until a fix is implemented, Active Countermeasures asks you
to be aware of the issue.
