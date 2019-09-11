#!/bin/bash

# Change dir to script dir
pushd "$(dirname "$BASH_SOURCE[0]")" > /dev/null

# Options and Usage
# -----------------------------------
usage() {
	scriptName=$(basename "$0")
	echo -n "${scriptName} [OPTION]...
Install needed docker code to support IPFIX-RITA.
Options:
  -g, --group-add       Add the current user to the 'docker' group
  -r, --replace-shell	(Implies -g) When finished, replaces the current shell
                        so the current user can control docker immediately.
                        This will prevent any calling scripts from executing further.
  -h, --help            Display this help and exit
"
}

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

ADD_DOCKER_GROUP=false

# Parse through command args to override values
while [[ $# -gt 0 ]]; do
	case $1 in
		-g|--group-add)
			ADD_DOCKER_GROUP=true
			;;
		-r|--replace-shell)
			ADD_DOCKER_GROUP=true
			REPLACE_SHELL=true
			;;
		-h|--help)
			usage >&2
			exit
			;;
		*)
			;;
	esac
	shift
done

helpers/check_docker.sh
# Store the exit code
DOCKER_CHECK=$?
if [ "$DOCKER_CHECK" -gt 3 ]; then
	# This may overwrite a file maintained by a package.
	echo "An unsupported version of Docker appears to already be installed. It will be replaced."
fi
if [ "$DOCKER_CHECK" -eq 0 ]; then
	echo "Docker appears to already be installed. Skipping."
elif [ -s /etc/redhat-release ] && grep -iq 'release 7' /etc/redhat-release ; then
	#This configuration file is used in both Redhat RHEL and Centos distributions, so we're running under RHEL/Centos 7.x
	# https://docs.docker.com/engine/installation/linux/docker-ce/centos/

	sudo yum makecache fast

	if rpm -q docker >/dev/null 2>&1 || rpm -q docker-common >/dev/null 2>&1 || rpm -q docker-selinux >/dev/null 2>&1 || rpm -q docker-engine >/dev/null 2>&1 ; then
		echo -n "One or more of these packages are installed: docker, docker-common, docker-selinux, and/or docker-engine .  The docker website encourages us to remove these before installing docker-ce.  Would you like to remove these older packages (recommended: yes)"
		if askYN ; then
			sudo yum -y remove docker docker-common docker-selinux docker-engine
		else
			echo "You chose not to remove the older docker packages.  The install may not succeed."
		fi
	fi

	sudo yum install -y yum-utils device-mapper-persistent-data lvm2 shadow-utils

	sudo yum-config-manager --enable extras

	if [ ! -f /etc/yum.repos.d/docker-ce.repo ]; then
		sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
	fi

	sudo wget -q https://download.docker.com/linux/centos/gpg -O ~/DOCKER-GPG-KEY
	sudo rpm --import ~/DOCKER-GPG-KEY

	sudo yum -y install docker-ce

	sudo systemctl start docker
	sudo systemctl enable docker
elif grep -iq '^DISTRIB_ID *= *Ubuntu' /etc/lsb-release ; then
	### Install Docker on Ubuntu ###
	# https://docs.docker.com/engine/installation/linux/docker-ce/ubuntu/#install-using-the-repository

	echo "Installing Docker package repo..."
	sudo apt-get update
	sudo apt-get install -y \
		apt-transport-https \
		ca-certificates \
		curl \
		software-properties-common

	curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -

	sudo add-apt-repository \
		"deb [arch=amd64] https://download.docker.com/linux/ubuntu \
		$(lsb_release -cs) \
		stable"

	echo "Installing latest Docker version..."
	sudo apt-get update
	sudo apt-get install -y docker-ce
	sudo service docker start
else
	echo "This system does not appear to be a Centos 7.x, RHEL 7.x, or Ubuntu Linux system.  Unable to install docker."
	exit 1
fi

helpers/check_docker-compose.sh
# Store the exit code
DOCKER_COMPOSE_CHECK=$?
if [ "$DOCKER_COMPOSE_CHECK" -gt 3 ]; then
	# This may overwrite a file maintained by a package.
	echo "An unsupported version of Docker-Compose appears to already be installed. It will be replaced."
fi
if [ "$DOCKER_COMPOSE_CHECK" -eq 0 ]; then
	echo "Docker-Compose appears to already be installed. Skipping."
else
	### Install Docker-Compose ###
	# https://docs.docker.com/compose/install/#install-compose
	DOCKER_COMPOSE_VERSION="1.21.2"

	echo "Installing Docker-Compose v${DOCKER_COMPOSE_VERSION}..."
	sudo curl -L https://github.com/docker/compose/releases/download/${DOCKER_COMPOSE_VERSION}/docker-compose-`uname -s`-`uname -m` -o /usr/bin/docker-compose
	sudo chmod +x /usr/bin/docker-compose
fi

if [ "${ADD_DOCKER_GROUP}" = "true" ]; then
	# Add current user to docker group
	echo "Adding current user to docker group..."
	#sudo groupadd docker
	sudo usermod -aG docker $USER

	if [ "${REPLACE_SHELL}" = "true" ]; then
		echo "Docker installation complete. You should have access to the 'docker' and 'docker-compose' commands immediately."
		# Hack to activate the docker group on the current user without logging out.
		# Downside is it completely replaces the shell and prevents calling scripts from continuing.
		# https://superuser.com/a/853897
		exec sg docker newgrp `id -gn`
	fi

	echo "You will need to login again for these changes to take effect."
	echo "Docker installation complete. You should have access to the 'docker' and 'docker-compose' commands once you log out and back in."
else
	echo "Docker installation complete. 'docker' and 'docker-compose' must be run using sudo or the root account."
fi

# Change back to original directory
popd > /dev/null
