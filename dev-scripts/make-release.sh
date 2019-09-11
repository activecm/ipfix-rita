#!/usr/bin/env bash

#This script creates a tgz file with the following structure  and contents.
#ipfix-rita
#├── docker-images.tgz
#├── docs
#|   ├── Additional Info.md
#│   ├── Developer Notes.md
#│   ├── Generating Data.md
#│   └── Router Support.md
#├── install_ipfix-rita.sh
#├── install-scripts
#│   ├── helpers
#│   │   ├── check_docker-compose.sh
#│   │   └── check_docker.sh
#│   └── install_docker.sh
#├── pkg
#│   ├── bin
#│   │   └── ipfix-rita
#│   ├── etc
#│   │   └── converter
#│   │       └── converter.yaml
#│   └── lib
#│       └── docker-compose
#│           ├── main.yaml
#│           ├── no-rotate.yaml
#│           └── xpack.yaml
#└── README.md
#

# Change dir to script dir
pushd "$(dirname "$BASH_SOURCE[0]")" > /dev/null

set -o errexit
set -o errtrace
set -o pipefail

export IPFIX_RITA_VERSION="$(cat ../VERSION)"

DOCKER_IMAGE_OUT="docker-images.tgz"
IPFIX_RITA_ARCHIVE="ipfix-rita"

IN_DEV_README="../README.md"
IN_DEV_DOCS_DIR="../docs"
IN_DEV_BIN_DIR="../runtime/bin"
IN_DEV_LIB_DIR="../runtime/lib"
IN_DEV_ETC_DIR="../runtime/etc"

IN_DEV_MAIN_SCRIPT="$IN_DEV_BIN_DIR/ipfix-rita"
IN_DEV_COMPOSE_DIR="$IN_DEV_LIB_DIR/docker-compose"
IN_DEV_MAIN_COMPOSE_FILE="$IN_DEV_COMPOSE_DIR/main.yaml"

################################################################################
echo "Building Docker images"
COMPOSE_FILE="$IN_DEV_MAIN_COMPOSE_FILE" docker-compose build
docker pull mongo:3.6

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
INSTALLER_DOCS_DIR="$INSTALLER_DIR/docs"
INSTALLER_BIN_DIR="$INSTALLER_PKG_DIR/bin"
INSTALLER_LIB_DIR="$INSTALLER_PKG_DIR/lib"
INSTALLER_ETC_DIR="$INSTALLER_PKG_DIR/etc"

INSTALLER_COMPOSE_DIR="$INSTALLER_LIB_DIR/docker-compose"
INSTALLER_MAIN_SCRIPT="$INSTALLER_BIN_DIR/ipfix-rita"

INSTALLER_SCRIPTS_DIR=$"../install-scripts"

INSTALLER_TARBALL="./$IPFIX_RITA_ARCHIVE.tgz"

# Final installation locations
INSTALLATION_DIR="/opt/ipfix-rita"
INSTALLATION_ETC_DIR="/etc/ipfix-rita"

echo "Creating installer tarball $INSTALLER_TARBALL..."

if [ -f "$INSTALLER_TARBALL" ]; then
    rm "$INSTALLER_TARBALL"
fi

# Insert the install scripts
# Start by adding the base install command
cp "$INSTALLER_SCRIPTS_DIR/install_ipfix-rita.sh" "$INSTALLER_DIR"
sed -i "s|INSTALLATION_DIR=\"REPLACE_WITH_INSTALL_DIR\"|INSTALLATION_DIR=\"$INSTALLATION_DIR\"|g" $INSTALLER_DIR/install_ipfix-rita.sh
sed -i "s|INSTALLATION_ETC_DIR=\"REPLACE_WITH_ETC_DIR\"|INSTALLATION_ETC_DIR=\"$INSTALLATION_ETC_DIR\"|g" $INSTALLER_DIR/install_ipfix-rita.sh
sed -i "s|DOCKER_IMAGES=\"./REPLACE_WITH_TARBALL\"|DOCKER_IMAGES=\"$DOCKER_IMAGE_OUT\"|g" $INSTALLER_DIR/install_ipfix-rita.sh
#Then add all the helper scirpts to the tarball and remove install-ipfix-rita.sh
cp -r "$INSTALLER_SCRIPTS_DIR" "$INSTALLER_DIR"
rm "$INSTALLER_DIR/install-scripts/install_ipfix-rita.sh"

# Copy in README
cp "$IN_DEV_README" "$INSTALLER_README"

# Copy in docs
cp -r "$IN_DEV_DOCS_DIR" "$INSTALLER_DOCS_DIR"

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
head -n 1  "$IN_DEV_MAIN_SCRIPT" >> "$INSTALLER_MAIN_SCRIPT"

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
git push origin master

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
