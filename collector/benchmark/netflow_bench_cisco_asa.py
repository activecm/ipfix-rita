#!/usr/bin/env python2
import socket
import sys
import time


# Netflow v9 template
tpl = '\x00\t\x00\r\x00\x1fz\xc4V\x17\x8dE\x00\x00\x02\x95\x00\x00\x00\x00\x00\x00\x03\xe0\x01\x00\x00\x15\x00\x94\x00\x04\x00\x08\x00\x04\x00\x07\x00\x02\x00\n\x00\x02\x00\x0c\x00\x04\x00\x0b\x00\x02\x00\x0e\x00\x02\x00\x04\x00\x01\x00\xb0\x00\x01\x00\xb1\x00\x01\x9cA\x00\x04\x9cB\x00\x04\x9cC\x00\x02\x9cD\x00\x02\x9cE\x00\x01\x80\xea\x00\x02\x01C\x00\x08\x00U\x00\x04\x80\xe8\x00\x0c\x80\xe9\x00\x0c\x9c@\x00\x14\x01\x01\x00\x15\x00\x94\x00\x04\x00\x08\x00\x04\x00\x07\x00\x02\x00\n\x00\x02\x00\x0c\x00\x04\x00\x0b\x00\x02\x00\x0e\x00\x02\x00\x04\x00\x01\x00\xb0\x00\x01\x00\xb1\x00\x01\x9cA\x00\x04\x9cB\x00\x04\x9cC\x00\x02\x9cD\x00\x02\x9cE\x00\x01\x80\xea\x00\x02\x01C\x00\x08\x00U\x00\x04\x80\xe8\x00\x0c\x80\xe9\x00\x0c\x9c@\x00A\x01\x02\x00\x11\x00\x94\x00\x04\x00\x1b\x00\x10\x00\x07\x00\x02\x00\n\x00\x02\x00\x1c\x00\x10\x00\x0b\x00\x02\x00\x0e\x00\x02\x00\x04\x00\x01\x00\xb2\x00\x01\x00\xb3\x00\x01\x9cE\x00\x01\x80\xea\x00\x02\x01C\x00\x08\x00U\x00\x04\x80\xe8\x00\x0c\x80\xe9\x00\x0c\x9c@\x00\x14\x01\x03\x00\x11\x00\x94\x00\x04\x00\x1b\x00\x10\x00\x07\x00\x02\x00\n\x00\x02\x00\x1c\x00\x10\x00\x0b\x00\x02\x00\x0e\x00\x02\x00\x04\x00\x01\x00\xb2\x00\x01\x00\xb3\x00\x01\x9cE\x00\x01\x80\xea\x00\x02\x01C\x00\x08\x00U\x00\x04\x80\xe8\x00\x0c\x80\xe9\x00\x0c\x9c@\x00A\x01\x04\x00\x12\x00\x08\x00\x04\x00\x07\x00\x02\x00\n\x00\x02\x00\x0c\x00\x04\x00\x0b\x00\x02\x00\x0e\x00\x02\x00\x04\x00\x01\x00\xb0\x00\x01\x00\xb1\x00\x01\x9cA\x00\x04\x9cB\x00\x04\x9cC\x00\x02\x9cD\x00\x02\x9cE\x00\x01\x80\xea\x00\x02\x01C\x00\x08\x80\xe8\x00\x0c\x80\xe9\x00\x0c\x01\x05\x00\x0e\x00\x08\x00\x04\x00\x07\x00\x02\x00\n\x00\x02\x00\x0c\x00\x04\x00\x0b\x00\x02\x00\x0e\x00\x02\x00\x04\x00\x01\x00\xb0\x00\x01\x00\xb1\x00\x01\x9cE\x00\x01\x80\xea\x00\x02\x01C\x00\x08\x80\xe8\x00\x0c\x80\xe9\x00\x0c\x01\x06\x00\x0e\x00\x1b\x00\x10\x00\x07\x00\x02\x00\n\x00\x02\x00\x1c\x00\x10\x00\x0b\x00\x02\x00\x0e\x00\x02\x00\x04\x00\x01\x00\xb2\x00\x01\x00\xb3\x00\x01\x9cE\x00\x01\x80\xea\x00\x02\x01C\x00\x08\x80\xe8\x00\x0c\x80\xe9\x00\x0c\x01\x07\x00\x12\x00\x94\x00\x04\x00\x08\x00\x04\x00\x07\x00\x02\x00\n\x00\x02\x00\x0c\x00\x04\x00\x0b\x00\x02\x00\x0e\x00\x02\x00\x04\x00\x01\x00\xb0\x00\x01\x00\xb1\x00\x01\x9cA\x00\x04\x9cB\x00\x04\x9cC\x00\x02\x9cD\x00\x02\x9cE\x00\x01\x80\xea\x00\x02\x01C\x00\x08\x00U\x00\x04\x01\x08\x00\x0e\x00\x94\x00\x04\x00\x1b\x00\x10\x00\x07\x00\x02\x00\n\x00\x02\x00\x1c\x00\x10\x00\x0b\x00\x02\x00\x0e\x00\x02\x00\x04\x00\x01\x00\xb2\x00\x01\x00\xb3\x00\x01\x9cE\x00\x01\x80\xea\x00\x02\x01C\x00\x08\x00U\x00\x04\x01\t\x00\x16\x00\x94\x00\x04\x00\x08\x00\x04\x00\x07\x00\x02\x00\n\x00\x02\x00\x0c\x00\x04\x00\x0b\x00\x02\x00\x0e\x00\x02\x00\x04\x00\x01\x00\xb0\x00\x01\x00\xb1\x00\x01\x9cA\x00\x04\x9cB\x00\x04\x9cC\x00\x02\x9cD\x00\x02\x9cE\x00\x01\x80\xea\x00\x02\x01C\x00\x08\x00U\x00\x04\x00\x98\x00\x08\x80\xe8\x00\x0c\x80\xe9\x00\x0c\x9c@\x00\x14\x01\n\x00\x16\x00\x94\x00\x04\x00\x08\x00\x04\x00\x07\x00\x02\x00\n\x00\x02\x00\x0c\x00\x04\x00\x0b\x00\x02\x00\x0e\x00\x02\x00\x04\x00\x01\x00\xb0\x00\x01\x00\xb1\x00\x01\x9cA\x00\x04\x9cB\x00\x04\x9cC\x00\x02\x9cD\x00\x02\x9cE\x00\x01\x80\xea\x00\x02\x01C\x00\x08\x00U\x00\x04\x00\x98\x00\x08\x80\xe8\x00\x0c\x80\xe9\x00\x0c\x9c@\x00A\x01\x0b\x00\x12\x00\x94\x00\x04\x00\x1b\x00\x10\x00\x07\x00\x02\x00\n\x00\x02\x00\x1c\x00\x10\x00\x0b\x00\x02\x00\x0e\x00\x02\x00\x04\x00\x01\x00\xb2\x00\x01\x00\xb3\x00\x01\x9cE\x00\x01\x80\xea\x00\x02\x01C\x00\x08\x00U\x00\x04\x00\x98\x00\x08\x80\xe8\x00\x0c\x80\xe9\x00\x0c\x9c@\x00\x14\x01\x0c\x00\x12\x00\x94\x00\x04\x00\x1b\x00\x10\x00\x07\x00\x02\x00\n\x00\x02\x00\x1c\x00\x10\x00\x0b\x00\x02\x00\x0e\x00\x02\x00\x04\x00\x01\x00\xb2\x00\x01\x00\xb3\x00\x01\x9cE\x00\x01\x80\xea\x00\x02\x01C\x00\x08\x00U\x00\x04\x00\x98\x00\x08\x80\xe8\x00\x0c\x80\xe9\x00\x0c\x9c@\x00A'

'''
Cisco NetFlow/IPFIX
    Version: 9
    Count: 13
    SysUptime: 2063.044000000 seconds
    Timestamp: Oct  9, 2015 03:47:49.000000000 MDT
        CurrentSecs: 1444384069
    FlowSequence: 661
    SourceId: 0
    FlowSet 1 [id=0] (Data Template): 256,257,258,259,260,261,262,263,264,265,266,267,268
        FlowSet Id: Data Template (V9) (0)
        FlowSet Length: 992
        Template (Id = 256, Count = 21)
            Template Id: 256
            Field Count: 21
            Field (1/21): flowId
                Type: flowId (148)
                Length: 4
            Field (2/21): IP_SRC_ADDR
                Type: IP_SRC_ADDR (8)
                Length: 4
            Field (3/21): L4_SRC_PORT
                Type: L4_SRC_PORT (7)
                Length: 2
            Field (4/21): INPUT_SNMP
                Type: INPUT_SNMP (10)
                Length: 2
            Field (5/21): IP_DST_ADDR
                Type: IP_DST_ADDR (12)
                Length: 4
            Field (6/21): L4_DST_PORT
                Type: L4_DST_PORT (11)
                Length: 2
            Field (7/21): OUTPUT_SNMP
                Type: OUTPUT_SNMP (14)
                Length: 2
            Field (8/21): PROTOCOL
                Type: PROTOCOL (4)
                Length: 1
            Field (9/21): ICMP_IPv4_TYPE
                Type: ICMP_IPv4_TYPE (176)
                Length: 1
            Field (10/21): ICMP_IPv4_CODE
                Type: ICMP_IPv4_CODE (177)
                Length: 1
            Field (11/21): XLATE_SRC_ADDR_IPV4
                Type: XLATE_SRC_ADDR_IPV4 (40001)
                Length: 4
            Field (12/21): XLATE_DST_ADDR_IPV4
                Type: XLATE_DST_ADDR_IPV4 (40002)
                Length: 4
            Field (13/21): XLATE_SRC_PORT
                Type: XLATE_SRC_PORT (40003)
                Length: 2
            Field (14/21): XLATE_DST_PORT
                Type: XLATE_DST_PORT (40004)
                Length: 2
            Field (15/21): FW_EVENT
                Type: FW_EVENT (40005)
                Length: 1
            Field (16/21): FW_EXT_EVENT
                Type: FW_EXT_EVENT (33002)
                Length: 2
            Field (17/21): observationTimeMilliseconds
                Type: observationTimeMilliseconds (323)
                Length: 8
            Field (18/21): BYTES_TOTAL
                Type: BYTES_TOTAL (85)
                Length: 4
            Field (19/21): INGRESS_ACL_ID
                Type: INGRESS_ACL_ID (33000)
                Length: 12
            Field (20/21): EGRESS_ACL_ID
                Type: EGRESS_ACL_ID (33001)
                Length: 12
            Field (21/21): AAA_USERNAME
                Type: AAA_USERNAME (40000)
                Length: 20
        Template (Id = 257, Count = 21)
            Template Id: 257
            Field Count: 21
            Field (1/21): flowId
                Type: flowId (148)
                Length: 4
            Field (2/21): IP_SRC_ADDR
                Type: IP_SRC_ADDR (8)
                Length: 4
            Field (3/21): L4_SRC_PORT
                Type: L4_SRC_PORT (7)
                Length: 2
            Field (4/21): INPUT_SNMP
                Type: INPUT_SNMP (10)
                Length: 2
            Field (5/21): IP_DST_ADDR
                Type: IP_DST_ADDR (12)
                Length: 4
            Field (6/21): L4_DST_PORT
                Type: L4_DST_PORT (11)
                Length: 2
            Field (7/21): OUTPUT_SNMP
                Type: OUTPUT_SNMP (14)
                Length: 2
            Field (8/21): PROTOCOL
                Type: PROTOCOL (4)
                Length: 1
            Field (9/21): ICMP_IPv4_TYPE
                Type: ICMP_IPv4_TYPE (176)
                Length: 1
            Field (10/21): ICMP_IPv4_CODE
                Type: ICMP_IPv4_CODE (177)
                Length: 1
            Field (11/21): XLATE_SRC_ADDR_IPV4
                Type: XLATE_SRC_ADDR_IPV4 (40001)
                Length: 4
            Field (12/21): XLATE_DST_ADDR_IPV4
                Type: XLATE_DST_ADDR_IPV4 (40002)
                Length: 4
            Field (13/21): XLATE_SRC_PORT
                Type: XLATE_SRC_PORT (40003)
                Length: 2
            Field (14/21): XLATE_DST_PORT
                Type: XLATE_DST_PORT (40004)
                Length: 2
            Field (15/21): FW_EVENT
                Type: FW_EVENT (40005)
                Length: 1
            Field (16/21): FW_EXT_EVENT
                Type: FW_EXT_EVENT (33002)
                Length: 2
            Field (17/21): observationTimeMilliseconds
                Type: observationTimeMilliseconds (323)
                Length: 8
            Field (18/21): BYTES_TOTAL
                Type: BYTES_TOTAL (85)
                Length: 4
            Field (19/21): INGRESS_ACL_ID
                Type: INGRESS_ACL_ID (33000)
                Length: 12
            Field (20/21): EGRESS_ACL_ID
                Type: EGRESS_ACL_ID (33001)
                Length: 12
            Field (21/21): AAA_USERNAME
                Type: AAA_USERNAME (40000)
                Length: 65
        Template (Id = 258, Count = 17)
            Template Id: 258
            Field Count: 17
            Field (1/17): flowId
                Type: flowId (148)
                Length: 4
            Field (2/17): IPV6_SRC_ADDR
                Type: IPV6_SRC_ADDR (27)
                Length: 16
            Field (3/17): L4_SRC_PORT
                Type: L4_SRC_PORT (7)
                Length: 2
            Field (4/17): INPUT_SNMP
                Type: INPUT_SNMP (10)
                Length: 2
            Field (5/17): IPV6_DST_ADDR
                Type: IPV6_DST_ADDR (28)
                Length: 16
            Field (6/17): L4_DST_PORT
                Type: L4_DST_PORT (11)
                Length: 2
            Field (7/17): OUTPUT_SNMP
                Type: OUTPUT_SNMP (14)
                Length: 2
            Field (8/17): PROTOCOL
                Type: PROTOCOL (4)
                Length: 1
            Field (9/17): ICMP_IPv6_TYPE
                Type: ICMP_IPv6_TYPE (178)
                Length: 1
            Field (10/17): ICMP_IPv6_CODE
                Type: ICMP_IPv6_CODE (179)
                Length: 1
            Field (11/17): FW_EVENT
                Type: FW_EVENT (40005)
                Length: 1
            Field (12/17): FW_EXT_EVENT
                Type: FW_EXT_EVENT (33002)
                Length: 2
            Field (13/17): observationTimeMilliseconds
                Type: observationTimeMilliseconds (323)
                Length: 8
            Field (14/17): BYTES_TOTAL
                Type: BYTES_TOTAL (85)
                Length: 4
            Field (15/17): INGRESS_ACL_ID
                Type: INGRESS_ACL_ID (33000)
                Length: 12
            Field (16/17): EGRESS_ACL_ID
                Type: EGRESS_ACL_ID (33001)
                Length: 12
            Field (17/17): AAA_USERNAME
                Type: AAA_USERNAME (40000)
                Length: 20
        Template (Id = 259, Count = 17)
            Template Id: 259
            Field Count: 17
            Field (1/17): flowId
                Type: flowId (148)
                Length: 4
            Field (2/17): IPV6_SRC_ADDR
                Type: IPV6_SRC_ADDR (27)
                Length: 16
            Field (3/17): L4_SRC_PORT
                Type: L4_SRC_PORT (7)
                Length: 2
            Field (4/17): INPUT_SNMP
                Type: INPUT_SNMP (10)
                Length: 2
            Field (5/17): IPV6_DST_ADDR
                Type: IPV6_DST_ADDR (28)
                Length: 16
            Field (6/17): L4_DST_PORT
                Type: L4_DST_PORT (11)
                Length: 2
            Field (7/17): OUTPUT_SNMP
                Type: OUTPUT_SNMP (14)
                Length: 2
            Field (8/17): PROTOCOL
                Type: PROTOCOL (4)
                Length: 1
            Field (9/17): ICMP_IPv6_TYPE
                Type: ICMP_IPv6_TYPE (178)
                Length: 1
            Field (10/17): ICMP_IPv6_CODE
                Type: ICMP_IPv6_CODE (179)
                Length: 1
            Field (11/17): FW_EVENT
                Type: FW_EVENT (40005)
                Length: 1
            Field (12/17): FW_EXT_EVENT
                Type: FW_EXT_EVENT (33002)
                Length: 2
            Field (13/17): observationTimeMilliseconds
                Type: observationTimeMilliseconds (323)
                Length: 8
            Field (14/17): BYTES_TOTAL
                Type: BYTES_TOTAL (85)
                Length: 4
            Field (15/17): INGRESS_ACL_ID
                Type: INGRESS_ACL_ID (33000)
                Length: 12
            Field (16/17): EGRESS_ACL_ID
                Type: EGRESS_ACL_ID (33001)
                Length: 12
            Field (17/17): AAA_USERNAME
                Type: AAA_USERNAME (40000)
                Length: 65
        Template (Id = 260, Count = 18)
            Template Id: 260
            Field Count: 18
            Field (1/18): IP_SRC_ADDR
                Type: IP_SRC_ADDR (8)
                Length: 4
            Field (2/18): L4_SRC_PORT
                Type: L4_SRC_PORT (7)
                Length: 2
            Field (3/18): INPUT_SNMP
                Type: INPUT_SNMP (10)
                Length: 2
            Field (4/18): IP_DST_ADDR
                Type: IP_DST_ADDR (12)
                Length: 4
            Field (5/18): L4_DST_PORT
                Type: L4_DST_PORT (11)
                Length: 2
            Field (6/18): OUTPUT_SNMP
                Type: OUTPUT_SNMP (14)
                Length: 2
            Field (7/18): PROTOCOL
                Type: PROTOCOL (4)
                Length: 1
            Field (8/18): ICMP_IPv4_TYPE
                Type: ICMP_IPv4_TYPE (176)
                Length: 1
            Field (9/18): ICMP_IPv4_CODE
                Type: ICMP_IPv4_CODE (177)
                Length: 1
            Field (10/18): XLATE_SRC_ADDR_IPV4
                Type: XLATE_SRC_ADDR_IPV4 (40001)
                Length: 4
            Field (11/18): XLATE_DST_ADDR_IPV4
                Type: XLATE_DST_ADDR_IPV4 (40002)
                Length: 4
            Field (12/18): XLATE_SRC_PORT
                Type: XLATE_SRC_PORT (40003)
                Length: 2
            Field (13/18): XLATE_DST_PORT
                Type: XLATE_DST_PORT (40004)
                Length: 2
            Field (14/18): FW_EVENT
                Type: FW_EVENT (40005)
                Length: 1
            Field (15/18): FW_EXT_EVENT
                Type: FW_EXT_EVENT (33002)
                Length: 2
            Field (16/18): observationTimeMilliseconds
                Type: observationTimeMilliseconds (323)
                Length: 8
            Field (17/18): INGRESS_ACL_ID
                Type: INGRESS_ACL_ID (33000)
                Length: 12
            Field (18/18): EGRESS_ACL_ID
                Type: EGRESS_ACL_ID (33001)
                Length: 12
        Template (Id = 261, Count = 14)
            Template Id: 261
            Field Count: 14
            Field (1/14): IP_SRC_ADDR
                Type: IP_SRC_ADDR (8)
                Length: 4
            Field (2/14): L4_SRC_PORT
                Type: L4_SRC_PORT (7)
                Length: 2
            Field (3/14): INPUT_SNMP
                Type: INPUT_SNMP (10)
                Length: 2
            Field (4/14): IP_DST_ADDR
                Type: IP_DST_ADDR (12)
                Length: 4
            Field (5/14): L4_DST_PORT
                Type: L4_DST_PORT (11)
                Length: 2
            Field (6/14): OUTPUT_SNMP
                Type: OUTPUT_SNMP (14)
                Length: 2
            Field (7/14): PROTOCOL
                Type: PROTOCOL (4)
                Length: 1
            Field (8/14): ICMP_IPv4_TYPE
                Type: ICMP_IPv4_TYPE (176)
                Length: 1
            Field (9/14): ICMP_IPv4_CODE
                Type: ICMP_IPv4_CODE (177)
                Length: 1
            Field (10/14): FW_EVENT
                Type: FW_EVENT (40005)
                Length: 1
            Field (11/14): FW_EXT_EVENT
                Type: FW_EXT_EVENT (33002)
                Length: 2
            Field (12/14): observationTimeMilliseconds
                Type: observationTimeMilliseconds (323)
                Length: 8
            Field (13/14): INGRESS_ACL_ID
                Type: INGRESS_ACL_ID (33000)
                Length: 12
            Field (14/14): EGRESS_ACL_ID
                Type: EGRESS_ACL_ID (33001)
                Length: 12
        Template (Id = 262, Count = 14)
            Template Id: 262
            Field Count: 14
            Field (1/14): IPV6_SRC_ADDR
                Type: IPV6_SRC_ADDR (27)
                Length: 16
            Field (2/14): L4_SRC_PORT
                Type: L4_SRC_PORT (7)
                Length: 2
            Field (3/14): INPUT_SNMP
                Type: INPUT_SNMP (10)
                Length: 2
            Field (4/14): IPV6_DST_ADDR
                Type: IPV6_DST_ADDR (28)
                Length: 16
            Field (5/14): L4_DST_PORT
                Type: L4_DST_PORT (11)
                Length: 2
            Field (6/14): OUTPUT_SNMP
                Type: OUTPUT_SNMP (14)
                Length: 2
            Field (7/14): PROTOCOL
                Type: PROTOCOL (4)
                Length: 1
            Field (8/14): ICMP_IPv6_TYPE
                Type: ICMP_IPv6_TYPE (178)
                Length: 1
            Field (9/14): ICMP_IPv6_CODE
                Type: ICMP_IPv6_CODE (179)
                Length: 1
            Field (10/14): FW_EVENT
                Type: FW_EVENT (40005)
                Length: 1
            Field (11/14): FW_EXT_EVENT
                Type: FW_EXT_EVENT (33002)
                Length: 2
            Field (12/14): observationTimeMilliseconds
                Type: observationTimeMilliseconds (323)
                Length: 8
            Field (13/14): INGRESS_ACL_ID
                Type: INGRESS_ACL_ID (33000)
                Length: 12
            Field (14/14): EGRESS_ACL_ID
                Type: EGRESS_ACL_ID (33001)
                Length: 12
        Template (Id = 263, Count = 18)
            Template Id: 263
            Field Count: 18
            Field (1/18): flowId
                Type: flowId (148)
                Length: 4
            Field (2/18): IP_SRC_ADDR
                Type: IP_SRC_ADDR (8)
                Length: 4
            Field (3/18): L4_SRC_PORT
                Type: L4_SRC_PORT (7)
                Length: 2
            Field (4/18): INPUT_SNMP
                Type: INPUT_SNMP (10)
                Length: 2
            Field (5/18): IP_DST_ADDR
                Type: IP_DST_ADDR (12)
                Length: 4
            Field (6/18): L4_DST_PORT
                Type: L4_DST_PORT (11)
                Length: 2
            Field (7/18): OUTPUT_SNMP
                Type: OUTPUT_SNMP (14)
                Length: 2
            Field (8/18): PROTOCOL
                Type: PROTOCOL (4)
                Length: 1
            Field (9/18): ICMP_IPv4_TYPE
                Type: ICMP_IPv4_TYPE (176)
                Length: 1
            Field (10/18): ICMP_IPv4_CODE
                Type: ICMP_IPv4_CODE (177)
                Length: 1
            Field (11/18): XLATE_SRC_ADDR_IPV4
                Type: XLATE_SRC_ADDR_IPV4 (40001)
                Length: 4
            Field (12/18): XLATE_DST_ADDR_IPV4
                Type: XLATE_DST_ADDR_IPV4 (40002)
                Length: 4
            Field (13/18): XLATE_SRC_PORT
                Type: XLATE_SRC_PORT (40003)
                Length: 2
            Field (14/18): XLATE_DST_PORT
                Type: XLATE_DST_PORT (40004)
                Length: 2
            Field (15/18): FW_EVENT
                Type: FW_EVENT (40005)
                Length: 1
            Field (16/18): FW_EXT_EVENT
                Type: FW_EXT_EVENT (33002)
                Length: 2
            Field (17/18): observationTimeMilliseconds
                Type: observationTimeMilliseconds (323)
                Length: 8
            Field (18/18): BYTES_TOTAL
                Type: BYTES_TOTAL (85)
                Length: 4
        Template (Id = 264, Count = 14)
            Template Id: 264
            Field Count: 14
            Field (1/14): flowId
                Type: flowId (148)
                Length: 4
            Field (2/14): IPV6_SRC_ADDR
                Type: IPV6_SRC_ADDR (27)
                Length: 16
            Field (3/14): L4_SRC_PORT
                Type: L4_SRC_PORT (7)
                Length: 2
            Field (4/14): INPUT_SNMP
                Type: INPUT_SNMP (10)
                Length: 2
            Field (5/14): IPV6_DST_ADDR
                Type: IPV6_DST_ADDR (28)
                Length: 16
            Field (6/14): L4_DST_PORT
                Type: L4_DST_PORT (11)
                Length: 2
            Field (7/14): OUTPUT_SNMP
                Type: OUTPUT_SNMP (14)
                Length: 2
            Field (8/14): PROTOCOL
                Type: PROTOCOL (4)
                Length: 1
            Field (9/14): ICMP_IPv6_TYPE
                Type: ICMP_IPv6_TYPE (178)
                Length: 1
            Field (10/14): ICMP_IPv6_CODE
                Type: ICMP_IPv6_CODE (179)
                Length: 1
            Field (11/14): FW_EVENT
                Type: FW_EVENT (40005)
                Length: 1
            Field (12/14): FW_EXT_EVENT
                Type: FW_EXT_EVENT (33002)
                Length: 2
            Field (13/14): observationTimeMilliseconds
                Type: observationTimeMilliseconds (323)
                Length: 8
            Field (14/14): BYTES_TOTAL
                Type: BYTES_TOTAL (85)
                Length: 4
        Template (Id = 265, Count = 22)
            Template Id: 265
            Field Count: 22
            Field (1/22): flowId
                Type: flowId (148)
                Length: 4
            Field (2/22): IP_SRC_ADDR
                Type: IP_SRC_ADDR (8)
                Length: 4
            Field (3/22): L4_SRC_PORT
                Type: L4_SRC_PORT (7)
                Length: 2
            Field (4/22): INPUT_SNMP
                Type: INPUT_SNMP (10)
                Length: 2
            Field (5/22): IP_DST_ADDR
                Type: IP_DST_ADDR (12)
                Length: 4
            Field (6/22): L4_DST_PORT
                Type: L4_DST_PORT (11)
                Length: 2
            Field (7/22): OUTPUT_SNMP
                Type: OUTPUT_SNMP (14)
                Length: 2
            Field (8/22): PROTOCOL
                Type: PROTOCOL (4)
                Length: 1
            Field (9/22): ICMP_IPv4_TYPE
                Type: ICMP_IPv4_TYPE (176)
                Length: 1
            Field (10/22): ICMP_IPv4_CODE
                Type: ICMP_IPv4_CODE (177)
                Length: 1
            Field (11/22): XLATE_SRC_ADDR_IPV4
                Type: XLATE_SRC_ADDR_IPV4 (40001)
                Length: 4
            Field (12/22): XLATE_DST_ADDR_IPV4
                Type: XLATE_DST_ADDR_IPV4 (40002)
                Length: 4
            Field (13/22): XLATE_SRC_PORT
                Type: XLATE_SRC_PORT (40003)
                Length: 2
            Field (14/22): XLATE_DST_PORT
                Type: XLATE_DST_PORT (40004)
                Length: 2
            Field (15/22): FW_EVENT
                Type: FW_EVENT (40005)
                Length: 1
            Field (16/22): FW_EXT_EVENT
                Type: FW_EXT_EVENT (33002)
                Length: 2
            Field (17/22): observationTimeMilliseconds
                Type: observationTimeMilliseconds (323)
                Length: 8
            Field (18/22): BYTES_TOTAL
                Type: BYTES_TOTAL (85)
                Length: 4
            Field (19/22): flowStartMilliseconds
                Type: flowStartMilliseconds (152)
                Length: 8
            Field (20/22): INGRESS_ACL_ID
                Type: INGRESS_ACL_ID (33000)
                Length: 12
            Field (21/22): EGRESS_ACL_ID
                Type: EGRESS_ACL_ID (33001)
                Length: 12
            Field (22/22): AAA_USERNAME
                Type: AAA_USERNAME (40000)
                Length: 20
        Template (Id = 266, Count = 22)
            Template Id: 266
            Field Count: 22
            Field (1/22): flowId
                Type: flowId (148)
                Length: 4
            Field (2/22): IP_SRC_ADDR
                Type: IP_SRC_ADDR (8)
                Length: 4
            Field (3/22): L4_SRC_PORT
                Type: L4_SRC_PORT (7)
                Length: 2
            Field (4/22): INPUT_SNMP
                Type: INPUT_SNMP (10)
                Length: 2
            Field (5/22): IP_DST_ADDR
                Type: IP_DST_ADDR (12)
                Length: 4
            Field (6/22): L4_DST_PORT
                Type: L4_DST_PORT (11)
                Length: 2
            Field (7/22): OUTPUT_SNMP
                Type: OUTPUT_SNMP (14)
                Length: 2
            Field (8/22): PROTOCOL
                Type: PROTOCOL (4)
                Length: 1
            Field (9/22): ICMP_IPv4_TYPE
                Type: ICMP_IPv4_TYPE (176)
                Length: 1
            Field (10/22): ICMP_IPv4_CODE
                Type: ICMP_IPv4_CODE (177)
                Length: 1
            Field (11/22): XLATE_SRC_ADDR_IPV4
                Type: XLATE_SRC_ADDR_IPV4 (40001)
                Length: 4
            Field (12/22): XLATE_DST_ADDR_IPV4
                Type: XLATE_DST_ADDR_IPV4 (40002)
                Length: 4
            Field (13/22): XLATE_SRC_PORT
                Type: XLATE_SRC_PORT (40003)
                Length: 2
            Field (14/22): XLATE_DST_PORT
                Type: XLATE_DST_PORT (40004)
                Length: 2
            Field (15/22): FW_EVENT
                Type: FW_EVENT (40005)
                Length: 1
            Field (16/22): FW_EXT_EVENT
                Type: FW_EXT_EVENT (33002)
                Length: 2
            Field (17/22): observationTimeMilliseconds
                Type: observationTimeMilliseconds (323)
                Length: 8
            Field (18/22): BYTES_TOTAL
                Type: BYTES_TOTAL (85)
                Length: 4
            Field (19/22): flowStartMilliseconds
                Type: flowStartMilliseconds (152)
                Length: 8
            Field (20/22): INGRESS_ACL_ID
                Type: INGRESS_ACL_ID (33000)
                Length: 12
            Field (21/22): EGRESS_ACL_ID
                Type: EGRESS_ACL_ID (33001)
                Length: 12
            Field (22/22): AAA_USERNAME
                Type: AAA_USERNAME (40000)
                Length: 65
        Template (Id = 267, Count = 18)
            Template Id: 267
            Field Count: 18
            Field (1/18): flowId
                Type: flowId (148)
                Length: 4
            Field (2/18): IPV6_SRC_ADDR
                Type: IPV6_SRC_ADDR (27)
                Length: 16
            Field (3/18): L4_SRC_PORT
                Type: L4_SRC_PORT (7)
                Length: 2
            Field (4/18): INPUT_SNMP
                Type: INPUT_SNMP (10)
                Length: 2
            Field (5/18): IPV6_DST_ADDR
                Type: IPV6_DST_ADDR (28)
                Length: 16
            Field (6/18): L4_DST_PORT
                Type: L4_DST_PORT (11)
                Length: 2
            Field (7/18): OUTPUT_SNMP
                Type: OUTPUT_SNMP (14)
                Length: 2
            Field (8/18): PROTOCOL
                Type: PROTOCOL (4)
                Length: 1
            Field (9/18): ICMP_IPv6_TYPE
                Type: ICMP_IPv6_TYPE (178)
                Length: 1
            Field (10/18): ICMP_IPv6_CODE
                Type: ICMP_IPv6_CODE (179)
                Length: 1
            Field (11/18): FW_EVENT
                Type: FW_EVENT (40005)
                Length: 1
            Field (12/18): FW_EXT_EVENT
                Type: FW_EXT_EVENT (33002)
                Length: 2
            Field (13/18): observationTimeMilliseconds
                Type: observationTimeMilliseconds (323)
                Length: 8
            Field (14/18): BYTES_TOTAL
                Type: BYTES_TOTAL (85)
                Length: 4
            Field (15/18): flowStartMilliseconds
                Type: flowStartMilliseconds (152)
                Length: 8
            Field (16/18): INGRESS_ACL_ID
                Type: INGRESS_ACL_ID (33000)
                Length: 12
            Field (17/18): EGRESS_ACL_ID
                Type: EGRESS_ACL_ID (33001)
                Length: 12
            Field (18/18): AAA_USERNAME
                Type: AAA_USERNAME (40000)
                Length: 20
        Template (Id = 268, Count = 18)
            Template Id: 268
            Field Count: 18
            Field (1/18): flowId
                Type: flowId (148)
                Length: 4
            Field (2/18): IPV6_SRC_ADDR
                Type: IPV6_SRC_ADDR (27)
                Length: 16
            Field (3/18): L4_SRC_PORT
                Type: L4_SRC_PORT (7)
                Length: 2
            Field (4/18): INPUT_SNMP
                Type: INPUT_SNMP (10)
                Length: 2
            Field (5/18): IPV6_DST_ADDR
                Type: IPV6_DST_ADDR (28)
                Length: 16
            Field (6/18): L4_DST_PORT
                Type: L4_DST_PORT (11)
                Length: 2
            Field (7/18): OUTPUT_SNMP
                Type: OUTPUT_SNMP (14)
                Length: 2
            Field (8/18): PROTOCOL
                Type: PROTOCOL (4)
                Length: 1
            Field (9/18): ICMP_IPv6_TYPE
                Type: ICMP_IPv6_TYPE (178)
                Length: 1
            Field (10/18): ICMP_IPv6_CODE
                Type: ICMP_IPv6_CODE (179)
                Length: 1
            Field (11/18): FW_EVENT
                Type: FW_EVENT (40005)
                Length: 1
            Field (12/18): FW_EXT_EVENT
                Type: FW_EXT_EVENT (33002)
                Length: 2
            Field (13/18): observationTimeMilliseconds
                Type: observationTimeMilliseconds (323)
                Length: 8
            Field (14/18): BYTES_TOTAL
                Type: BYTES_TOTAL (85)
                Length: 4
            Field (15/18): flowStartMilliseconds
                Type: flowStartMilliseconds (152)
                Length: 8
            Field (16/18): INGRESS_ACL_ID
                Type: INGRESS_ACL_ID (33000)
                Length: 12
            Field (17/18): EGRESS_ACL_ID
                Type: EGRESS_ACL_ID (33001)
                Length: 12
            Field (18/18): AAA_USERNAME
                Type: AAA_USERNAME (40000)
                Length: 65
'''

data = '\x00\t\x00\x0e\x00\x1f\x80\xfdV\x17\x8dG\x00\x00\x02\x96\x00\x00\x00\x00\x01\t\x05\x98\x00\x00!4\xc0\xa8\x0e\x01\x00\x00\x00\x03\x02\x02\x02\x0bD\x8d\x00\x02\x01\x00\x00\xc0\xa8\x0e\x01\x02\x02\x02\x0b\x00\x00D\x8d\x02\x07\xe9\x00\x00\x01PK\xff\xd7\xdf\x00\x00\x008\x00\x00\x01PK\xff\xcf\xf1\x0f\x8e\x7f\xf3\xfc\x1a\x03\x0f\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00!5\xc0\xa8\x17\x16D\x8d\x00\x02\xa4\xa4%\x0b\x00\x00\x00\x03\x01\x08\x00\xc0\xa8\x17\x16\xa4\xa4%\x0bD\x8d\x00\x00\x02\x07\xe9\x00\x00\x01PK\xff\xda#\x00\x00\x008\x00\x00\x01PK\xff\xd2I\x0f\x8e\x7f\xf3\xfc\x1a\x03\x0f\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00!6\xa4\xa4%\x0b\x00\x00\x00\x03\xc0\xa8\x17\x16D\x8d\x00\x02\x01\x00\x00\xa4\xa4%\x0b\xc0\xa8\x17\x16\x00\x00D\x8d\x02\x07\xe9\x00\x00\x01PK\xff\xdaK\x00\x00\x008\x00\x00\x01PK\xff\xd2S\x0f\x8e\x7f\xf3\xfc\x1a\x03\x0f\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00!7\xc0\xa8\x17\x14E\x8d\x00\x02\xa4\xa4%\x0b\x00\x00\x00\x03\x01\x08\x00\xc0\xa8\x17\x14\xa4\xa4%\x0bE\x8d\x00\x00\x02\x07\xe9\x00\x00\x01PK\xff\xdb\x13\x00\x00\x008\x00\x00\x01PK\xff\xd3/\x0f\x8e\x7f\xf3\xfc\x1a\x03\x0f\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00!8\xa4\xa4%\x0b\x00\x00\x00\x03\xc0\xa8\x17\x14E\x8d\x00\x02\x01\x00\x00\xa4\xa4%\x0b\xc0\xa8\x17\x14\x00\x00E\x8d\x02\x07\xe9\x00\x00\x01PK\xff\xdb\x1d\x00\x00\x008\x00\x00\x01PK\xff\xd39\x0f\x8e\x7f\xf3\xfc\x1a\x03\x0f\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00!9\xc0\xa8\x0e\x0bE\x8d\x00\x03\x02\x02\x02\x0b\x00\x00\x00\x02\x01\x08\x00\xc0\xa8\x0e\x0b\x02\x02\x02\x0bE\x8d\x00\x00\x02\x07\xe9\x00\x00\x01PK\xff\xdb\xdb\x00\x00\x008\x00\x00\x01PK\xff\xd3\xed\x0f\x8e\x7f\xf3\xfc\x1a\x03\x0f\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00!:\x02\x02\x02\x0b\x00\x00\x00\x02\xc0\xa8\x0e\x0bE\x8d\x00\x03\x01\x00\x00\x02\x02\x02\x0b\xc0\xa8\x0e\x0b\x00\x00E\x8d\x02\x07\xe9\x00\x00\x01PK\xff\xdb\xef\x00\x00\x008\x00\x00\x01PK\xff\xd3\xf7\x0f\x8e\x7f\xf3\xfc\x1a\x03\x0f\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00!;\x02\x02\x02\x0bE\x8d\x00\x02\xc0\xa8\x0e\x01\x00\x00\x00\x03\x01\x08\x00\x02\x02\x02\x0b\xc0\xa8\x0e\x01E\x8d\x00\x00\x02\x07\xe9\x00\x00\x01PK\xff\xdb\xef\x00\x00\x008\x00\x00\x01PK\xff\xd4\x01\x0f\x8e\x7f\xf3\xfc\x1a\x03\x0f\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00!<\xc0\xa8\x0e\x01\x00\x00\x00\x03\x02\x02\x02\x0bE\x8d\x00\x02\x01\x00\x00\xc0\xa8\x0e\x01\x02\x02\x02\x0b\x00\x00E\x8d\x02\x07\xe9\x00\x00\x01PK\xff\xdb\xef\x00\x00\x008\x00\x00\x01PK\xff\xd4\x0b\x0f\x8e\x7f\xf3\xfc\x1a\x03\x0f\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00!M\xa4\xa4%\x0b\x00\x00\x00\x03\xc0\xa8\x17\x01\x00\x00\x00\x02\x01\x03\x03\xa4\xa4%\x0b\xc0\xa8\x17\x01\x00\x00\x00\x00\x02\x07\xe0\x00\x00\x01PK\xff\xdee\x00\x00\x00\xa0\x00\x00\x01PK\xff\xdee\x0f\x8e\x7f\xf3\xfc\x1a\x03\x0f\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00!=\xc0\xa8\x17\x16F\x8d\x00\x02\xa4\xa4%\x0b\x00\x00\x00\x03\x01\x08\x00\xc0\xa8\x17\x16\xa4\xa4%\x0bF\x8d\x00\x00\x02\x07\xe9\x00\x00\x01PK\xff\xdee\x00\x00\x008\x00\x00\x01PK\xff\xd6\x81\x0f\x8e\x7f\xf3\xfc\x1a\x03\x0f\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00!>\xa4\xa4%\x0b\x00\x00\x00\x03\xc0\xa8\x17\x16F\x8d\x00\x02\x01\x00\x00\xa4\xa4%\x0b\xc0\xa8\x17\x16\x00\x00F\x8d\x02\x07\xe9\x00\x00\x01PK\xff\xdey\x00\x00\x008\x00\x00\x01PK\xff\xd6\x8b\x0f\x8e\x7f\xf3\xfc\x1a\x03\x0f\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00!?\xc0\xa8\x17\x14F\x8d\x00\x02\xa4\xa4%\x0b\x00\x00\x00\x03\x01\x08\x00\xc0\xa8\x17\x14\xa4\xa4%\x0bF\x8d\x00\x00\x02\x07\xe9\x00\x00\x01PK\xff\xdfA\x00\x00\x008\x00\x00\x01PK\xff\xd7]\x0f\x8e\x7f\xf3\xfc\x1a\x03\x0f\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00!@\xa4\xa4%\x0b\x00\x00\x00\x03\xc0\xa8\x17\x14F\x8d\x00\x02\x01\x00\x00\xa4\xa4%\x0b\xc0\xa8\x17\x14\x00\x00F\x8d\x02\x07\xe9\x00\x00\x01PK\xff\xdfU\x00\x00\x008\x00\x00\x01PK\xff\xd7g\x0f\x8e\x7f\xf3\xfc\x1a\x03\x0f\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00'

'''
Cisco NetFlow/IPFIX
    Version: 9
    Count: 14
    SysUptime: 2064.637000000 seconds
    Timestamp: Oct  9, 2015 03:47:51.000000000 MDT
        CurrentSecs: 1444384071
    FlowSequence: 662
    SourceId: 0
    FlowSet 1 [id=265] (14 flows)
        FlowSet Id: (Data) (265)
        FlowSet Length: 1432
        [Template Frame: 1]
        Flow 1
            Flow Id: 8500
            SrcAddr: 192.168.14.1
            SrcPort: 0
            InputInt: 3
            DstAddr: 2.2.2.11
            DstPort: 17549
            OutputInt: 2
            Protocol: ICMP (1)
            IPv4 ICMP Type: 0
            IPv4 ICMP Code: 0
            Post NAT Source IPv4 Address: 192.168.14.1
            Post NAT Destination IPv4 Address: 2.2.2.11
            Post NAPT Source Transport Port: 0
            Post NAPT Destination Transport Port: 17549
            Firewall Event: Flow deleted (2)
            Extended firewall event code: Unknown (2025)
            Observation Time Milliseconds: Oct  9, 2015 03:47:49.599000000 MDT
            Permanent Octets: 56
            Ingress ACL ID: 0f8e7ff3fc1a030f00000000
            Egress ACL ID: 000000000000000000000000
            AAA username:
            StartTime: Oct  9, 2015 03:47:47.569000000 MDT
        Flow 2
            Flow Id: 8501
            SrcAddr: 192.168.23.22
            SrcPort: 17549
            InputInt: 2
            DstAddr: 164.164.37.11
            DstPort: 0
            OutputInt: 3
            Protocol: ICMP (1)
            IPv4 ICMP Type: 8
            IPv4 ICMP Code: 0
            Post NAT Source IPv4 Address: 192.168.23.22
            Post NAT Destination IPv4 Address: 164.164.37.11
            Post NAPT Source Transport Port: 17549
            Post NAPT Destination Transport Port: 0
            Firewall Event: Flow deleted (2)
            Extended firewall event code: Unknown (2025)
            Observation Time Milliseconds: Oct  9, 2015 03:47:50.179000000 MDT
            Permanent Octets: 56
            Ingress ACL ID: 0f8e7ff3fc1a030f00000000
            Egress ACL ID: 000000000000000000000000
            AAA username:
            StartTime: Oct  9, 2015 03:47:48.169000000 MDT
        Flow 3
            Flow Id: 8502
            SrcAddr: 164.164.37.11
            SrcPort: 0
            InputInt: 3
            DstAddr: 192.168.23.22
            DstPort: 17549
            OutputInt: 2
            Protocol: ICMP (1)
            IPv4 ICMP Type: 0
            IPv4 ICMP Code: 0
            Post NAT Source IPv4 Address: 164.164.37.11
            Post NAT Destination IPv4 Address: 192.168.23.22
            Post NAPT Source Transport Port: 0
            Post NAPT Destination Transport Port: 17549
            Firewall Event: Flow deleted (2)
            Extended firewall event code: Unknown (2025)
            Observation Time Milliseconds: Oct  9, 2015 03:47:50.219000000 MDT
            Permanent Octets: 56
            Ingress ACL ID: 0f8e7ff3fc1a030f00000000
            Egress ACL ID: 000000000000000000000000
            AAA username:
            StartTime: Oct  9, 2015 03:47:48.179000000 MDT
        Flow 4
            Flow Id: 8503
            SrcAddr: 192.168.23.20
            SrcPort: 17805
            InputInt: 2
            DstAddr: 164.164.37.11
            DstPort: 0
            OutputInt: 3
            Protocol: ICMP (1)
            IPv4 ICMP Type: 8
            IPv4 ICMP Code: 0
            Post NAT Source IPv4 Address: 192.168.23.20
            Post NAT Destination IPv4 Address: 164.164.37.11
            Post NAPT Source Transport Port: 17805
            Post NAPT Destination Transport Port: 0
            Firewall Event: Flow deleted (2)
            Extended firewall event code: Unknown (2025)
            Observation Time Milliseconds: Oct  9, 2015 03:47:50.419000000 MDT
            Permanent Octets: 56
            Ingress ACL ID: 0f8e7ff3fc1a030f00000000
            Egress ACL ID: 000000000000000000000000
            AAA username:
            StartTime: Oct  9, 2015 03:47:48.399000000 MDT
        Flow 5
            Flow Id: 8504
            SrcAddr: 164.164.37.11
            SrcPort: 0
            InputInt: 3
            DstAddr: 192.168.23.20
            DstPort: 17805
            OutputInt: 2
            Protocol: ICMP (1)
            IPv4 ICMP Type: 0
            IPv4 ICMP Code: 0
            Post NAT Source IPv4 Address: 164.164.37.11
            Post NAT Destination IPv4 Address: 192.168.23.20
            Post NAPT Source Transport Port: 0
            Post NAPT Destination Transport Port: 17805
            Firewall Event: Flow deleted (2)
            Extended firewall event code: Unknown (2025)
            Observation Time Milliseconds: Oct  9, 2015 03:47:50.429000000 MDT
            Permanent Octets: 56
            Ingress ACL ID: 0f8e7ff3fc1a030f00000000
            Egress ACL ID: 000000000000000000000000
            AAA username:
            StartTime: Oct  9, 2015 03:47:48.409000000 MDT
        Flow 6
            Flow Id: 8505
            SrcAddr: 192.168.14.11
            SrcPort: 17805
            InputInt: 3
            DstAddr: 2.2.2.11
            DstPort: 0
            OutputInt: 2
            Protocol: ICMP (1)
            IPv4 ICMP Type: 8
            IPv4 ICMP Code: 0
            Post NAT Source IPv4 Address: 192.168.14.11
            Post NAT Destination IPv4 Address: 2.2.2.11
            Post NAPT Source Transport Port: 17805
            Post NAPT Destination Transport Port: 0
            Firewall Event: Flow deleted (2)
            Extended firewall event code: Unknown (2025)
            Observation Time Milliseconds: Oct  9, 2015 03:47:50.619000000 MDT
            Permanent Octets: 56
            Ingress ACL ID: 0f8e7ff3fc1a030f00000000
            Egress ACL ID: 000000000000000000000000
            AAA username:
            StartTime: Oct  9, 2015 03:47:48.589000000 MDT
        Flow 7
            Flow Id: 8506
            SrcAddr: 2.2.2.11
            SrcPort: 0
            InputInt: 2
            DstAddr: 192.168.14.11
            DstPort: 17805
            OutputInt: 3
            Protocol: ICMP (1)
            IPv4 ICMP Type: 0
            IPv4 ICMP Code: 0
            Post NAT Source IPv4 Address: 2.2.2.11
            Post NAT Destination IPv4 Address: 192.168.14.11
            Post NAPT Source Transport Port: 0
            Post NAPT Destination Transport Port: 17805
            Firewall Event: Flow deleted (2)
            Extended firewall event code: Unknown (2025)
            Observation Time Milliseconds: Oct  9, 2015 03:47:50.639000000 MDT
            Permanent Octets: 56
            Ingress ACL ID: 0f8e7ff3fc1a030f00000000
            Egress ACL ID: 000000000000000000000000
            AAA username:
            StartTime: Oct  9, 2015 03:47:48.599000000 MDT
        Flow 8
            Flow Id: 8507
            SrcAddr: 2.2.2.11
            SrcPort: 17805
            InputInt: 2
            DstAddr: 192.168.14.1
            DstPort: 0
            OutputInt: 3
            Protocol: ICMP (1)
            IPv4 ICMP Type: 8
            IPv4 ICMP Code: 0
            Post NAT Source IPv4 Address: 2.2.2.11
            Post NAT Destination IPv4 Address: 192.168.14.1
            Post NAPT Source Transport Port: 17805
            Post NAPT Destination Transport Port: 0
            Firewall Event: Flow deleted (2)
            Extended firewall event code: Unknown (2025)
            Observation Time Milliseconds: Oct  9, 2015 03:47:50.639000000 MDT
            Permanent Octets: 56
            Ingress ACL ID: 0f8e7ff3fc1a030f00000000
            Egress ACL ID: 000000000000000000000000
            AAA username:
            StartTime: Oct  9, 2015 03:47:48.609000000 MDT
        Flow 9
            Flow Id: 8508
            SrcAddr: 192.168.14.1
            SrcPort: 0
            InputInt: 3
            DstAddr: 2.2.2.11
            DstPort: 17805
            OutputInt: 2
            Protocol: ICMP (1)
            IPv4 ICMP Type: 0
            IPv4 ICMP Code: 0
            Post NAT Source IPv4 Address: 192.168.14.1
            Post NAT Destination IPv4 Address: 2.2.2.11
            Post NAPT Source Transport Port: 0
            Post NAPT Destination Transport Port: 17805
            Firewall Event: Flow deleted (2)
            Extended firewall event code: Unknown (2025)
            Observation Time Milliseconds: Oct  9, 2015 03:47:50.639000000 MDT
            Permanent Octets: 56
            Ingress ACL ID: 0f8e7ff3fc1a030f00000000
            Egress ACL ID: 000000000000000000000000
            AAA username:
            StartTime: Oct  9, 2015 03:47:48.619000000 MDT
        Flow 10
            Flow Id: 8525
            SrcAddr: 164.164.37.11
            SrcPort: 0
            InputInt: 3
            DstAddr: 192.168.23.1
            DstPort: 0
            OutputInt: 2
            Protocol: ICMP (1)
            IPv4 ICMP Type: 3
            IPv4 ICMP Code: 3
            Post NAT Source IPv4 Address: 164.164.37.11
            Post NAT Destination IPv4 Address: 192.168.23.1
            Post NAPT Source Transport Port: 0
            Post NAPT Destination Transport Port: 0
            Firewall Event: Flow deleted (2)
            Extended firewall event code: Unknown (2016)
            Observation Time Milliseconds: Oct  9, 2015 03:47:51.269000000 MDT
            Permanent Octets: 160
            Ingress ACL ID: 0f8e7ff3fc1a030f00000000
            Egress ACL ID: 000000000000000000000000
            AAA username:
            StartTime: Oct  9, 2015 03:47:51.269000000 MDT
        Flow 11
            Flow Id: 8509
            SrcAddr: 192.168.23.22
            SrcPort: 18061
            InputInt: 2
            DstAddr: 164.164.37.11
            DstPort: 0
            OutputInt: 3
            Protocol: ICMP (1)
            IPv4 ICMP Type: 8
            IPv4 ICMP Code: 0
            Post NAT Source IPv4 Address: 192.168.23.22
            Post NAT Destination IPv4 Address: 164.164.37.11
            Post NAPT Source Transport Port: 18061
            Post NAPT Destination Transport Port: 0
            Firewall Event: Flow deleted (2)
            Extended firewall event code: Unknown (2025)
            Observation Time Milliseconds: Oct  9, 2015 03:47:51.269000000 MDT
            Permanent Octets: 56
            Ingress ACL ID: 0f8e7ff3fc1a030f00000000
            Egress ACL ID: 000000000000000000000000
            AAA username:
            StartTime: Oct  9, 2015 03:47:49.249000000 MDT
        Flow 12
            Flow Id: 8510
            SrcAddr: 164.164.37.11
            SrcPort: 0
            InputInt: 3
            DstAddr: 192.168.23.22
            DstPort: 18061
            OutputInt: 2
            Protocol: ICMP (1)
            IPv4 ICMP Type: 0
            IPv4 ICMP Code: 0
            Post NAT Source IPv4 Address: 164.164.37.11
            Post NAT Destination IPv4 Address: 192.168.23.22
            Post NAPT Source Transport Port: 0
            Post NAPT Destination Transport Port: 18061
            Firewall Event: Flow deleted (2)
            Extended firewall event code: Unknown (2025)
            Observation Time Milliseconds: Oct  9, 2015 03:47:51.289000000 MDT
            Permanent Octets: 56
            Ingress ACL ID: 0f8e7ff3fc1a030f00000000
            Egress ACL ID: 000000000000000000000000
            AAA username:
            StartTime: Oct  9, 2015 03:47:49.259000000 MDT
        Flow 13
            Flow Id: 8511
            SrcAddr: 192.168.23.20
            SrcPort: 18061
            InputInt: 2
            DstAddr: 164.164.37.11
            DstPort: 0
            OutputInt: 3
            Protocol: ICMP (1)
            IPv4 ICMP Type: 8
            IPv4 ICMP Code: 0
            Post NAT Source IPv4 Address: 192.168.23.20
            Post NAT Destination IPv4 Address: 164.164.37.11
            Post NAPT Source Transport Port: 18061
            Post NAPT Destination Transport Port: 0
            Firewall Event: Flow deleted (2)
            Extended firewall event code: Unknown (2025)
            Observation Time Milliseconds: Oct  9, 2015 03:47:51.489000000 MDT
            Permanent Octets: 56
            Ingress ACL ID: 0f8e7ff3fc1a030f00000000
            Egress ACL ID: 000000000000000000000000
            AAA username:
            StartTime: Oct  9, 2015 03:47:49.469000000 MDT
        Flow 14
            Flow Id: 8512
            SrcAddr: 164.164.37.11
            SrcPort: 0
            InputInt: 3
            DstAddr: 192.168.23.20
            DstPort: 18061
            OutputInt: 2
            Protocol: ICMP (1)
            IPv4 ICMP Type: 0
            IPv4 ICMP Code: 0
            Post NAT Source IPv4 Address: 164.164.37.11
            Post NAT Destination IPv4 Address: 192.168.23.20
            Post NAPT Source Transport Port: 0
            Post NAPT Destination Transport Port: 18061
            Firewall Event: Flow deleted (2)
            Extended firewall event code: Unknown (2025)
            Observation Time Milliseconds: Oct  9, 2015 03:47:51.509000000 MDT
            Permanent Octets: 56
            Ingress ACL ID: 0f8e7ff3fc1a030f00000000
            Egress ACL ID: 000000000000000000000000
            AAA username:
            StartTime: Oct  9, 2015 03:47:49.479000000 MDT
'''

host = sys.argv[1]
port = 2055
N = 150000
flowsPerPacket = 14

sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
sock.sendto(tpl, (host, port))
time.sleep(0.2)

ts = time.time()
print("%d: started sending %d Cisco ASA flows in %d packets totaling %d bytes" % (ts, N*flowsPerPacket, N, N*len(data)))
print("%d: flow size %d, packet size %d" % (ts, len(data) / flowsPerPacket, len(data)))

for i in range(0, N):
    sock.sendto(data, (host, port))
