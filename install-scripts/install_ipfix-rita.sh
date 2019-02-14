#!/usr/bin/env bash

askYN () {
	TESTYN=""
	while [ "$TESTYN" != 'Y' ] && [ "$TESTYN" != 'N' ] ; do
		echo -n '?' >&2
		read TESTYN || :
		case $TESTYN in
		T*|t*|Y*|y*)		TESTYN='Y'	;;
		F*|f*|N*|n*)		TESTYN='N'	;;
		esac
	done

	if [ "$TESTYN" = 'Y' ]; then
		return 0 #True
	else
		return 1 #False
	fi
} #End of askYN

check_remote_mongo() {
	hostip=$1
	if [`echo 'show dbs' | mongo --host $hostip | grep '^bye'` = 'bye']; then
		return 0 #True
	else
		return 1 #False
	fi
} #End of check_remote_mongo


strip_mongo() {
	MONGO_URI_STR=$1

	if [ `echo $MONGO_URI_STR | grep mongo` ] ; then
		RITA_STR=${MONGO_URI_STR##m*/}
		RITA_STR=${RITA_STR%:*}
	else
		RITA_STR=$MONGO_URI_STR
	fi

	echo $RITA_STR
}


add_mongo() {
	URI_STR=$1

	if [ `echo $URI_STR | grep mongo` ] ; then
		MONGO_URI=$URI_STR
	else
		MONGO_URI="mongodb://$URI_STR:27017"
	fi

	echo $MONGO_URI
}



can_ssh () {
	#Test that we can reach the target system over ssh.
	success_code=1
	if [ "$1" = "127.0.0.1" ]; then
		success_code=0
	elif [ -n "$1" ]; then
		token="$RANDOM.$RANDOM"
		echo "Attempting to verify that we can ssh to $@ - you may need to provide a password to access this system." >&2
		ssh_out=`ssh "$@" '/bin/echo '"$token"`
		if [ "$token" = "$ssh_out" ]; then
			#echo "successfully connected to $@"
			success_code=0
		#else
			#echo "cannot connect to $@"
		fi
	else
		echo "Please supply an ssh target as a command line parameter to can_ssh" >&2
	fi

	return $success_code
} #End of can_ssh



fail () {
	echo "$* ." >&2
	echo "This is normally an unrecoverable problem, and we recommend fixing the problem and restarting the script.  Please contact technical support for help in resolving the issue.  If you feel the script should continue, enter   Y   and the script will attempt to finish.  Entering   N    will cause this script to exit." >&2
	if askYN ; then
		echo "Script will continue at user request.  This may not result in a working configuration." >&2
		sleep 5
	else
		exit 1
	fi
} #end of fail


#TODO update this and makes sure it's called before running the full install script
# do_system_tests () {
#	while [ -n "$1" ]; do
#		if [ "$1" = "check_sse4_2" ]; then
#			status "Hardware tests"
#			require_file /proc/cpuinfo			|| fail "Missing /proc/cpuinfo - is this a Linux system?"
#		fi
#		shift
#	done

#	for one_dir in $HOME / /usr ; do
		#"tr" in next command removes the linefeed at the end of the number of free megabytes (and any other whitespace)
#		[ `df -h $one_dir --output=avail --block-size=1048576 | grep -v '^Avail' | tr -dc '[0-9]'` -gt 5000 ]	|| fail "${one_dir} has less than 5GB free space"
#	done

#	status "Installed software tests"
#	if [ -x /usr/bin/apt-get -a -x /usr/bin/dpkg-query ]; then
		#We have apt-get , good.
#		apt-get -qq update > /dev/null 2>&1
#		inst_retcode=100000
#		while [ "$inst_retcode" -gt 0 ]; do
#			apt-get -y install gdb git wget curl make realpath lsb-release rsync tar
#			inst_retcode=$?
#			if [ "$inst_retcode" -gt 0 ]; then
#				echo "Error installing packages, perhaps because a system update is running; will wait 60 seconds and try again" >&2
#				sleep 60
#			fi
#		done
		#TODO test IPFIX-RITA on Contos/Red Hat System
	#elif [ -x /usr/bin/yum -a -x /bin/rpm ]; then
	#	if [ ! -x /bin/yum-config-manager ]; then
	#		yum -y install yum-utils
	#	fi
		#We have yum, good.
	#	yum -q makecache > /dev/null 2>&1
	#	#FIXME - put in place a similar loop like above for apt-get
	#	yum -y install gdb git wget curl make coreutils coreutils redhat-lsb-core rsync tar
#	else
#		fail "(apt-get and dpkg-query) is installed on this system"
#	fi
#	require_util awk cat cp curl date egrep gdb getent git grep ip lsb_release make mkdir mv printf rm rsync sed sleep tar tr wc wget		|| fail "A needed tool is missing"

#	if [ -s /etc/redhat-release ] && grep -iq 'release 7' /etc/redhat-release ; then
#		echo "Centos or Redhat 7 installation detected, good." >&2
#	elif grep -iq '^DISTRIB_ID *= *Ubuntu' /etc/lsb-release ; then
#		echo "Ubuntu Linux installation detected, good." >&2
#	else
#		fail "This system does not appear to be a Centos/RHEL 7 or Ubuntu Linux system"
#	fi

#	return 0
# } #End of do_system_tests

usage_text () {
	cat >&2 <<EOHELP
This script will install ipfix-rita.
On the command line, enter one of the following:
$0
$0 rita ip.address.for.rita
The IP address can be 127.0.0.1 to indicate a local RITA installation.
EOHELP
	exit 1
}

validated_ssh_target () {
	#Return an ssh target that we've confirmed we can reach.  "$2" is the initial target to try, and may be replaced.
	#The potential target - both as supplied as parameter 2 and as returned at the end - may be blank, indicating "do not install this"
	target_type="$1"
	potential_target="$2"

	echo 'About to execute commands on '"$potential_target"' .  You may be prompted one or more times for the password for this system.' >&2
	echo 'Note: if you are using an ssh client other than openssh and have not set up ssh key access to this system, you may be prompted for your password multiple times.' >&2
	while [ -n "$potential_target" ] && ! can_ssh "$potential_target" ; do
		echo "What is the hostname or IP address where you would like to send IPFIX logs to?  Enter 127.0.0.1 if you want to run on this system.  Press Ctrl-C to exit." >&2
		read potential_target <&2
		if [ -n "$potential_target" -a "$potential_target" != '127.0.0.1' ]; then
			echo "Do you access that system with a username different than ${USER}?  If so, enter that username now, otherwise just press enter if your account is ${USER} on the remote system too." >&2
			read potential_user <&2
			if [ -n "$potential_user" ]; then
				potential_target="${potential_user}@${potential_target}"
			fi
		fi
	done

	echo "$potential_target"
} #End of validated_ssh_tartget

get_line() {
	rita_file=$1
	search_key=$2

	#get the line that matches the search key that isn't commented out
	line=`cat $rita_file | grep $search_key`
	line=`echo $line | cut -d ' ' -f 2`

	echo $line
} #End of get line

get_rita_data_interactive() {
	IPFIX_RITA_NETWORK_GATEWAY=$(docker inspect ipfix_rita_default --format "{{with (index .IPAM.Config 0)}}{{.Gateway}}{{end}}")
	RITA_MONGO_URI="mongodb://$IPFIX_RITA_NETWORK_GATEWAY:27017"

	echo "" >&2
	echo "IPFIX-RITA needs to write to a MongoDB database controlled by RITA." >&2
	echo "By default, this installer assumes RITA and MongoDB are installed on the Docker host." >&2
	echo "In order to support this type of installation, you will need to" >&2
	echo "add the suggested Docker interface below to the list of bindIP's in /etc/mongod.conf." >&2
	echo "If needed, please do so, and restart MongoDB before continuing." >&2
	echo "Restart MongoDB with \$ sudo systemctl restart mongod.service" >&2
	echo "Note: the default configuration is not recommended. IPFIX-RITA will likely perform" >&2
	echo "better if it is installed on a machine separate from RITA/ MongoDB." >&2
	echo "" >&2

	echo "What MongoDB URI should IPFIX-RITA use to contact the RITA database [$RITA_MONGO_URI]: " >&2
	read RITA_MONGO_URI <&2
	if [  -n "$RITA_MONGO_URI" -a  "$RITA_MONGO_URI" != '127.0.0.1' ]; then
		rita_uri=`strip_mongo $RITA_MONGO_URI`
		RITA_MONGO_URI=`add_mongo $RITA_MONGO_URI`
	        echo -n "Do you access that system with a username different than ${USER} (Y/N)" >&2
	        if askYN ; then
	                echo "Enter that username now, otherwise just press enter if your account is ${USER} on the remote system too." >&2
	                read rita_user
	                if [ -n "$rita_user" ]; then
	                        rita_system="${rita_user}@${rita_uri}"
	                fi
	        fi
					#This will check if the Rita system is accessible to our installer
					#TODO, this cases issues, we have mongodb://127.0.0.1:27017 then we
					# have ritauser@mongodb://127.0.0.1:27017 need to detect if they have
					# mongodb and the port before trying to validate
	        rita_system=`validated_ssh_target rita "$rita_system"`
	else
	        rita_system="127.0.0.1"
	fi

	#TODO check if the mongo server is available

	RITA_MONGO_AUTH="null"
	#instead of prompting get the RITA address and split from there to fill in our
	#  config file
	echo "" >&2
	echo "Which authentication scheme should be used to contact the database if any? [None]" >&2
	echo "1) None" >&2
	echo "2) SCRAM-SHA-1" >&2
	echo "3) MONGODB-CR" >&2

	while read  <&2 && [[ ! ( "$REPLY" =~ ^[123]$ || -z "$REPLY" ) ]]; do
		echo "Which authentication scheme should be used to contact the database if any? [None]" >&2
		echo "1) None" >&2
		echo "2) SCRAM-SHA-1" >&2
		echo "3) MONGODB-CR" >&2
	done

	if [ "$REPLY" = "2" ]; then
		RITA_MONGO_AUTH="SCRAM-SHA-1"
	elif [ "$REPLY" = "3" ]; then
		RITA_MONGO_AUTH="MONGODB-CR"
	fi

	RITA_MONGO_TLS="false"
	RITA_MONGO_TLS_CHECK_CERT="false"
	RITA_MONGO_TLS_CERT_PATH="null"
	echo "" >&2
	read -p "Does the MongoDB server accept TLS connections? (y/n) [n] "  -r  <&2
	if [[ "$REPLY" =~ ^[Yy]$ ]]; then
		RITA_MONGO_TLS="true"
		RITA_MONGO_TLS_CHECK_CERT="true"
		RITA_MONGO_TLS_CERT_PATH="null"
		read -p "Would you like to provide a certificate authority? (y/n) [n] "  -r  <&2
		if [[ "$REPLY" =~ ^[Yy]$ ]]; then
			read -p "CA Path: " <&2
			RITA_MONGO_TLS_CERT_PATH="$REPLY"
		fi

		if [ "$RITA_MONGO_TLS_CERT_PATH" = "null" ]; then
			read -p "Would you like to disable certificate checks? [n] "  -r  <&2
			if [[ "$REPLY" =~ ^[Yy]$ ]]; then
				RITA_MONGO_TLS_CHECK_CERT="false"
			fi
		fi
	fi

	echo "" >&2
	echo "Each dataset produced with IPFIX-RITA will be named DBROOT-DATE" >&2
	echo "where DBROOT consists of alphanumerics, underscores, and hyphens." >&2
	RITA_DATASET_DBROOT="IPFIX"
	read -p "What would you like to set DBROOT to for this IPFIX collector? [IPFIX] " -r  <&2
	if [ -n "$REPLY" ]; then
		RITA_DATASET_DBROOT="$REPLY"
	fi

	echo "Sending $RITA_MONGO_AUTH to the function"
	write_converter_conf $RITA_DATASET_DBROOT $RITA_MONGO_URI $RITA_MONGO_AUTH $RITA_MONGO_TLS $RITA_MONGO_TLS_CHECK_CERT $RITA_MONGO_TLS_CERT_PATH
} #End of get_rita_data_interactive

get_rita_data_noninteractive() {
	rita_system=$1
	if [ -n "$rita_system" -a  "$rita_system" != '127.0.0.1' ]; then
		rita_system=`strip_mongo $rita_system`
	        echo -n "Do you access that system with a username different than ${USER} (Y/N)" >&2
	        if askYN ; then
	                echo "Enter that username now, otherwise just press enter if your account is ${USER} on the remote system too." >&2
	                read rita_user <&2
	                if [ -n "$rita_user" ]; then
	                        rita_system="${rita_user}@${rita_system}"
	                fi
	        fi
	        rita_system=`validated_ssh_target rita "$rita_system"`
	else
	        rita_system="127.0.0.1"
	fi

	if [ "$rita_system" != '127.0.0.1' ]; then
	        scp $rita_system:/etc/rita/config.yaml .
	        rita_conf="./config.yaml"
	        delete_conf=true
	else
	        #do something great here
	        rita_conf="/etc/rita/config.yaml"
	fi

	db_root="IPFIX"
	mongo_uri="mongodb://${rita_system#*@}:27017"
	mongo_auth=`get_line $rita_conf "AuthenticationMechanism: "`
	mongo_tls_enable=`get_line $rita_conf "Enable: "`
	mongo_tls_cert_check=`get_line $rita_conf "VerifyCertificate: "`
	mongo_tls_ca_path=`get_line $rita_conf "CAFile: "`

	#If the config file was copied from a remote server, delete that file
	if [ "$delete_conf" == "true" ]; then
	        rm "./config.yaml"
	fi

	#If the config specified a CA file, let's copy that
	if [[ $mongo_tls_ca_path != *"null" ]]; then
		ca_file=${mongo_tls_ca_path#"CAFile: "}
		scp $rita_system:$ca_file $ca_file
	fi

	write_converter_conf $db_root $mongo_uri $mongo_auth $mongo_tls_enable $mongo_tls_cert_check $mongo_tls_ca_path
} #End of get_rita_data_noninteractive

write_converter_conf() {
	RITA_DATASET_DBROOT=$1
	RITA_MONGO_URI=$2
	RITA_MONGO_AUTH=$3
	RITA_MONGO_TLS=$4
	RITA_MONGO_TLS_CHECK_CERT=$5
	RITA_MONGO_TLS_CERT_PATH=$6

awk -v db_root="$RITA_DATASET_DBROOT" \
-v mongo_uri="$RITA_MONGO_URI" \
-v mongo_auth="$RITA_MONGO_AUTH" \
-v mongo_tls_enable="$RITA_MONGO_TLS" \
-v mongo_tls_cert_check="$RITA_MONGO_TLS_CHECK_CERT" \
-v mongo_tls_ca_path="$RITA_MONGO_TLS_CERT_PATH" '
# flag is used to determine if we are in the right scope

# Unset the flag if we see "abc:" or "  abc:" on a line
# by itself if there are 2 or less preceding spaces
/^(  )?[^ ]+:$/{
  flag=""
}

# Trigger the flag as we are entering the scope
/  RITA-MongoDB:/{
  flag=1
}

flag && NF && /ConnectionString:/{
  match($0,/^ +/);
  val=substr($0,RSTART,RLENGTH);
  $NF=mongo_uri;
  print val $0;
  next
}

flag && NF && /AuthenticationMechanism:/{
  match($0,/^ +/);
  val=substr($0,RSTART,RLENGTH);
  $NF=mongo_auth;
  print val $0;
  next
}

flag && NF && /Enable:/{
  match($0,/^ +/);
  val=substr($0,RSTART,RLENGTH);
  $NF=mongo_tls_enable;
  print val $0;
  next
}

flag && NF && /VerifyCertificate:/{
  match($0,/^ +/);
  val=substr($0,RSTART,RLENGTH);
  $NF=mongo_tls_cert_check;
  print val $0;
  next
}

flag && NF && /CAFile:/{
  match($0,/^ +/);
  val=substr($0,RSTART,RLENGTH);
  $NF=mongo_tls_ca_path;
  print val $0;
  next
}

flag && NF && /DBRoot:/{
  match($0,/^ +/);
  val=substr($0,RSTART,RLENGTH);
  $NF=db_root;
  print val $0;
  next
}

1
' $INSTALLATION_ETC_DIR/converter/converter.yaml > $INSTALLATION_ETC_DIR/converter/converter-new.yaml && \
	mv $INSTALLATION_ETC_DIR/converter/converter-new.yaml $INSTALLATION_ETC_DIR/converter/converter.yaml
} #End of write_converter_conf

# Stop if there are any errors
set -e
# Change dir to script dir
_OLD_DIR=$(pwd); cd "$(dirname "$BASH_SOURCE[0]")";

if [[ $EUID -ne 0 ]]; then
   echo "This script must be run with administrator privileges." >&2
   exit 1
fi

# Parse through command args to override values
echo "Loading command line params" >&2
if [ "$1" = 'help' -o "$1" = '--help' ]; then
	usage_text
	exit 1
fi

#SEARCHFORTHIS
while [ -n "$1" ]; do
	case "$1" in
	rita|Rita|RITA)
		if [ -n "$2" ] && ! echo "$2" | egrep -iq '(^rita$)' ; then
			rita_system="$2"
			echo "Got the rita_system"
			shift
		else
			rita_system='127.0.0.1'
		fi
		;;
	*)
		echo "Unrecognized parameter $1" >&2
		usage_text
		exit 1
		;;
	esac
	shift
done

#TODO check system requirements

# Install docker if needed
chmod +x "install-scripts/install_docker.sh"
install-scripts/install_docker.sh

# Set by make-release
INSTALLATION_DIR="/opt/ipfix-rita"
INSTALLATION_BIN_DIR="$INSTALLATION_DIR/bin"
INSTALLATION_LIB_DIR="$INSTALLATION_DIR/lib"
INSTALLATION_ETC_DIR="/etc/ipfix-rita"
DOCKER_IMAGES="docker-images.tgz"

#Should we open these at the end?
echo "Loading IPFIX-RITA Docker images... This may take a few minutes."  >&2
gzip -d -c ${DOCKER_IMAGES} | docker load

echo "Installing configuration files to $INSTALLATION_ETC_DIR"  >&2

SETUP_CONFIG="true"
if [ ! -d "$INSTALLATION_ETC_DIR" ]; then
  cp -r pkg/etc "$INSTALLATION_ETC_DIR"
else
  # TODO: set up migration
  echo "Existing configuration found. Skipping..."  >&2
  SETUP_CONFIG="false"
fi

echo "Installing ipfix-rita in $INSTALLATION_DIR"  >&2

if [ -d "$INSTALLATION_DIR" ]; then
  rm -rf "$INSTALLATION_DIR"
fi

mkdir -p "$INSTALLATION_DIR"

cp -r ./pkg/bin "$INSTALLATION_BIN_DIR"
chmod +x "$INSTALLATION_BIN_DIR/ipfix-rita"

cp -r ./pkg/lib "$INSTALLATION_LIB_DIR"

# set receive buffer size for logstash collector
RECV_BUFF_SIZE=$(sysctl -n net.core.rmem_max)
RECV_BUFF_OPT_SIZE="$((1024*1024*64))"
if [ "$RECV_BUFF_SIZE" -lt "$RECV_BUFF_OPT_SIZE" ]; then
  sysctl -w net.core.rmem_max=$RECV_BUFF_OPT_SIZE
  echo "net.core.rmem_max=$RECV_BUFF_OPT_SIZE" >> /etc/sysctl.conf  >&2
fi

"$INSTALLATION_BIN_DIR/ipfix-rita" up --no-start

if [ "$SETUP_CONFIG" = "true" ]; then
	#From here out we want to move to a interactive/active install command
	if [ -n "$rita_system" ]; then
		get_rita_data_noninteractive $rita_system
	else
		get_rita_data_interactive
	fi

#We have written to config
	echo ""  >&2
	echo "Your settings have been saved to $INSTALLATION_ETC_DIR/converter/converter.yaml" >&2
	echo "Note: By default IPFIX-RITA, considers all Class A, B, and C IPv4 networks" >&2
	echo "as local networks. If this is not the case, please edit the list 'LocalNetworks'" >&2
	echo "in $INSTALLATION_ETC_DIR/converter/converter.yaml." >&2
fi

echo "" >&2
echo "IPFIX-RITA will run at start up unless the system has been stopped." >&2
echo "In order to stop IPFIX-RITA, run 'ipfix-rita stop'." >&2
echo "To restart IPFIX-RITA, run 'ipfix-rita up -d'." >&2
echo "To view the system logs, run 'ipfix-rita logs -f'." >&2
echo ""

echo "Adding a symbolic link from /usr/local/bin/ipfix-rita to $INSTALLATION_BIN_DIR/ipfix-rita." >&2

ln -fs "$INSTALLATION_BIN_DIR/ipfix-rita" /usr/local/bin/ipfix-rita

echo "" >&2
echo "Starting IPFIX-RITA..." >&2

"$INSTALLATION_BIN_DIR/ipfix-rita" up -d

echo "The IPFIX-RITA installer has finished."

# Change back to the old directory at the end
cd $_OLD_DIR; unset _OLD_DIR
