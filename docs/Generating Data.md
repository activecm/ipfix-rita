# Generating IPFIX Data

### YAF (Yet Another Flowmeter)

From [https://tools.netsa.cert.org/index.html](https://tools.netsa.cert.org/index.html):

The Network Situational Awareness (NetSA) group at CERT has developed and maintains a suite of open source tools for monitoring large-scale networks using flow data. These tools have grown out of the work of the AirCERT project, the SiLK project and the effort to integrate this work into a unified, standards-compliant flow collection and analysis platform.

CERT is a part of the Software Engineering Institute (SEI), a federally funded research and development center (FFRDC) operated by Carnegie Mellon University.

YAF is one of these open source monitoring tools. It is capable of creating
and exporting IPFIX records from PCAP files and network taps.

#### Installing

In order to install YAF, visit the [YAF Download Page](https://tools.netsa.cert.org/yaf/download.html) and download the latest
version of the YAF source code. Next, install the required development files
for libpcap, libfixbuf3, and glib2. Finally, extract the YAF archive, configure
the build, make the build, and install the resulting binaries with Make. You
may need to run ldconfig to help the dynamic linker find the libraries needed
for YAF after installing.

Example script for installing YAF 2.10.0 on Ubuntu 16.04:
```
apt install build-essential wget
apt install libpcap-dev libfixbuf3-dev libglib2.0-dev
wget https://tools.netsa.cert.org/releases/yaf-2.10.0.tar.gz
tar -xf yaf-2.10.0.tar.gz
cd yaf-2.10.0
./configure
make
sudo make install
sudo ldconfig
```

#### Extract Flow Data from PCAPs using YAF
YAF supports a variety of options, but most of it can be ignored when trying
to convert PCAP files to IPFIX records for use with IPFIX-RITA.
```
yaf --uniflow --in [input.pcap] --out [IPFIX-RITA Address] --ipfix-port 2055 --ipfix udp
```
Note: `--uniflow` is required to produce the unidirectional flows needed by
IPFIX-RITA. At the time of writing, most flow exporters do not support
[RFC5103 Bidirectional Flows](https://tools.ietf.org/html/rfc5103). YAF, however,
supports RFC5103 by default.
