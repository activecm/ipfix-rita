# Generating IPFIX Data

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
Before using YAF with IPFIX-RITA, you must ensure your PCAP file contains connection
records timestamped within the current day. In order to fix up any old PCAP:

- Find the start and end times of an existing PCAP file using `capinfos [pcap file]`
- Split the pcap into 24H periods `editcap -i 86400 [in pcap file] [out pcap file]`
   - `[out pcap file]` should be something of the form `my_pcap_name.pcap`. `editcap` will insert date information into the output file names right before the file extension.
- Align a 24H PCAP with the current day
  - One liner: `editcap -t $(expr $(date +%s --date="$(date -I)") -  $(date +%s --date="$(capinfos [split pcap file] | grep 'First packet time' | cut -d' ' -f6-)")) [split pcap file] [out pcap file]`
  - Explanation:
      - Find the current day's unix timestamp `date +%s --date="$(date -I)"`
      - Find the start of the pcap's date: `capinfos [split pcap file] | grep 'First packet time' | cut -d' ' -f6-`
          - This could be done manually by inspecting the output of `capinfos [split pcap file]`
      - Find the unix timestamp for the start of the pcap: `date +%s --date="[date output from capinfos]"`
      - Subtract the two timestamps using `expr`
      - Pass the difference in time to `editcap` in order to shift all of the records to the current day

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
