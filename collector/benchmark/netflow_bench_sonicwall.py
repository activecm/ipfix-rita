#!/usr/bin/env python2
import socket
import sys
import time


# IPFIX template
tpl = "00090001000265485b6b4e5100000404a07e8c0000000048010000100001000400020004000400010008000400070002000a0004000b0002000c0004000e0004000f0004001500040016000400e1000400e2000400e3000200e40002".decode("hex")

'''
Cisco NetFlow/IPFIX
    Version: 9
    Count: 1
    SysUptime: 157.000000000 seconds
    Timestamp: Aug  8, 2018 14:10:57.000000000 MDT
        CurrentSecs: 1533759057
    FlowSequence: 1028
    SourceId: 2692647936
    FlowSet 1 [id=0] (Data Template): 256
        FlowSet Id: Data Template (V9) (0)
        FlowSet Length: 72
        Template (Id = 256, Count = 16)
            Template Id: 256
            Field Count: 16
            Field (1/16): BYTES
                Type: BYTES (1)
                Length: 4
            Field (2/16): PKTS
                Type: PKTS (2)
                Length: 4
            Field (3/16): PROTOCOL
                Type: PROTOCOL (4)
                Length: 1
            Field (4/16): IP_SRC_ADDR
                Type: IP_SRC_ADDR (8)
                Length: 4
            Field (5/16): L4_SRC_PORT
                Type: L4_SRC_PORT (7)
                Length: 2
            Field (6/16): INPUT_SNMP
                Type: INPUT_SNMP (10)
                Length: 4
            Field (7/16): L4_DST_PORT
                Type: L4_DST_PORT (11)
                Length: 2
            Field (8/16): IP_DST_ADDR
                Type: IP_DST_ADDR (12)
                Length: 4
            Field (9/16): OUTPUT_SNMP
                Type: OUTPUT_SNMP (14)
                Length: 4
            Field (10/16): IP_NEXT_HOP
                Type: IP_NEXT_HOP (15)
                Length: 4
            Field (11/16): LAST_SWITCHED
                Type: LAST_SWITCHED (21)
                Length: 4
            Field (12/16): FIRST_SWITCHED
                Type: FIRST_SWITCHED (22)
                Length: 4
            Field (13/16): postNATSourceIPv4Address
                Type: postNATSourceIPv4Address (225)
                Length: 4
            Field (14/16): postNATDestinationIPv4Address
                Type: postNATDestinationIPv4Address (226)
                Length: 4
            Field (15/16): postNAPTSourceTransportPort
                Type: postNAPTSourceTransportPort (227)
                Length: 2
            Field (16/16): postNAPTDestinationTransportPort
                Type: postNAPTDestinationTransportPort (228)
                Length: 2
'''

data = "00090005000e1d485b6b515100000c09a07e8c000100010d0000045d0000000511acd90a2e01bb0000000224d70a0000ed000000010a000001000d9878000d5de0acd90a2ec0a8a84101bbc0c6000000ae00000001114b4b4c4c003500000002d21e0a0000ed000000010a000001000da818000da8184b4b4c4cc0a8a8410035ec7d000000e800000001114b4b4c4c0035000000023c1b0a0000ed000000010a000001000da818000da8184b4b4c4cc0a8a8410035c6c30000039d0000000611acd9c5bd01bb00000002e7530a0000ed000000010a000001000da048000da048acd9c5bdc0a8a84101bbd96a000000dd00000001114b4b4c4c003500000002d7b50a0000ed000000010a000001000da818000da8184b4b4c4cc0a8a8410035cff4".decode("hex")

'''
FlowSet 1 [id=256] (5 flows)
    FlowSet Id: (Data) (256)
    FlowSet Length: 269
    [Template Frame: 841]
    Flow 1
        Octets: 1117
        Packets: 5
        Protocol: UDP (17)
        SrcAddr: 172.217.10.46
        SrcPort: 443 (443)
        InputInt: 2
        DstPort: 9431 (9431)
        DstAddr: 10.0.0.237
        OutputInt: 1
        NextHop: 10.0.0.1
        [Duration: 15.000000000 seconds (switched)]
            StartTime: 876.000000000 seconds
            EndTime: 891.000000000 seconds
        Post NAT Source IPv4 Address: 172.217.10.46
        Post NAT Destination IPv4 Address: 192.168.168.65
        Post NAPT Source Transport Port: 443
        Post NAPT Destination Transport Port: 49350
    Flow 2
        Octets: 174
        Packets: 1
        Protocol: UDP (17)
        SrcAddr: 75.75.76.76
        SrcPort: 53 (53)
        InputInt: 2
        DstPort: 53790 (53790)
        DstAddr: 10.0.0.237
        OutputInt: 1
        NextHop: 10.0.0.1
        [Duration: 0.000000000 seconds (switched)]
            StartTime: 895.000000000 seconds
            EndTime: 895.000000000 seconds
        Post NAT Source IPv4 Address: 75.75.76.76
        Post NAT Destination IPv4 Address: 192.168.168.65
        Post NAPT Source Transport Port: 53
        Post NAPT Destination Transport Port: 60541
    Flow 3
        Octets: 232
        Packets: 1
        Protocol: UDP (17)
        SrcAddr: 75.75.76.76
        SrcPort: 53 (53)
        InputInt: 2
        DstPort: 15387 (15387)
        DstAddr: 10.0.0.237
        OutputInt: 1
        NextHop: 10.0.0.1
        [Duration: 0.000000000 seconds (switched)]
            StartTime: 895.000000000 seconds
            EndTime: 895.000000000 seconds
        Post NAT Source IPv4 Address: 75.75.76.76
        Post NAT Destination IPv4 Address: 192.168.168.65
        Post NAPT Source Transport Port: 53
        Post NAPT Destination Transport Port: 50883
    Flow 4
        Octets: 925
        Packets: 6
        Protocol: UDP (17)
        SrcAddr: 172.217.197.189
        SrcPort: 443 (443)
        InputInt: 2
        DstPort: 59219 (59219)
        DstAddr: 10.0.0.237
        OutputInt: 1
        NextHop: 10.0.0.1
        [Duration: 0.000000000 seconds (switched)]
            StartTime: 893.000000000 seconds
            EndTime: 893.000000000 seconds
        Post NAT Source IPv4 Address: 172.217.197.189
        Post NAT Destination IPv4 Address: 192.168.168.65
        Post NAPT Source Transport Port: 443
        Post NAPT Destination Transport Port: 55658
    Flow 5
        Octets: 221
        Packets: 1
        Protocol: UDP (17)
        SrcAddr: 75.75.76.76
        SrcPort: 53 (53)
        InputInt: 2
        DstPort: 55221 (55221)
        DstAddr: 10.0.0.237
        OutputInt: 1
        NextHop: 10.0.0.1
        [Duration: 0.000000000 seconds (switched)]
            StartTime: 895.000000000 seconds
            EndTime: 895.000000000 seconds
        Post NAT Source IPv4 Address: 75.75.76.76
        Post NAT Destination IPv4 Address: 192.168.168.65
        Post NAPT Source Transport Port: 53
        Post NAPT Destination Transport Port: 53236
'''

host = sys.argv[1]
port = 2055
N = 150000
flowsPerPacket = 5

sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
sock.sendto(tpl, (host, port))
time.sleep(0.2)

ts = time.time()
print("%d: started sending %d SonicWALL v9 flows in %d packets totaling %d bytes" % (ts,N*flowsPerPacket, N, N*len(data)))
print("%d: flow size %d, packet size %d" % (ts, len(data) / flowsPerPacket, len(data)))

for i in range(0, N):
    sock.sendto(data, (host, port))
