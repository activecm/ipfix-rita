#!/usr/bin/env bash

PCAP_IN=$1
PCAP_OUT=$2

INTERVAL=86400

function __help()
{
	echo "Usage: $0 input_pcap output_pcap"
	echo "The first 24 hours of data from input_pcap will be aligned to the current day in output_pcap"
	exit
}

if [ "$PCAP_IN" = "-h" -o "$PCAP_IN" = "--help" -o -z "$PCAP_IN" -o -z "$PCAP_OUT" ]; then
	__help
fi

# Create a temporary directory to work in
WORK_DIR=`mktemp -d` || exit 1 # Exit if we can't get the temp dir

editcap -i $INTERVAL "$PCAP_IN" "$WORK_DIR/$(basename $PCAP_IN)"

INTERVAL_FILES=`find $WORK_DIR -maxdepth 1 -type f`

NUM_INTERVAL_FILES=`echo "$INTERVAL_FILES" | wc -l`

if [ $NUM_INTERVAL_FILES -gt 1 ]; then
	echo "$(basename $PCAP_IN) consists of multiple 24H intervals."
	echo "Which interval would you like to use?"
	echo "NOTE: The last interval may not contain a full 24H capture."
	while [ -z "$INTERVAL_PCAP" ]; do
		echo ""
		echo "$(basename $PCAP_IN) contained $NUM_INTERVAL_FILES intervals."
		for file in $INTERVAL_FILES; do
			printf "\t$(basename $file)\n"
		done
		for file in $INTERVAL_FILES; do
			echo ""
			capinfos -aecst "$file"
			read -p "Align $(basename $file) to today? (y/n) [n] " -r
			if [[ "$REPLY" =~ ^[Yy] ]]; then
				INTERVAL_PCAP="$file"
				echo ""
				break
			fi
		done
	done
fi


CURR_DATE=`date -I`
LAST_MIDNIGHT_TS=`date +%s --date="$CURR_DATE"`

INTERVAL_PCAP_DATE=`capinfos $INTERVAL_PCAP | grep 'First packet time' | cut -d' ' -f6-`
INTERVAL_PCAP_TS=`date +%s --date="$INTERVAL_PCAP_DATE"`

TS_OFFSET=`expr $LAST_MIDNIGHT_TS - $INTERVAL_PCAP_TS`

echo "Writing out $PCAP_OUT"

editcap -t $TS_OFFSET "$INTERVAL_PCAP" "$PCAP_OUT"

rm -rf "$WORK_DIR"
