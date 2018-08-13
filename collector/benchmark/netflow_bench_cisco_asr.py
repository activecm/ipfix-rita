#!/usr/bin/env python2
import socket
import sys
import time


# Netflow v9 template
tpl = '\x00\t\x00\x01e\x9c\xc0_XF\x8eU\x01u\xc7\x03\x00\x00\x08\x81\x00\x00\x00d\x01\x04\x00\x17\x00\x02\x00\x04\x00\x01\x00\x04\x00\x08\x00\x04\x00\x0c\x00\x04\x00\n\x00\x04\x00\x0e\x00\x04\x00\x15\x00\x04\x00\x16\x00\x04\x00\x07\x00\x02\x00\x0b\x00\x02\x00\x10\x00\x04\x00\x11\x00\x04\x00\x12\x00\x04\x00\t\x00\x01\x00\r\x00\x01\x00\x04\x00\x01\x00\x06\x00\x01\x00\x05\x00\x01\x00=\x00\x01\x00Y\x00\x01\x000\x00\x02\x00\xea\x00\x04\x00\xeb\x00\x04'

'''
Cisco NetFlow/IPFIX
    Version: 9
    Count: 1
    SysUptime: 1704771.679000000 seconds
    Timestamp: Dec  6, 2016 03:09:25.000000000 MST
        CurrentSecs: 1481018965
    FlowSequence: 24495875
    SourceId: 2177
    FlowSet 1 [id=0] (Data Template): 260
        FlowSet Id: Data Template (V9) (0)
        FlowSet Length: 100
        Template (Id = 260, Count = 23)
            Template Id: 260
            Field Count: 23
            Field (1/23): PKTS
                Type: PKTS (2)
                Length: 4
            Field (2/23): BYTES
                Type: BYTES (1)
                Length: 4
            Field (3/23): IP_SRC_ADDR
                Type: IP_SRC_ADDR (8)
                Length: 4
            Field (4/23): IP_DST_ADDR
                Type: IP_DST_ADDR (12)
                Length: 4
            Field (5/23): INPUT_SNMP
                Type: INPUT_SNMP (10)
                Length: 4
            Field (6/23): OUTPUT_SNMP
                Type: OUTPUT_SNMP (14)
                Length: 4
            Field (7/23): LAST_SWITCHED
                Type: LAST_SWITCHED (21)
                Length: 4
            Field (8/23): FIRST_SWITCHED
                Type: FIRST_SWITCHED (22)
                Length: 4
            Field (9/23): L4_SRC_PORT
                Type: L4_SRC_PORT (7)
                Length: 2
            Field (10/23): L4_DST_PORT
                Type: L4_DST_PORT (11)
                Length: 2
            Field (11/23): SRC_AS
                Type: SRC_AS (16)
                Length: 4
            Field (12/23): DST_AS
                Type: DST_AS (17)
                Length: 4
            Field (13/23): BGP_NEXT_HOP
                Type: BGP_NEXT_HOP (18)
                Length: 4
            Field (14/23): SRC_MASK
                Type: SRC_MASK (9)
                Length: 1
            Field (15/23): DST_MASK
                Type: DST_MASK (13)
                Length: 1
            Field (16/23): PROTOCOL
                Type: PROTOCOL (4)
                Length: 1
            Field (17/23): TCP_FLAGS
                Type: TCP_FLAGS (6)
                Length: 1
            Field (18/23): IP_TOS
                Type: IP_TOS (5)
                Length: 1
            Field (19/23): DIRECTION
                Type: DIRECTION (61)
                Length: 1
            Field (20/23): FORWARDING_STATUS
                Type: FORWARDING_STATUS (89)
                Length: 1
            Field (21/23): FLOW_SAMPLER_ID
                Type: FLOW_SAMPLER_ID (48)
                Length: 2
            Field (22/23): ingressVRFID
                Type: ingressVRFID (234)
                Length: 4
            Field (23/23): egressVRFID
                Type: egressVRFID (235)
                Length: 4
'''

data = '\x00\t\x00\x15e\x9c\xbcqXF\x8eT\x01u\xc6\xa1\x00\x00\x08\x81\x01\x04\x05\\\x00\x00\x00\x01\x00\x00\x00(\n\x00\t\x92\n\x00\x1fQ\x00\x00\x00n\x00\x00\x00\x9ee\x9cG\x05e\x9cG\x05\xd3\x01\x01\xbb\x00\x00\x00\x00\x00\x00\xfb\xf0\n\x00\x0e!\x10\x14\x06\x10\x00\x01@\x00\x01`\x00\x00\x00`\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00h\n\x00\x11*\n\x00#\x04\x00\x00\x00W\x00\x00\x00\x9ee\x9cI\x88e\x9cG\x07\x8e\x84\x01\xbb\x00\x00\x00\x00\x00\x00\xfb\xf0\n\x00\x0e!\x15\x10\x06\x10\x00\x01@\x00\x01`\x00\x00\x00`\x00\x00\x00\x00\x00\x00\x01\x00\x00\x004\n\x00\x16o\n\x00"\x8d\x00\x00\x00h\x00\x00\x00\x9ee\x9cG\ne\x9cG\nA\xae\x01\xbb\x00\x00\x00\x00\x00\x00\xfb\xf0\n\x00\x0e!\x18\x10\x06\x11\x00\x01@\x00\x01`\x00\x00\x00`\x00\x00\x00\x00\x00\x00\x01\x00\x00\x01\xb3\n\x00\x17;\n\x00$\xaa\x00\x00\x00V\x00\x00\x00\x9ee\x9cG\x0ce\x9cG\x0c\x005\xfd,\x00\x00\x00\x00\x00\x00\xfb\xf1\n\x00\x0e\x1f\x19\x13\x11\x00\x00\x01@\x00\x01`\x00\x00\x00`\x00\x00\x00\x00\x00\x00\x01\x00\x00\x03\xc9\n\x00"G\n\x00\x14\xf2\x00\x00\x00\x9e\x00\x00\x00je\x9cG\re\x9cG\r\x01\xbb\x07\xdd\x00\x00\xfb\xf0\x00\x00\xff\xa2\n\x00\x12\x05\x10\x15\x06\x18\x00\x00@\x00\x01`\x00\x00\x00`\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00h\n\x00\n\x85\n\x00\x1ef\x00\x00\x00n\x00\x00\x00\x9ee\x9cG\re\x9cF\xba\x89\xc9\x00P\x00\x00\x00\x00\x00\x00\xfb\xf0\n\x00\x0e!\x10\x10\x06\x10\x00\x01@\x00\x01`\x00\x00\x00`\x00\x00\x00\x00\x00\x00\x01\x00\x00\x004\n\x00%\x1d\n\x00\x06\x18\x00\x00\x00f\x00\x00\x00\xa2e\x9cG\x10e\x9cG\x10\x00P\xdd\xc3\x00\x00;\x1d\x00\x00\xff\x97\n\x00\x00\xf2\x18\x10\x06\x10 \x00@\x00\x01`\x00\x00\x00`\x00\x00\x00\x00\x00\x00\x01\x00\x00\x02f\n\x00 \xb0\n\x00\x0bq\x00\x00\x00\x9e\x00\x00\x00.e\x9cG\x10e\x9cG\x10\x01\xbb\xdd\xfe\x00\x00\xfb\xf0\x00\x00\xff\x98\n\x00\x12i\x14\x10\x06\x18\x00\x00@\x00\x01`\x00\x00\x00`\x00\x00\x00\x00\x00\x00\x03\x00\x00\x10\xfe\n\x00\x0c\x15\n\x00\x0f&\x00\x00\x00W\x00\x00\x00\x9ee\x9cG\x11e\x9c1\xe7\x01\xbb\x9c\x8e\x00\x00\x80\xa6\x00\x00\xfb\xf2\n\x00\x0e\x1b\x18\x18\x06\x10\x00\x01@\x00\x01`\x00\x00\x00`\x00\x00\x00\x00\x00\x00\x02\x00\x00\x02\x15\n\x00\x04\xd4\n\x00\x03n\x00\x00\x00\xa2\x00\x00\x00fe\x9cT\x07e\x9cG\x12\xc6\x03\x01\xbb\x00\x00\xff\x97\x00\x00\x00F\n\x00\x10e\x10\x11\x06\x18\x00\x01@\x00\x01`\x00\x00\x00`\x00\x00\x00\x00\x00\x01E\x00\x005\\\n\x00!z\n\x00\x01\x88\x00\x00\x00\x9e\x00\x00\x00he\x9co\xd0e\x9c"\x1a\xe5\xbe\x00P\x00\x00\xfb\xf1\x00\x00\x00\x00\x00\x00\x00\x00\x15\x1b\x06\x10\x00\x00@\x00\x01`\x00\x00\x00`\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00Y\n\x00\x14\xf2\n\x00"G\x00\x00\x00j\x00\x00\x00\x9ee\x9cG\x14e\x9cG\x14\x07\xdd\x01\xbb\x00\x00\xff\xa2\x00\x00\xfb\xf0\n\x00\x0e!\x15\x10\x06\x18`\x01@\x00\x01`\x00\x00\x00`\x00\x00\x00\x00\x00\x00\x01\x00\x00\x03A\n\x00\r\x19\n\x00\x0f&\x00\x00\x00W\x00\x00\x00\x9ee\x9cG\x16e\x9cG\x16\x01\xbb\xc9\xa5\x00\x00\x80\xa6\x00\x00\xfb\xf2\n\x00\x0e\x1b\x18\x18\x06\x18\x00\x01@\x00\x01`\x00\x00\x00`\x00\x00\x00\x00\x00\x00\x02\x00\x00\x06Y\n\x00\x19;\n\x00\x02\x12\x00\x00\x00\x9e\x00\x00\x00ne\x9cG\x18e\x9cF\xbf\x01\xbb\xf4\x00\x00\x00\xfb\xf0\x00\x00\xff\x9d\n\x00\x12~\x10\x10\x06\x18\x00\x00@\x00\x01`\x00\x00\x00`\x00\x00\x00\x00\x00\x00a\x00\x02+h\n\x00\x07I\n\x00\x1b\xa8\x00\x00\x00V\x00\x00\x00\x9ee\x9cu\xabe\x9c1\xfe\xeb\x98\x01\xd1\x00\x00\xff\x9c\x00\x00\xfb\xf0\n\x00\x0e!\x10\x10\x06\x18\x00\x01@\x00\x01`\x00\x00\x00`\x00\x00\x00\x00\x00\x00:\x00\x00\x0b\xc8\n\x00\x132\n\x00\x1b\xa9\x00\x00\x00j\x00\x00\x00\x9ee\x9cO\xcbe\x9cE:\x86\x94\x03\xe3\x00\x00\xff\xb7\x00\x00\xfb\xf0\n\x00\x0e!\x12\x10\x06\x10\x00\x01@\x00\x01`\x00\x00\x00`\x00\x00\x00\x00\x00\x00\x15\x00\x00{\x0c\n\x00\x1c\x96\n\x00\x18\r\x00\x00\x00\x9e\x00\x00\x00he\x9cHYe\x9cF\xf0\x01\xbb\xc2\xfd\x00\x00\xfb\xf0\x00\x00\x00\x00\x00\x00\x00\x00\x10\x19\x06\x10\x00\x00@\x00\x01`\x00\x00\x00`\x00\x00\x00\x00\x00\x00\x03\x00\x00\x0bg\n\x00\x1a\xbc\n\x00\x15\xc8\x00\x00\x00\x9e\x00\x00\x00We\x9cGfe\x9cE\xec\x03\xe1\xc4N\x00\x00\xfb\xf0\x00\x00\x00\x00\x00\x00\x00\x00\x10\x19\x06\x18\x00\x00@\x00\x01`\x00\x00\x00`\x00\x00\x00\x00\x00\x00\x05\x00\x00\x11\xa2\n\x00\x1d"\n\x00\x0f&\x00\x00\x00K\x00\x00\x00\x9ee\x9cm`e\x9cA\xfe\x01\xbb\x8c\x8f\x00\x00;A\x00\x00\xfb\xf2\n\x00\x0e\x1b\x18\x18\x06\x18\x00\x01@\x00\x01`\x00\x00\x00`\x00\x00\x00\x00\x00\x00\x01\x00\x00\x01F\n\x00\x08\xc8\n\x00\x05\xe0\x00\x00\x00f\x00\x00\x00\xa2e\x9cG\x1de\x9cG\x1dZX\xc9\xd7\x00\x00\x03\x15\x00\x00\xff\x97\n\x00\x00\xf2\x10\x10\x06\x18\x00\x00@\x00\x01`\x00\x00\x00`\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00p\n\x00\x1d.\n\x00\x0f&\x00\x00\x00K\x00\x00\x00\x9ee\x9cG\x1de\x9c@\xea\x01\xbb\xcc\x8c\x00\x00;A\x00\x00\xfb\xf2\n\x00\x0e\x1b\x18\x18\x06\x12\x00\x01@\x00\x01`\x00\x00\x00`\x00\x00\x00\x00\x00\x00'

'''
Cisco NetFlow/IPFIX
    Version: 9
    Count: 21
    SysUptime: 1704770.673000000 seconds
    Timestamp: Dec  6, 2016 03:09:24.000000000 MST
        CurrentSecs: 1481018964
    FlowSequence: 24495777 (expected 24495876)
        [Expert Info (Warning/Sequence): Unexpected flow sequence for domain ID 2177 (expected 24495876, got 24495777)]
            [Unexpected flow sequence for domain ID 2177 (expected 24495876, got 24495777)]
            [Severity level: Warning]
            [Group: Sequence]
    SourceId: 2177
    FlowSet 1 [id=260] (21 flows)
        FlowSet Id: (Data) (260)
        FlowSet Length: 1372
        [Template Frame: 1]
        Flow 1
            Packets: 1
            Octets: 40
            SrcAddr: 10.0.9.146
            DstAddr: 10.0.31.81
            InputInt: 110
            OutputInt: 158
            [Duration: 0.000000000 seconds (switched)]
                StartTime: 1704740.613000000 seconds
                EndTime: 1704740.613000000 seconds
            SrcPort: 54017
            DstPort: 443
            SrcAS: 0
            DstAS: 64496
            BGPNextHop: 10.0.14.33
            SrcMask: 16
            DstMask: 20
            Protocol: TCP (6)
            TCP Flags: 0x10, ACK
            IP ToS: 0x00
            Direction: Egress (1)
            Forwarding Status
            SamplerID: 1
            Ingress VRFID: 1610612736
            Egress VRFID: 1610612736
        Flow 2
            Packets: 2
            Octets: 104
            SrcAddr: 10.0.17.42
            DstAddr: 10.0.35.4
            InputInt: 87
            OutputInt: 158
            [Duration: 0.641000000 seconds (switched)]
                StartTime: 1704740.615000000 seconds
                EndTime: 1704741.256000000 seconds
            SrcPort: 36484
            DstPort: 443
            SrcAS: 0
            DstAS: 64496
            BGPNextHop: 10.0.14.33
            SrcMask: 21
            DstMask: 16
            Protocol: TCP (6)
            TCP Flags: 0x10, ACK
            IP ToS: 0x00
            Direction: Egress (1)
            Forwarding Status
            SamplerID: 1
            Ingress VRFID: 1610612736
            Egress VRFID: 1610612736
        Flow 3
            Packets: 1
            Octets: 52
            SrcAddr: 10.0.22.111
            DstAddr: 10.0.34.141
            InputInt: 104
            OutputInt: 158
            [Duration: 0.000000000 seconds (switched)]
                StartTime: 1704740.618000000 seconds
                EndTime: 1704740.618000000 seconds
            SrcPort: 16814
            DstPort: 443
            SrcAS: 0
            DstAS: 64496
            BGPNextHop: 10.0.14.33
            SrcMask: 24
            DstMask: 16
            Protocol: TCP (6)
            TCP Flags: 0x11, ACK, FIN
            IP ToS: 0x00
            Direction: Egress (1)
            Forwarding Status
            SamplerID: 1
            Ingress VRFID: 1610612736
            Egress VRFID: 1610612736
        Flow 4
            Packets: 1
            Octets: 435
            SrcAddr: 10.0.23.59
            DstAddr: 10.0.36.170
            InputInt: 86
            OutputInt: 158
            [Duration: 0.000000000 seconds (switched)]
                StartTime: 1704740.620000000 seconds
                EndTime: 1704740.620000000 seconds
            SrcPort: 53
            DstPort: 64812
            SrcAS: 0
            DstAS: 64497
            BGPNextHop: 10.0.14.31
            SrcMask: 25
            DstMask: 19
            Protocol: UDP (17)
            TCP Flags: 0x00
            IP ToS: 0x00
            Direction: Egress (1)
            Forwarding Status
            SamplerID: 1
            Ingress VRFID: 1610612736
            Egress VRFID: 1610612736
        Flow 5
            Packets: 1
            Octets: 969
            SrcAddr: 10.0.34.71
            DstAddr: 10.0.20.242
            InputInt: 158
            OutputInt: 106
            [Duration: 0.000000000 seconds (switched)]
                StartTime: 1704740.621000000 seconds
                EndTime: 1704740.621000000 seconds
            SrcPort: 443
            DstPort: 2013
            SrcAS: 64496
            DstAS: 65442
            BGPNextHop: 10.0.18.5
            SrcMask: 16
            DstMask: 21
            Protocol: TCP (6)
            TCP Flags: 0x18, ACK, PSH
            IP ToS: 0x00
            Direction: Ingress (0)
            Forwarding Status
            SamplerID: 1
            Ingress VRFID: 1610612736
            Egress VRFID: 1610612736
        Flow 6
            Packets: 2
            Octets: 104
            SrcAddr: 10.0.10.133
            DstAddr: 10.0.30.102
            InputInt: 110
            OutputInt: 158
            [Duration: 0.083000000 seconds (switched)]
                StartTime: 1704740.538000000 seconds
                EndTime: 1704740.621000000 seconds
            SrcPort: 35273
            DstPort: 80
            SrcAS: 0
            DstAS: 64496
            BGPNextHop: 10.0.14.33
            SrcMask: 16
            DstMask: 16
            Protocol: TCP (6)
            TCP Flags: 0x10, ACK
            IP ToS: 0x00
            Direction: Egress (1)
            Forwarding Status
            SamplerID: 1
            Ingress VRFID: 1610612736
            Egress VRFID: 1610612736
        Flow 7
            Packets: 1
            Octets: 52
            SrcAddr: 10.0.37.29
            DstAddr: 10.0.6.24
            InputInt: 102
            OutputInt: 162
            [Duration: 0.000000000 seconds (switched)]
                StartTime: 1704740.624000000 seconds
                EndTime: 1704740.624000000 seconds
            SrcPort: 80
            DstPort: 56771
            SrcAS: 15133
            DstAS: 65431
            BGPNextHop: 10.0.0.242
            SrcMask: 24
            DstMask: 16
            Protocol: TCP (6)
            TCP Flags: 0x10, ACK
            IP ToS: 0x20
            Direction: Ingress (0)
            Forwarding Status
            SamplerID: 1
            Ingress VRFID: 1610612736
            Egress VRFID: 1610612736
        Flow 8
            Packets: 1
            Octets: 614
            SrcAddr: 10.0.32.176
            DstAddr: 10.0.11.113
            InputInt: 158
            OutputInt: 46
            [Duration: 0.000000000 seconds (switched)]
                StartTime: 1704740.624000000 seconds
                EndTime: 1704740.624000000 seconds
            SrcPort: 443
            DstPort: 56830
            SrcAS: 64496
            DstAS: 65432
            BGPNextHop: 10.0.18.105
            SrcMask: 20
            DstMask: 16
            Protocol: TCP (6)
            TCP Flags: 0x18, ACK, PSH
            IP ToS: 0x00
            Direction: Ingress (0)
            Forwarding Status
            SamplerID: 1
            Ingress VRFID: 1610612736
            Egress VRFID: 1610612736
        Flow 9
            Packets: 3
            Octets: 4350
            SrcAddr: 10.0.12.21
            DstAddr: 10.0.15.38
            InputInt: 87
            OutputInt: 158
            [Duration: 5.418000000 seconds (switched)]
                StartTime: 1704735.207000000 seconds
                EndTime: 1704740.625000000 seconds
            SrcPort: 443
            DstPort: 40078
            SrcAS: 32934
            DstAS: 64498
            BGPNextHop: 10.0.14.27
            SrcMask: 24
            DstMask: 24
            Protocol: TCP (6)
            TCP Flags: 0x10, ACK
            IP ToS: 0x00
            Direction: Egress (1)
            Forwarding Status
            SamplerID: 1
            Ingress VRFID: 1610612736
            Egress VRFID: 1610612736
        Flow 10
            Packets: 2
            Octets: 533
            SrcAddr: 10.0.4.212
            DstAddr: 10.0.3.110
            InputInt: 162
            OutputInt: 102
            [Duration: 3.317000000 seconds (switched)]
                StartTime: 1704740.626000000 seconds
                EndTime: 1704743.943000000 seconds
            SrcPort: 50691
            DstPort: 443
            SrcAS: 65431
            DstAS: 70
            BGPNextHop: 10.0.16.101
            SrcMask: 16
            DstMask: 17
            Protocol: TCP (6)
            TCP Flags: 0x18, ACK, PSH
            IP ToS: 0x00
            Direction: Egress (1)
            Forwarding Status
            SamplerID: 1
            Ingress VRFID: 1610612736
            Egress VRFID: 1610612736
        Flow 11
            Packets: 325
            Octets: 13660
            SrcAddr: 10.0.33.122
            DstAddr: 10.0.1.136
            InputInt: 158
            OutputInt: 104
            [Duration: 19.894000000 seconds (switched)]
                StartTime: 1704731.162000000 seconds
                EndTime: 1704751.056000000 seconds
            SrcPort: 58814
            DstPort: 80
            SrcAS: 64497
            DstAS: 0
            BGPNextHop: 0.0.0.0
            SrcMask: 21
            DstMask: 27
            Protocol: TCP (6)
            TCP Flags: 0x10, ACK
            IP ToS: 0x00
            Direction: Ingress (0)
            Forwarding Status
            SamplerID: 1
            Ingress VRFID: 1610612736
            Egress VRFID: 1610612736
        Flow 12
            Packets: 1
            Octets: 89
            SrcAddr: 10.0.20.242
            DstAddr: 10.0.34.71
            InputInt: 106
            OutputInt: 158
            [Duration: 0.000000000 seconds (switched)]
                StartTime: 1704740.628000000 seconds
                EndTime: 1704740.628000000 seconds
            SrcPort: 2013
            DstPort: 443
            SrcAS: 65442
            DstAS: 64496
            BGPNextHop: 10.0.14.33
            SrcMask: 21
            DstMask: 16
            Protocol: TCP (6)
            TCP Flags: 0x18, ACK, PSH
            IP ToS: 0x60
            Direction: Egress (1)
            Forwarding Status
            SamplerID: 1
            Ingress VRFID: 1610612736
            Egress VRFID: 1610612736
        Flow 13
            Packets: 1
            Octets: 833
            SrcAddr: 10.0.13.25
            DstAddr: 10.0.15.38
            InputInt: 87
            OutputInt: 158
            [Duration: 0.000000000 seconds (switched)]
                StartTime: 1704740.630000000 seconds
                EndTime: 1704740.630000000 seconds
            SrcPort: 443
            DstPort: 51621
            SrcAS: 32934
            DstAS: 64498
            BGPNextHop: 10.0.14.27
            SrcMask: 24
            DstMask: 24
            Protocol: TCP (6)
            TCP Flags: 0x18, ACK, PSH
            IP ToS: 0x00
            Direction: Egress (1)
            Forwarding Status
            SamplerID: 1
            Ingress VRFID: 1610612736
            Egress VRFID: 1610612736
        Flow 14
            Packets: 2
            Octets: 1625
            SrcAddr: 10.0.25.59
            DstAddr: 10.0.2.18
            InputInt: 158
            OutputInt: 110
            [Duration: 0.089000000 seconds (switched)]
                StartTime: 1704740.543000000 seconds
                EndTime: 1704740.632000000 seconds
            SrcPort: 443
            DstPort: 62464
            SrcAS: 64496
            DstAS: 65437
            BGPNextHop: 10.0.18.126
            SrcMask: 16
            DstMask: 16
            Protocol: TCP (6)
            TCP Flags: 0x18, ACK, PSH
            IP ToS: 0x00
            Direction: Ingress (0)
            Forwarding Status
            SamplerID: 1
            Ingress VRFID: 1610612736
            Egress VRFID: 1610612736
        Flow 15
            Packets: 97
            Octets: 142184
            SrcAddr: 10.0.7.73
            DstAddr: 10.0.27.168
            InputInt: 86
            OutputInt: 158
            [Duration: 17.325000000 seconds (switched)]
                StartTime: 1704735.230000000 seconds
                EndTime: 1704752.555000000 seconds
            SrcPort: 60312
            DstPort: 465
            SrcAS: 65436
            DstAS: 64496
            BGPNextHop: 10.0.14.33
            SrcMask: 16
            DstMask: 16
            Protocol: TCP (6)
            TCP Flags: 0x18, ACK, PSH
            IP ToS: 0x00
            Direction: Egress (1)
            Forwarding Status
            SamplerID: 1
            Ingress VRFID: 1610612736
            Egress VRFID: 1610612736
        Flow 16
            Packets: 58
            Octets: 3016
            SrcAddr: 10.0.19.50
            DstAddr: 10.0.27.169
            InputInt: 106
            OutputInt: 158
            [Duration: 2.705000000 seconds (switched)]
                StartTime: 1704740.154000000 seconds
                EndTime: 1704742.859000000 seconds
            SrcPort: 34452
            DstPort: 995
            SrcAS: 65463
            DstAS: 64496
            BGPNextHop: 10.0.14.33
            SrcMask: 18
            DstMask: 16
            Protocol: TCP (6)
            TCP Flags: 0x10, ACK
            IP ToS: 0x00
            Direction: Egress (1)
            Forwarding Status
            SamplerID: 1
            Ingress VRFID: 1610612736
            Egress VRFID: 1610612736
        Flow 17
            Packets: 21
            Octets: 31500
            SrcAddr: 10.0.28.150
            DstAddr: 10.0.24.13
            InputInt: 158
            OutputInt: 104
            [Duration: 0.361000000 seconds (switched)]
                StartTime: 1704740.592000000 seconds
                EndTime: 1704740.953000000 seconds
            SrcPort: 443
            DstPort: 49917
            SrcAS: 64496
            DstAS: 0
            BGPNextHop: 0.0.0.0
            SrcMask: 16
            DstMask: 25
            Protocol: TCP (6)
            TCP Flags: 0x10, ACK
            IP ToS: 0x00
            Direction: Ingress (0)
            Forwarding Status
            SamplerID: 1
            Ingress VRFID: 1610612736
            Egress VRFID: 1610612736
        Flow 18
            Packets: 3
            Octets: 2919
            SrcAddr: 10.0.26.188
            DstAddr: 10.0.21.200
            InputInt: 158
            OutputInt: 87
            [Duration: 0.378000000 seconds (switched)]
                StartTime: 1704740.332000000 seconds
                EndTime: 1704740.710000000 seconds
            SrcPort: 993
            DstPort: 50254
            SrcAS: 64496
            DstAS: 0
            BGPNextHop: 0.0.0.0
            SrcMask: 16
            DstMask: 25
            Protocol: TCP (6)
            TCP Flags: 0x18, ACK, PSH
            IP ToS: 0x00
            Direction: Ingress (0)
            Forwarding Status
            SamplerID: 1
            Ingress VRFID: 1610612736
            Egress VRFID: 1610612736
        Flow 19
            Packets: 5
            Octets: 4514
            SrcAddr: 10.0.29.34
            DstAddr: 10.0.15.38
            InputInt: 75
            OutputInt: 158
            [Duration: 11.106000000 seconds (switched)]
                StartTime: 1704739.326000000 seconds
                EndTime: 1704750.432000000 seconds
            SrcPort: 443
            DstPort: 35983
            SrcAS: 15169
            DstAS: 64498
            BGPNextHop: 10.0.14.27
            SrcMask: 24
            DstMask: 24
            Protocol: TCP (6)
            TCP Flags: 0x18, ACK, PSH
            IP ToS: 0x00
            Direction: Egress (1)
            Forwarding Status
            SamplerID: 1
            Ingress VRFID: 1610612736
            Egress VRFID: 1610612736
        Flow 20
            Packets: 1
            Octets: 326
            SrcAddr: 10.0.8.200
            DstAddr: 10.0.5.224
            InputInt: 102
            OutputInt: 162
            [Duration: 0.000000000 seconds (switched)]
                StartTime: 1704740.637000000 seconds
                EndTime: 1704740.637000000 seconds
            SrcPort: 23128
            DstPort: 51671
            SrcAS: 789
            DstAS: 65431
            BGPNextHop: 10.0.0.242
            SrcMask: 16
            DstMask: 16
            Protocol: TCP (6)
            TCP Flags: 0x18, ACK, PSH
            IP ToS: 0x00
            Direction: Ingress (0)
            Forwarding Status
            SamplerID: 1
            Ingress VRFID: 1610612736
            Egress VRFID: 1610612736
        Flow 21
            Packets: 2
            Octets: 112
            SrcAddr: 10.0.29.46
            DstAddr: 10.0.15.38
            InputInt: 75
            OutputInt: 158
            [Duration: 1.587000000 seconds (switched)]
                StartTime: 1704739.050000000 seconds
                EndTime: 1704740.637000000 seconds
            SrcPort: 443
            DstPort: 52364
            SrcAS: 15169
            DstAS: 64498
            BGPNextHop: 10.0.14.27
            SrcMask: 24
            DstMask: 24
            Protocol: TCP (6)
            TCP Flags: 0x12, ACK, SYN
            IP ToS: 0x00
            Direction: Egress (1)
            Forwarding Status
            SamplerID: 1
            Ingress VRFID: 1610612736
            Egress VRFID: 1610612736
        Padding: 000000
    [Expected Sequence Number: 24495876]
    [Previous Frame in Sequence: 1]
'''

host = sys.argv[1]
port = 2055
N = 150000
flowsPerPacket = 21

sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
sock.sendto(tpl, (host, port))
time.sleep(0.2)

ts = time.time()
print("%d: started sending %d Cisco ASR 9000 flows in %d packets totaling %d bytes" % (ts,N*flowsPerPacket, N, N*len(data)))
print("%d: flow size %d, packet size %d" % (ts, len(data) / flowsPerPacket, len(data)))

for i in range(0, N):
    sock.sendto(data, (host, port))
