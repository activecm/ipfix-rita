#!/usr/bin/env python3

################################################################################
# This script reads in a [pcap-file], extracts the UDP packets sent to
# [old-dest-ip] on [old-dest-port], and sends the data in the packets to
# [new-dest-ip] on [new-dest-port].
#
# Usage:
# ./replay-pcap.py pcap-file old-source-ip old-dest-ip old-dest-port new-dest-ip new-dest-port
#
#
# Dependencies:
# python3, scapy
################################################################################

import sys
import socket
import time
from scapy.all import PcapReader, IP, UDP

MAX_PACKETS_PER_SECOND = 10

def print_usage():
    print(
"""This script reads in a [pcap-file], extracts the UDP packets sent to
[old-dest-ip] on [old-dest-port], and sends the data in the packets to
[new-dest-ip] on [new-dest-port].

Usage:
./replay-pcap.py pcap-file old-dest-ip old-dest-port new-dest-ip new-dest-port"""
    )

def main():
    if len(sys.argv) != 6:
        print_usage()
        return 1
    try:
        pcap_file = sys.argv[1]
        old_dst_ip = sys.argv[2]
        old_dst_port = int(sys.argv[3])
        new_dst_ip = sys.argv[4]
        new_dst_port = int(sys.argv[5])
    except Exception as e:
        print(e)
        print_usage()
        return 1

    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)

    try:
        print("Opening {0} for reading...".format(pcap_file))
        print(flush=True)
        packet_reader = PcapReader(pcap_file)
    except FileNotFoundError:
        print("Could not read {0}".format(pcap_file))
        return


    print("Sending UDP data that was sent to {0}:{1} to {2}:{3}".format(
        old_dst_ip,
        old_dst_port,
        new_dst_ip,
        new_dst_port
    ))
    print("+: a packet was matched and sent")
    print("-: a packet was skipped")
    print("")
    for old_packet in packet_reader:
        if not (
            UDP in old_packet and
            old_packet[IP].dst == old_dst_ip and
            old_packet[IP].dport == old_dst_port
        ):
            print("-", end="", flush=True)
            continue

        sock.sendto(bytes(old_packet[UDP].payload), (new_dst_ip, new_dst_port))
        print("+", end="", flush=True)

        time.sleep(1 / MAX_PACKETS_PER_SECOND)

if __name__ == "__main__":
    try:
        sys.exit(main())
    except KeyboardInterrupt:
        print()
        sys.exit(0)
