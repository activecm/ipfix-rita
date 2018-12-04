# Thank You For Your Interest In Helping Us Support More Devices

The IPFIX-RITA project needs your help in order to support more
IPFIX/Netflow v9/Netflow v5 enabled devices. In this guide, we will walk
through gathering an error log, capturing relevant network traffic, and
packaging up the results for analysis by the Active Countermeasures team.

First off, run the installer as explained in the [Readme](../README.md).

### Automatic Data Collection
The [ipfix-rita-debug script](../user-scripts/ipfix-rita-debug.py) may be used
to automatically collect the information needed to support your router. The
script will ensure data is arriving on UDP port 2055, capture that traffic,
monitor the IPFIX-RITA logs for errors, and record the application log if an
error is found. If no errors are found, all of the recorded data is deleted.

Download the script by cloning the repository or by running:
```
wget https://raw.githubusercontent.com/activecm/ipfix-rita/master/user-scripts/ipfix-rita-debug.py
chmod +x ipfix-rita-debug.py
```

The `ipfix-rita-debug.py` script must be run with administrator privileges in
order to record a packet capture and start/ stop IPFIX-RITA. The script
requires Python 3.4 or above which should be available for most modern Linux
distributions.

If errors are found, the script will create an archive with the data it
collected. Please email this archive to support@activecountermeasures.com along
with a description of the device you are trying to use. Additionally, please
include any relevant settings you have set on your device such as which flow
version you are using (IPFIX, Netflow v9, or Netflow v5), if you are sending
any "additional reports" (proprietary data), and the flow reporting mode being
used (realtime/ bulk/ etc.).

If no errors are found and your device is not listed in the compatibility
matrix in the main [README](../README.md), please send an email to
support@activecountermeasures.com containing a description of the device you
are using and any relevant settings.

### Manual Data Collection
If you want to manually collect the data instead of using the `ipfix-rita-debug.py` script, please follow the steps below.

After IPFIX-RITA is installed and running, ensure your
IPFIX/Netflow v9/Netflow v5 device is actively sending records to your
collector using tcpdump. Tcpdump can be installed using your system package
manager. Once tcpdump is installed, run `tcpdump -i 
[IPFIX/Netflow v9/Netflow v5 Interface] 'udp port 2055'`. This will bring up a
live stream of the IPFIX/Netflow v9/Netflow v5 data entering your system.
Before continuing, ensure you see active IPFIX/Netflow v9/Netflow v5 traffic
appearing in your terminal. You can exit out of tcpdump by hitting `CTRL-C`.

If tcpdump does not display any IFPIX/Netflow v9/Netflow v5 traffic, please
ensure you have correctly configured your IPFIX/Netflow v9/Netflow v5 enabled
device and that the connection between your device and IPFIX-RITA system is
working as it should.

Now we can begin gathering the data needed to support your device.

Note: You will likely need to open up two terminal sessions.

In the first terminal:
- Restart IPFIX-RITA by running `sudo ipfix-rita stop` and `sudo ipfix-rita up -d`
- Begin capturing an error log by running `sudo ipfix-rita logs -f | tee ipfix-rita-errs.log`
- Wait a a few minutes and see if any error messages come up
- If no error messages come up, good news! Your device may be supported but undocumented.
  - On your RITA system, run `rita show-databases`. If you see a new database with today's date, everything is working as it should!
  - Let us know the type of device you're using and any relevant settings you have set on your device by sending us an email at support@activecountermeasures.com
- If you see some error messages, don't worry. Continue through the guide, and we will work with you to fix everything up.
- Go ahead, and leave the error log running

Open up another terminal session and follow these steps:
- Begin a packet capture using `sudo tcpdump -i [IPFIX/Netflow v9/Netflow v5 Interface] -C 50 -w ipfix-rita-debug.pcap -s 0 'udp port 2055'`
- Leave the packet capture running

Once both the error log and packet capture are running, continue to let them run for five minutes or so. After some time has elapsed, hit `CTRL-C` in both terminal sessions.

In either terminal:
- Stop IPFIX-RITA: `ipfix-rita stop`
- Make a directory to hold the error log and packet capture: `mkdir ipfix-rita-debug`
- Move the error log into the folder: `mv /path/to/ipfix-rita-errs.log ./ipfix-rita-debug`
- Move the first packet capture into the folder: `mv /path/to/ipfix-rita-debug.pcap1 ./ipfix-rita-debug`
- Compress the folder: `tar czf ipfix-rita-debug.tgz ipfix-rita-debug`

Finally, send us an email at support@activecountermeasures.com with the name of
the device you are using, the settings you have set on your device pertaining
to IPFIX/Netflow v9/Netflow v5, and the compressed data we just gathered.
