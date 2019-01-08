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

# Stop if there are any errors
set -e
# Change dir to script dir
_OLD_DIR=\$(pwd); cd "\$(dirname "\$BASH_SOURCE[0]")";

if [[ \$EUID -ne 0 ]]; then
   echo "This script must be run with administrator privileges."
   exit 1
fi

# Install docker if needed
chmod +x "scripts/install_docker.sh"
sudo scripts/install_docker.sh
require_file /usr/bin/docker				|| fail "Docker does not appear to have been installed successfully"
if sudo docker run hello-world 2>/dev/null | grep -iq 'Hello from Docker!' ; then
  status 'Docker appears to be working, continuing.'
else
  echo 'Docker does not appear to be able to pull down a docker instance and run it successfully.' >&2
  echo 'The most likely explanation is that this system is not able to make outbound connections to the Internet.' >&2
  echo 'We recommend you fix this, but will proceed anyways in 20 seconds.' >&2
  sleep 20
fi

# Ensure docker-compose is functional
#TODO If we install docker automatically intall docker-compose automatically too
if [ ! -x "\$(command -v docker-compose)" ]; then
  echo "'docker-compose' was not found in the PATH. Please install the latest"
  echo "version of docker-compose using the official instructions for your OS."
  exit 1
fi

DOCKER_COMPOSE_VERSION="\$(docker-compose -v | sed 's/^.* \([0-9][0-9]*\)\.\([0-9][0-9]*\)\.\([0-9][0-9]*\).*$/\1 \2 \3/')"
DOCKER_COMPOSE_VERSION_MAJOR="\$(echo \$DOCKER_COMPOSE_VERSION | cut -d' ' -f1)"
DOCKER_COMPOSE_VERSION_MINOR="\$(echo \$DOCKER_COMPOSE_VERSION | cut -d' ' -f2)"

MIN_DOCKER_COMPOSE_VERSION_MAJOR=1
MIN_DOCKER_COMPOSE_VERSION_MINOR=17

if [ "\$DOCKER_COMPOSE_VERSION_MAJOR" -lt "\$MIN_DOCKER_COMPOSE_VERSION_MAJOR" ] ||
[ "\$DOCKER_COMPOSE_VERSION_MAJOR" -eq "\$MIN_DOCKER_COMPOSE_VERSION_MAJOR" -a "\$DOCKER_COMPOSE_VERSION_MINOR" -lt "\$MIN_DOCKER_COMPOSE_VERSION_MINOR" ]; then
  echo "IPFIX-RITA requires docker-compose version \$MIN_DOCKER_COMPOSE_VERSION_MAJOR.\$MIN_DOCKER_COMPOSE_VERSION_MINOR+. Please upgrade to the latest"
  echo "version of docker-compose using the official instructions for your OS."
  exit 1
fi
#end of TODO's

# Set by make-release
INSTALLATION_DIR="$INSTALLATION_DIR"
INSTALLATION_BIN_DIR="$INSTALLATION_DIR/bin"
INSTALLATION_LIB_DIR="$INSTALLATION_DIR/lib"
INSTALLATION_ETC_DIR="$INSTALLATION_ETC_DIR"
DOCKER_IMAGES="./$DOCKER_IMAGE_OUT"

echo "Loading IPFIX-RITA Docker images... This may take a few minutes."
gzip -d -c \${DOCKER_IMAGES} | docker load

echo "Installing configuration files to \$INSTALLATION_ETC_DIR"

SETUP_CONFIG="true"
if [ ! -d "\$INSTALLATION_ETC_DIR" ]; then
  cp -r pkg/etc "\$INSTALLATION_ETC_DIR"
else
  # TODO: set up migration
  echo "Existing configuration found. Skipping..."
  SETUP_CONFIG="false"
fi

echo "Installing ipfix-rita in \$INSTALLATION_DIR"

if [ -d "\$INSTALLATION_DIR" ]; then
  rm -rf "\$INSTALLATION_DIR"
fi

mkdir -p "\$INSTALLATION_DIR"

cp -r ./pkg/bin "\$INSTALLATION_BIN_DIR"
chmod +x "\$INSTALLATION_BIN_DIR/ipfix-rita"

cp -r ./pkg/lib "\$INSTALLATION_LIB_DIR"

# set receive buffer size for logstash collector
RECV_BUFF_SIZE=\$(sysctl -n net.core.rmem_max)
RECV_BUFF_OPT_SIZE="\$((1024*1024*64))"
if [ "\$RECV_BUFF_SIZE" -lt "\$RECV_BUFF_OPT_SIZE" ]; then
  sysctl -w net.core.rmem_max=\$RECV_BUFF_OPT_SIZE
  echo "net.core.rmem_max=\$RECV_BUFF_OPT_SIZE" >> /etc/sysctl.conf
fi

"\$INSTALLATION_BIN_DIR/ipfix-rita" up --no-start

if [ "\$SETUP_CONFIG" = "true" ]; then
  IPFIX_RITA_NETWORK_GATEWAY=\$(docker inspect ipfix_rita_default --format "{{with (index .IPAM.Config 0)}}{{.Gateway}}{{end}}")
  RITA_MONGO_URI="mongodb://\$IPFIX_RITA_NETWORK_GATEWAY:27017"

  echo ""
  echo "IPFIX-RITA needs to write to a MongoDB database controlled by RITA."
  echo "By default, this installer assumes RITA and MongoDB are installed on the Docker host."
  echo "In order to support this type of installation, you will need to"
  echo "add the suggested Docker interface below to the list of bindIP's in /etc/mongod.conf."
  echo "If needed, please do so, and restart MongoDB before continuing."
  echo "Restart MongoDB with \$ sudo systemctl restart mongod.service"
  echo "Note: the default configuration is not recommended. IPFIX-RITA will likely perform"
  echo "better if it is installed on a machine separate from RITA/ MongoDB."
  echo ""
  read -p "What MongoDB URI should IPFIX-RITA use to contact the RITA database [\$RITA_MONGO_URI]: " -r
  if [ -n "\$REPLY" ]; then
    RITA_MONGO_URI="\$REPLY"
  fi

  RITA_MONGO_AUTH="null"

  echo ""
  echo "Which authentication scheme should be used to contact the database if any? [None]"
  echo "1) None"
  echo "2) SCRAM-SHA-1"
  echo "3) MONGODB-CR"

  while read && [[ ! ( "\$REPLY" =~ ^[123]\$ || -z "\$REPLY" ) ]]; do
    echo "Which authentication scheme should be used to contact the database if any? [None]"
    echo "1) None"
    echo "2) SCRAM-SHA-1"
    echo "3) MONGODB-CR"
  done

  if [ "\$REPLY" = "2" ]; then
    RITA_MONGO_AUTH="SCRAM-SHA-1"
  elif [ "\$REPLY" = "3" ]; then
    RITA_MONGO_AUTH="MONGODB-CR"
  fi

  RITA_MONGO_TLS="false"
  RITA_MONGO_TLS_CHECK_CERT="false"
  RITA_MONGO_TLS_CERT_PATH="null"
  echo ""
  read -p "Does the MongoDB server accept TLS connections? (y/n) [n] "  -r
  if [[ "\$REPLY" =~ ^[Yy]\$ ]]; then
    RITA_MONGO_TLS="true"
    RITA_MONGO_TLS_CHECK_CERT="true"
    RITA_MONGO_TLS_CERT_PATH="null"
    read -p "Would you like to provide a certificate authority? (y/n) [n] "  -r
    if [[ "\$REPLY" =~ ^[Yy]\$ ]]; then
      read -p "CA Path: "
      RITA_MONGO_TLS_CERT_PATH="\$REPLY"
    fi

    if [ "\$RITA_MONGO_TLS_CERT_PATH" = "null" ]; then
      read -p "Would you like to disable certificate checks? [n] "  -r
      if [[ "\$REPLY" =~ ^[Yy]\$ ]]; then
        RITA_MONGO_TLS_CHECK_CERT="false"
      fi
    fi
  fi

  echo ""
  echo "Each dataset produced with IPFIX-RITA will be named DBROOT-DATE"
  echo "where DBROOT consists of alphanumerics, underscores, and hyphens."
  RITA_DATASET_DBROOT="IPFIX"
  read -p "What would you like to set DBROOT to for this IPFIX collector? [IPFIX] " -r
  if [ -n "\$REPLY" ]; then
    RITA_DATASET_DBROOT="\$REPLY"
  fi

  #unindent to ensure nothing breaks with awk

awk -v db_root="\$RITA_DATASET_DBROOT" \\
-v mongo_uri="\$RITA_MONGO_URI" \\
-v mongo_auth="\$RITA_MONGO_AUTH" \\
-v mongo_tls_enable="\$RITA_MONGO_TLS" \\
-v mongo_tls_cert_check="\$RITA_MONGO_TLS_CHECK_CERT" \\
-v mongo_tls_ca_path="\$RITA_MONGO_TLS_CERT_PATH" '
# flag is used to determine if we are in the right scope

# Unset the flag if we see "abc:" or "  abc:" on a line
# by itself if there are 2 or less preceding spaces
/^(  )?[^ ]+:\$/{
  flag=""
}

# Trigger the flag as we are entering the scope
/  RITA-MongoDB:/{
  flag=1
}

flag && NF && /ConnectionString:/{
  match(\$0,/^ +/);
  val=substr(\$0,RSTART,RLENGTH);
  \$NF=mongo_uri;
  print val \$0;
  next
}

flag && NF && /AuthenticationMechanism:/{
  match(\$0,/^ +/);
  val=substr(\$0,RSTART,RLENGTH);
  \$NF=mongo_auth;
  print val \$0;
  next
}

flag && NF && /Enable:/{
  match(\$0,/^ +/);
  val=substr(\$0,RSTART,RLENGTH);
  \$NF=mongo_tls_enable;
  print val \$0;
  next
}

flag && NF && /VerifyCertificate:/{
  match(\$0,/^ +/);
  val=substr(\$0,RSTART,RLENGTH);
  \$NF=mongo_tls_cert_check;
  print val \$0;
  next
}

flag && NF && /CAFile:/{
  match(\$0,/^ +/);
  val=substr(\$0,RSTART,RLENGTH);
  \$NF=mongo_tls_ca_path;
  print val \$0;
  next
}

flag && NF && /DBRoot:/{
  match(\$0,/^ +/);
  val=substr(\$0,RSTART,RLENGTH);
  \$NF=db_root;
  print val \$0;
  next
}

1
' \$INSTALLATION_ETC_DIR/converter/converter.yaml > \$INSTALLATION_ETC_DIR/converter/converter-new.yaml && \\
mv \$INSTALLATION_ETC_DIR/converter/converter-new.yaml \$INSTALLATION_ETC_DIR/converter/converter.yaml

echo ""
echo "Your settings have been saved to \$INSTALLATION_ETC_DIR/converter/converter.yaml"
echo "Note: By default IPFIX-RITA, considers all Class A, B, and C IPv4 networks"
echo "as local networks. If this is not the case, please edit the list 'LocalNetworks'"
echo "in \$INSTALLATION_ETC_DIR/converter/converter.yaml."
fi

echo ""
echo "IPFIX-RITA will run at start up unless the system has been stopped."
echo "In order to stop IPFIX-RITA, run 'ipfix-rita stop'."
echo "To restart IPFIX-RITA, run 'ipfix-rita up -d'."
echo "To view the system logs, run 'ipfix-rita logs -f'."
echo ""

echo "Adding a symbolic link from /usr/local/bin/ipfix-rita to \$INSTALLATION_BIN_DIR/ipfix-rita."

ln -fs "\$INSTALLATION_BIN_DIR/ipfix-rita" /usr/local/bin/ipfix-rita

echo ""
echo "Starting IPFIX-RITA..."

"\$INSTALLATION_BIN_DIR/ipfix-rita" up -d

echo "The IPFIX-RITA installer has finished."

# Change back to the old directory at the end
cd \$_OLD_DIR; unset _OLD_DIR
