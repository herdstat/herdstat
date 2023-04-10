#!/usr/bin/env bash

# Copyright The Helm Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# The install script is based off of the MIT-licensed script from glide,
# the package manager for Go: https://github.com/Masterminds/glide.sh/blob/master/get

# Modifications copyright (c) 2023 The Herdstat Authors

: ${BINARY_NAME:="herdstat"}
: ${USE_SUDO:="true"}
: ${DEBUG:="false"}
: ${HERDSTAT_INSTALL_DIR:="/usr/local/bin"}

HAS_CURL="$(type "curl" &> /dev/null && echo true || echo false)"
HAS_WGET="$(type "wget" &> /dev/null && echo true || echo false)"

# initArch discovers the architecture for this system.
initArch() {
  ARCH=$(uname -m)
  case $ARCH in
    armv5*) ARCH="armv5";;
    armv6*) ARCH="armv6";;
    armv7*) ARCH="arm";;
    aarch64) ARCH="arm64";;
    x86) ARCH="386";;
    x86_64) ARCH="amd64";;
    i686) ARCH="386";;
    i386) ARCH="386";;
  esac
}

# initOS discovers the operating system for this system.
initOS() {
  OS=$(echo `uname`|tr '[:upper:]' '[:lower:]')
  case "$OS" in
    # Minimalist GNU for Windows
    mingw*|cygwin*) OS='windows';;
  esac
}

# runs the given command as root (detects if we are root already)
runAsRoot() {
  if [ $EUID -ne 0 -a "$USE_SUDO" = "true" ]; then
    sudo "${@}"
  else
    "${@}"
  fi
}

# verifySupported checks that the os/arch combination is supported for
# binary builds, as well whether or not necessary tools are present.
verifySupported() {
  local supported="darwin-amd64\ndarwin-arm64\nlinux-amd64\nlinux-arm64\nwindows-amd64"
  if ! echo "${supported}" | grep -q "${OS}-${ARCH}"; then
    echo "No prebuilt binary for ${OS}-${ARCH}."
    echo "To build from source, go to https://github.com/herdstat/herdstat"
    exit 1
  fi

  if [ "${HAS_CURL}" != "true" ] && [ "${HAS_WGET}" != "true" ]; then
    echo "Either curl or wget is required"
    exit 1
  fi
}

# checkDesiredVersion checks if the desired version is available.
checkDesiredVersion() {
  if [ "x$DESIRED_VERSION" == "x" ]; then
    # Get tag from release URL
    local latest_release_url="https://github.com/herdstat/herdstat/releases"
    if [ "${HAS_CURL}" == "true" ]; then
      TAG=$(curl -Ls $latest_release_url | grep 'href="/herdstat/herdstat/releases/tag/v[0-9]*.[0-9]*.[0-9]*\"' | sed -E 's/.*\/herdstat\/herdstat\/releases\/tag\/(v[0-9\.]+)".*/\1/g' | head -1)
    elif [ "${HAS_WGET}" == "true" ]; then
      TAG=$(wget $latest_release_url -O - 2>&1 | grep 'href="/herdstat/herdstat/releases/tag/v[0-9]*.[0-9]*.[0-9]*\"' | sed -E 's/.*\/herdstat\/herdstat\/releases\/tag\/(v[0-9\.]+)".*/\1/g' | head -1)
    fi
  else
    TAG=$DESIRED_VERSION
  fi
}

# checkHerdstatInstalledVersion checks which version of herdstat is installed and
# if it needs to be changed.
checkHerdstatInstalledVersion() {
  if [[ -f "${HERDSTAT_INSTALL_DIR}/${BINARY_NAME}" ]]; then
    local version=$("${HERDSTAT_INSTALL_DIR}/${BINARY_NAME}" version -o=short)
    if [[ "$version" == "$TAG" ]]; then
      echo "Herdstat ${version} is already ${DESIRED_VERSION:-latest}"
      return 0
    else
      echo "Herdstat ${TAG} is available. Changing from version ${version}."
      return 1
    fi
  else
    return 1
  fi
}

# downloadFile downloads the latest binary package and also the checksum
# for that binary.
downloadFile() {
  HERDSTAT_DIST="herdstat-$TAG-$OS-$ARCH.tar.gz"
  DOWNLOAD_URL="https://github.com/herdstat/herdstat/releases/download/$TAG/$HERDSTAT_DIST"
  HERDSTAT_TMP_ROOT="$(mktemp -dt herdstat-installer-$TAG-$OS-$ARCH)"
  HERDSTAT_TMP_FILE="$HERDSTAT_TMP_ROOT/$HERDSTAT_DIST"
  echo "Downloading $DOWNLOAD_URL"
  if [ "${HAS_CURL}" == "true" ]; then
    curl -SsL "$DOWNLOAD_URL" -o "$HERDSTAT_TMP_FILE"
  elif [ "${HAS_WGET}" == "true" ]; then
    wget -q -O "$HERDSTAT_TMP_FILE" "$DOWNLOAD_URL"
  fi
}

# installFile installs the Herdstat binary.
installFile() {
  HERDSTAT_TMP="$HERDSTAT_TMP_ROOT/$BINARY_NAME"
  mkdir -p "$HERDSTAT_TMP"
  tar xf "$HERDSTAT_TMP_FILE" -C "$HERDSTAT_TMP"
  HERDSTAT_TMP_BIN="$HERDSTAT_TMP/herdstat"
  echo "Preparing to install $BINARY_NAME into ${HERDSTAT_INSTALL_DIR}"
  runAsRoot cp "$HERDSTAT_TMP_BIN" "$HERDSTAT_INSTALL_DIR/$BINARY_NAME"
  echo "$BINARY_NAME installed into $HERDSTAT_INSTALL_DIR/$BINARY_NAME"
}

# fail_trap is executed if an error occurs.
fail_trap() {
  result=$?
  if [ "$result" != "0" ]; then
    if [[ -n "$INPUT_ARGUMENTS" ]]; then
      echo "Failed to install $BINARY_NAME with the arguments provided: $INPUT_ARGUMENTS"
      help
    else
      echo "Failed to install $BINARY_NAME"
    fi
    echo -e "\tFor support, go to https://github.com/herdstat/herdstat."
  fi
  cleanup
  exit $result
}

# testVersion tests the installed client to make sure it is working.
testVersion() {
  set +e
  HERDSTAT="$(command -v $BINARY_NAME)"
  if [ "$?" = "1" ]; then
    echo "$BINARY_NAME not found. Is $HERDSTAT_INSTALL_DIR on your "'$PATH?'
    exit 1
  fi
  set -e
}

# help provides possible installation arguments
help () {
  echo "Accepted arguments are:"
  echo -e "  [--help|-h ] \t\t\t prints this help"
  echo -e "  [--version|-v <version>] \t the version to install, e.g., 'v0.9.2'; will install latest release from GitHub if not defined"
  echo -e "  [--no-sudo] \t\t\t install without sudo"
}

# cleanup temporary files
cleanup() {
  if [[ -d "${HERDSTAT_TMP_ROOT:-}" ]]; then
    rm -rf "$HERDSTAT_TMP_ROOT"
  fi
}

# Execution

#Stop execution on any error
trap "fail_trap" EXIT
set -e

# Set debug if desired
if [ "${DEBUG}" == "true" ]; then
  set -x
fi

# Parsing input arguments (if any)
export INPUT_ARGUMENTS="${@}"
set -u
while [[ $# -gt 0 ]]; do
  case $1 in
    '--version'|-v)
       shift
       if [[ $# -ne 0 ]]; then
           export DESIRED_VERSION="${1}"
       else
           echo -e "Please provide the desired version. e.g. --version v0.9.2"
           exit 0
       fi
       ;;
    '--no-sudo')
       USE_SUDO="false"
       ;;
    '--help'|-h)
       help
       exit 0
       ;;
    *) exit 1
       ;;
  esac
  shift
done
set +u

initArch
initOS
verifySupported
checkDesiredVersion
if ! checkHerdstatInstalledVersion; then
  downloadFile
  installFile
fi
testVersion
cleanup
