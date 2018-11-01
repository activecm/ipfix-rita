#!/usr/bin/env bash

# Change dir to script dir
pushd "$(dirname "$BASH_SOURCE[0]")" > /dev/null

set -o errexit
set -o errtrace
set -o pipefail

export IPFIX_RITA_VERSION="$(cat ../VERSION)"
IPFIX_RITA_VERSION_HYPHENATED="$(echo $IPFIX_RITA_VERSION | sed 's/\./-/g')"

DOCKER_IMAGE_OUT="docker-images.tgz"
IPFIX_RITA_ARCHIVE="ipfix-rita-$IPFIX_RITA_VERSION_HYPHENATED"

IN_DEV_README="../README.md"
IN_DEV_BIN_DIR="../runtime/bin"
IN_DEV_LIB_DIR="../runtime/lib"
IN_DEV_ETC_DIR="../runtime/etc"

IN_DEV_MAIN_SCRIPT="$IN_DEV_BIN_DIR/ipfix-rita"
IN_DEV_COMPOSE_DIR="$IN_DEV_LIB_DIR/docker-compose"
IN_DEV_MAIN_COMPOSE_FILE="$IN_DEV_COMPOSE_DIR/main.yaml"

################################################################################
echo "Building Docker images"
COMPOSE_FILE="$IN_DEV_MAIN_COMPOSE_FILE" docker-compose build

################################################################################
TMP_DIR=`mktemp -d -q "/tmp/IPFIX-RITA.XXXXXXXX" </dev/null`
if [ ! -d "$TMP_DIR" ]; then
  echo "Unable to create temporary directory."
  exit 1
else
  INSTALLER_DIR="$TMP_DIR/$IPFIX_RITA_ARCHIVE"
  INSTALLER_PKG_DIR="$INSTALLER_DIR/pkg"
  mkdir -p "$INSTALLER_PKG_DIR"
  echo "Building installer in temporary directory: $INSTALLER_DIR"
fi

INSTALLER_README="$INSTALLER_DIR/README.md"
INSTALLER_BIN_DIR="$INSTALLER_PKG_DIR/bin"
INSTALLER_LIB_DIR="$INSTALLER_PKG_DIR/lib"
INSTALLER_ETC_DIR="$INSTALLER_PKG_DIR/etc"

INSTALLER_COMPOSE_DIR="$INSTALLER_LIB_DIR/docker-compose"
INSTALLER_MAIN_SCRIPT="$INSTALLER_BIN_DIR/ipfix-rita"

INSTALLER_INSTALL_SCRIPT="$INSTALLER_DIR/install-ipfix-rita.sh"

INSTALLER_TARBALL="./$IPFIX_RITA_ARCHIVE.tgz"

# Final installation locations
INSTALLATION_DIR="/opt/ipfix-rita"
INSTALLATION_ETC_DIR="/etc/ipfix-rita"

echo "Creating installer tarball $INSTALLER_TARBALL..."

if [ -f "$INSTALLER_TARBALL" ]; then
    rm "$INSTALLER_TARBALL"
fi

# Copy in README
cp "$IN_DEV_README" "$INSTALLER_README"

# Copy over the etc files
cp -r "$IN_DEV_ETC_DIR" "$INSTALLER_ETC_DIR"

# Get rid of the build statements in the docker-compose files
# and add them to the install tarball
mkdir "$INSTALLER_LIB_DIR"
mkdir "$INSTALLER_COMPOSE_DIR"
for file in "$IN_DEV_COMPOSE_DIR"/*.yaml; do
  grep -v '^\s*build' $file > "$INSTALLER_COMPOSE_DIR/$(basename $file)"
done

mkdir "$INSTALLER_BIN_DIR"
# Construct the main script
# Copy the shebang line in
head  "$IN_DEV_MAIN_SCRIPT" >> "$INSTALLER_MAIN_SCRIPT"

# Ensure the right versions of the images are ran
echo "export IPFIX_RITA_VERSION=$IPFIX_RITA_VERSION" >> "$INSTALLER_MAIN_SCRIPT"

# Copy the rest of the main script in
tail -n +2 "$IN_DEV_MAIN_SCRIPT" >> "$INSTALLER_MAIN_SCRIPT"

# Copy in the docker images
docker save \
  "mongo:3.6" \
  "quay.io/activecm/ipfix-rita-converter:$IPFIX_RITA_VERSION" \
  "quay.io/activecm/ipfix-rita-logstash:$IPFIX_RITA_VERSION" \
  | gzip -c - > "$INSTALLER_DIR/$DOCKER_IMAGE_OUT"

# Insert the install script
cat << EOF > $INSTALLER_INSTALL_SCRIPT
#!/usr/bin/env bash
# Stop if there are any errors
set -e
# Change dir to script dir
_OLD_DIR=\$(pwd); cd "\$(dirname "\$BASH_SOURCE[0]")";

if [[ \$EUID -ne 0 ]]; then
   echo "This script must be run with administrator privileges."
   exit 1
fi

# Ensure docker is functional
if [ ! -x "\$(command -v docker)" ]; then
  echo "'docker' was not found in the PATH. Please install the latest"
  echo "version of Docker using the official instructions for your OS."
	exit 1
fi

DOCKER_VERSION="\$(docker -v | sed 's/^.* \([0-9][0-9]*\)\.\([0-9][0-9]*\)\.\([0-9][0-9]*\).*$/\1 \2 \3/')"
DOCKER_VERSION_MAJOR="\$(echo \$DOCKER_VERSION | cut -d' ' -f1)"
DOCKER_VERSION_MINOR="\$(echo \$DOCKER_VERSION | cut -d' ' -f2)"

MIN_DOCKER_VERSION_MAJOR=17
MIN_DOCKER_VERSION_MINOR=06

if [ "\$DOCKER_VERSION_MAJOR" -lt "\$MIN_DOCKER_VERSION_MAJOR" ] ||
[ "\$DOCKER_VERSION_MAJOR" -eq "\$MIN_DOCKER_VERSION_MAJOR" -a "\$DOCKER_VERSION_MINOR" -lt "\$MIN_DOCKER_VERSION_MINOR" ]; then
  echo "IPFIX_RITA requires Docker version \$MIN_DOCKER_VERSION_MAJOR.\$MIN_DOCKER_VERSION_MINOR+. Please upgrade to the latest"
  echo "version of Docker using the official instructions for your OS."
  exit 1
fi

# Ensure docker-compose is functional
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

# Unset the flag if we see "abc:" on a line
# by itself if there are 2 or less preceding spaces
/^ {0,2}[^ ]*:\$/{
  flag=""
}

# Trigger the flag as we are entering the scope
/  RITA-MongoDB:/{
  flag=1
}

flag && NF && /ConnectionString:/{
  match(\$0,/^[[:space:]]+/);
  val=substr(\$0,RSTART,RLENGTH);
  \$NF=mongo_uri;
  print val \$0;
  next
}

flag && NF && /AuthenticationMechanism:/{
  match(\$0,/^[[:space:]]+/);
  val=substr(\$0,RSTART,RLENGTH);
  \$NF=mongo_auth;
  print val \$0;
  next
}

flag && NF && /Enable:/{
  match(\$0,/^[[:space:]]+/);
  val=substr(\$0,RSTART,RLENGTH);
  \$NF=mongo_tls_enable;
  print val \$0;
  next
}

flag && NF && /VerifyCertificate:/{
  match(\$0,/^[[:space:]]+/);
  val=substr(\$0,RSTART,RLENGTH);
  \$NF=mongo_tls_cert_check;
  print val \$0;
  next
}

flag && NF && /CAFile:/{
  match(\$0,/^[[:space:]]+/);
  val=substr(\$0,RSTART,RLENGTH);
  \$NF=mongo_tls_ca_path;
  print val \$0;
  next
}

flag && NF && /DBRoot:/{
  match(\$0,/^[[:space:]]+/);
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
EOF

chmod +x "$INSTALLER_INSTALL_SCRIPT"

tar -C $TMP_DIR -czf $INSTALLER_TARBALL $IPFIX_RITA_ARCHIVE
################################################################################

echo "Please ensure you are on the master branch and that all"
echo "local changes have been committed and pushed before proceeding."

read -p "Tag this release and publish the Docker images? (y/n) "  -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    exit 0
fi

GIT_BRANCH_NAME=$(git symbolic-ref -q HEAD)
GIT_BRANCH_NAME=${GIT_BRANCH_NAME##refs/heads/}
GIT_BRANCH_NAME=${GIT_BRANCH_NAME:-HEAD}
if [[ $GIT_BRANCH_NAME != "master" ]]; then
    echo "Ensure the current Git branch is master."
    exit 1
fi

if ! git diff-index --quiet HEAD -- ; then
    echo "Ensure all local changes are committed and pushed before proceeding."
    exit 1
fi

echo "Creating a tagged version for the current Git commit."
git tag -a "$IPFIX_RITA_VERSION" -m "version $IPFIX_RITA_VERSION"
git push --tags

echo "Tagging version $IPFIX_RITA_VERSION as latest"
# Set the latest tag as docker doesn't do that itself
docker tag quay.io/activecm/ipfix-rita-converter:$IPFIX_RITA_VERSION quay.io/activecm/ipfix-rita-converter:latest
docker tag quay.io/activecm/ipfix-rita-logstash:$IPFIX_RITA_VERSION quay.io/activecm/ipfix-rita-logstash:latest

echo "Pushing to quay.io"
docker login quay.io
docker push quay.io/activecm/ipfix-rita-converter:latest
docker push quay.io/activecm/ipfix-rita-converter:$IPFIX_RITA_VERSION

docker push quay.io/activecm/ipfix-rita-logstash:latest
docker push quay.io/activecm/ipfix-rita-logstash:$IPFIX_RITA_VERSION

echo "The current git commit has been tagged as $IPFIX_RITA_VERSION and quay.io has been updated. Please publish $INSTALLER_TARBALL on the GitHub releases page."

# Change back to original directory
popd > /dev/null
