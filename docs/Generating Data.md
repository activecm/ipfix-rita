# Generating IPFIX Data From a PCAP

### YAF (Yet Another Flowmeter)

From [https://tools.netsa.cert.org/index.html](https://tools.netsa.cert.org/index.html):

The Network Situational Awareness (NetSA) group at CERT has developed and
maintains a suite of open source tools for monitoring large-scale networks
using flow data. These tools have grown out of the work of the AirCERT
project, the SiLK project and the effort to integrate this work into a
unified, standards-compliant flow collection and analysis platform.

CERT is a part of the Software Engineering Institute (SEI), a federally
funded research and development center (FFRDC) operated by Carnegie Mellon University.

YAF is one of these open source monitoring tools. It is capable of creating
and exporting IPFIX records from PCAP files and network taps.

#### Installing
Before installing YAF, install its dependencies libpcap, and glib2, and libfixbuf.

libpcap and glib2 are likely to be up to date in your package manager.

The version of libfixbuf supported by your package manager is likely out of date.
At the time of writing, YAF 2.10.0 required libfixbuf 2.0.0, but
the Ubuntu Xenial repositories only held version 1.7.x.

In order to install the latest version of libfixbuf, head to the
[libfixbuf Download Page](https://tools.netsa.cert.org/fixbuf/download.html)
and download the latest release. Extract the archive, configure the build,
run the build, and install the resulting library with Make.

Now that the dependencies are installed, visit the
[YAF Download Page](https://tools.netsa.cert.org/yaf/download.html)
and download the latest version of the YAF source code. Finally,
extract the YAF archive, configure the build, run the build, and install
the resulting binaries with Make. You may need to run ldconfig to help the
dynamic linker find the libraries needed for YAF after installing.

Example script for installing YAF 2.10.0 on Ubuntu 16.04:
```
sudo apt update
sudo apt install build-essential wget
sudo apt install libpcap-dev libglib2.0-dev
wget https://tools.netsa.cert.org/releases/libfixbuf-2.0.0.tar.gz
tar xf libfixbuf-2.0.0.tar.gz
cd libfixbuf-2.0.0
./configure
make
sudo make install
cd ..
wget https://tools.netsa.cert.org/releases/yaf-2.10.0.tar.gz
tar -xf yaf-2.10.0.tar.gz
cd yaf-2.10.0
./configure
make
sudo make install
sudo ldconfig
```

#### Preparing a PCAP for use with IPFIX-RITA
NOTICE: If you already have a PCAP Active Countermeasures encourages you to convert the PCAP
to Bro logs. This can be accomplished by [generating PCAPs outside of Bro](https://github.com/activecm/rita/blob/master/Readme.md#obtaining-data-generating-bro-logs).
Before using YAF with IPFIX-RITA, you must ensure your PCAP file contains connection
records timestamped within the current day. In order to fix up any old PCAP, use the
`align_pcap_to_today.sh` script in the `dev-scripts` folder. The script takes in a
pcap, splits it up into 24 hour chunks, and aligns a given interval of the data to the current day.

Example usage:

```
ipfix-rita/dev-scripts$ ./align_pcap_to_today.sh ps_empire_https_python.pcap ps_empire_https_python_today.pcap
ps_empire_https_python.pcap consists of multiple 24H intervals.
Which interval would you like to use?
NOTE: The last interval may not contain a full 24H capture.

ps_empire_https_python.pcap contained 2 intervals.
        ps_empire_https_python_00000_20170815172141.pcap
        ps_empire_https_python_00001_20170816172145.pcap

File name:           /tmp/tmp.cSrLvbmX7B/ps_empire_https_python_00000_20170815172141.pcap
File type:           Wireshark/... - pcapng
Number of packets:   273 k
File size:           81 MB
First packet time:   2017-08-15 17:21:41.447486
Last packet time:    2017-08-16 17:21:40.272218
Align ps_empire_https_python_00000_20170815172141.pcap to today? (y/n) [n] y

Writing out ps_empire_https_python_today.pcap

ipfix-rita/dev-scripts$ capinfos ps_empire_https_python_today.pcap
File name:           ps_empire_https_python_today.pcap
File type:           Wireshark/... - pcapng
File encapsulation:  Ethernet
File timestamp precision:  microseconds (6)
Packet size limit:   file hdr: (not set)
Number of packets:   273 k
File size:           81 MB
Data size:           72 MB
Capture duration:    86398.824732 seconds
First packet time:   2018-11-01 00:00:00.447486
Last packet time:    2018-11-01 23:59:59.272218
Data byte rate:      839 bytes/s
Data bit rate:       6713 bits/s
Average packet size: 265.12 bytes
Average packet rate: 3 packets/s
SHA1:                8b8ed9f56abad4871a22562b35c53ae80b20f02a
RIPEMD160:           76ed30abd854b2673eecf98a37fb5514430fd394
MD5:                 d44550356003475d38f731866ad37b3d
Strict time order:   True
Capture application: Editcap 2.4.2
Number of interfaces in file: 1
Interface #0 info:
                     Encapsulation = Ethernet (1 - ether)
                     Capture length = 262144
                     Time precision = microseconds (6)
                     Time ticks per second = 1000000
                     Number of stat entries = 0
                     Number of packets = 273483

```
#### Extract Flow Data from PCAPs using YAF
YAF supports a variety of options, but most of it can be ignored when trying
to convert PCAP files to IPFIX records for use with IPFIX-RITA.

First, ensure the collector is running as you will need to specify
the the domain name or IP address of the collector when running YAF.

```
yaf --uniflow --in [input.pcap] --out [Collector Address] --ipfix-port 2055 --ipfix udp
```

Note: `--uniflow` is required to produce the unidirectional flows needed by
IPFIX-RITA. At the time of writing, most flow exporters do not support
[RFC5103 Bidirectional Flows](https://tools.ietf.org/html/rfc5103). YAF, however,
supports RFC5103 by default.
