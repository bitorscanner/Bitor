#!/bin/bash

# Color definitions for tput
BLACK=0
RED=1
GREEN=2
YELLOW=3
BLUE=4
CYAN=6
RESET=$(tput sgr0)
TEXT_COLOR="tput setaf "
BACKGROUND_COLOR="tput setab "
CLEAR_UP="#tput cuu 1; tput ed;"

version_regex="[[:digit:]]*\.[[:digit:]]*\.[[:digit:]]"
VERSION_REGEX="[[:digit:]]*\.[[:digit:]]*\.[[:digit:]]"
is_latest_version="yes"

function print_welcome() {
  echo -e "$($TEXT_COLOR $CYAN)
\t  ____  ___       ___ _   _ ____ _____  _    _     _     _____ ____  
\t / ___|/ _ \     |_ _| \ | / ___|_   _|/ \  | |   | |   | ____|  _ \ 
\t| |  _| | | |_____| ||  \| \___ \ | | / _ \ | |   | |   |  _| | |_) |
\t| |_| | |_| |_____| || |\  |___) || |/ ___ \| |___| |___| |___|  _ < 
\t \____|\___/     |___|_| \_|____/ |_/_/   \_\_____|_____|_____|_| \_\\
\t ${RESET}"
}

function print_help() {
  echo -e "\t$($TEXT_COLOR $BLUE)go.sh${RESET} is a tool that helps you easily install, update or uninstall Go\n
  \t$($TEXT_COLOR $GREEN)-------------------------------  Usage  -------------------------------\n
  \t$($TEXT_COLOR $YELLOW)bash go.sh${RESET}\t\t\tInstalls or update Go (if installed)
  \t$($TEXT_COLOR $YELLOW)bash go.sh --version [version]${RESET}\tInstalls a specific version of Go
  \t$($TEXT_COLOR $YELLOW)bash go.sh remove${RESET}\t\tUninstalls the installed version of Go
  \t$($TEXT_COLOR $YELLOW)bash go.sh help${RESET}\t\t\tPrints this help message
  "
}

function what_platform() {
  os="$(uname -s)"
  arch="$(uname -m)"

  case $os in
  "Linux")
    case $arch in
    "x86_64")
      arch=amd64
      ;;
    "armv6")
      arch=armv6l
      ;;
    "armv8" | "aarch64")
      arch=arm64
      ;;
    .*386.*)
      arch=386
      ;;
    esac
    platform="linux-$arch"
    ;;
  "Darwin")
    case $arch in
    "x86_64")
      arch=amd64
      ;;
    "arm64")
      arch=arm64
      ;;
    esac
    platform="darwin-$arch"
    ;;
  esac
}

function what_shell_profile() {
  if [ -n "$($SHELL -c 'echo $ZSH_VERSION')" ]; then
    shell_profile="zshrc"
  elif [ -n "$($SHELL -c 'echo $BASH_VERSION')" ]; then
    shell_profile="bashrc"
  fi
}

function what_installed_version() {
  INSTALLED_VERSION=$(go version)
}

function extract_version_from() {
  local version
  version=$(grep -o "$VERSION_REGEX" <<<"$1")
  echo "$version"
}

function get_download_command() {
  if command -v curl &>/dev/null; then
    echo "curl -s"
  elif command -v wget &>/dev/null; then
    echo "wget -qO-"
  else
    echo "$($TEXT_COLOR $RED)Error: Neither curl nor wget is available. Please install one of them.${RESET}" >&2
    exit 1
  fi
}

function find_version_link() {
  file_name="go$version_regex.$platform.tar.gz"
  link_regex="dl/$file_name"
  go_website="https://go.dev/"

  download_command=$(get_download_command)

  latest_version_link="$go_website$(
    $download_command "$go_website/dl/" | # get the HTML of golang page
      grep -o "$link_regex" |             # select installation links
      head -1                             # only get the first link i.e.(latest version)
  )"

  latest_version_file_name=$(grep -o "$file_name" <<<"$latest_version_link")
  [[ -z $latest_version_file_name ]] && echo "$($TEXT_COLOR $RED)Couldn't find $file_name on $go_website${RESET}" && exit 1
}

function go_exists() {
  go version &>/dev/null
}

function remove() {
  if ! go_exists; then
    echo "$($TEXT_COLOR $RED)Go is not installed!${RESET}"
    exit
  fi

  what_shell_profile
  what_installed_version
  echo "$($TEXT_COLOR $RED)removing $INSTALLED_VERSION${RESET} from ${GOROOT}"

  if ! rm -r -f "$GOROOT"; then
    echo "$($TEXT_COLOR $RED)Couldn't remove Go${RESET}."
    echo "Can't remove contents of $GOROOT"
    echo "Maybe you need to run the script with root privileges!"
    echo "sudo bash go.sh"
    exit 1
  fi

  RC_PROFILE="$HOME/.${shell_profile}"

  echo "Creating a backup of your ${RC_PROFILE} to ${RC_PROFILE}-BACKUP"
  cp "$RC_PROFILE" "${RC_PROFILE}-BACKUP"
  echo "Removing exports for GOROOT & GOPATH from ${RC_PROFILE}"
  sed -i'' -e '/export GOROOT/d' "${RC_PROFILE}"
  sed -i'' -e '/:$GOROOT/d' "${RC_PROFILE}"
  sed -i'' -e '/export GOPATH/d' "${RC_PROFILE}"
  sed -i'' -e '/:$GOPATH/d' "${RC_PROFILE}"

  echo "$($TEXT_COLOR $GREEN)Uninstalled Go Successfully!${RESET}"
}

function test_installation() {

  if [ $? -ne 0 ]; then
    echo "$($TEXT_COLOR $RED)Installation failed!!${RESET}"
    exit 1
  fi

  echo "$($TEXT_COLOR $CYAN)Go${RESET} ($VERSION) has been installed $($TEXT_COLOR $GREEN)successfully!${RESET}"
  echo "Open a new terminal(to re login) or you can do: $($TEXT_COLOR $YELLOW)source $HOME/.${shell_profile}${RESET}"
}

function install_go() {

  eval "$CLEAR_UP"

  VERSION=$(extract_version_from "$latest_version_link")
  version_name="latest version"
  [[ $is_latest_version == "no" ]] && version_name="version"
  echo "Downloading $($TEXT_COLOR $CYAN)Go${RESET} $version_name ($(
    $BACKGROUND_COLOR $BLACK
    tput smul
  )$VERSION${RESET})..."

  download_command=$(get_download_command)

  if [[ $download_command == "curl -s" ]]; then
    if ! curl -fSL --progress-bar "$latest_version_link" -o "$latest_version_file_name"; then
      echo "$($TEXT_COLOR $RED)Download failed!${RESET}"
      exit 1
    fi
  else
    # wget2 v2.1.0 changed --show-progress to --force-progress, so we need to check which one to use
    progress_arg="--show-progress"
    wget --help | grep -q -- --force-progress && progress_arg="--force-progress"

    if ! wget --quiet --continue $progress_arg "$latest_version_link"; then
      echo "$($TEXT_COLOR $RED)Download failed!${RESET}"
      exit 1
    fi
  fi

  [ -z "$GOROOT" ] && GOROOT="$HOME/.go"
  [ -z "$GOPATH" ] && GOPATH="$HOME/go"

  eval "$CLEAR_UP"

  mkdir -p "$GOPATH"/{src,pkg,bin} "$GOROOT"

  echo "Extracting $latest_version_file_name files to $GOROOT..."

  tar -xzf "$latest_version_file_name"

  mv go/* "$GOROOT"
  rmdir go

  what_shell_profile

  touch "$HOME/.${shell_profile}"
  {
    echo "export GOROOT=$GOROOT"
    echo "export GOPATH=$GOPATH"
    echo 'export PATH=$PATH:$GOROOT/bin:$GOPATH/bin'
  } >>"$HOME/.${shell_profile}"

  eval "$CLEAR_UP"
}

function echo_finding() {
  finding="Finding latest version"
  [[ $is_latest_version == "no" ]] && finding="You chose to install version $version_regex"
  echo "$finding of $($TEXT_COLOR $CYAN)Go${RESET} for $($TEXT_COLOR $YELLOW)$platform${RESET}..."
}

function update_go() {
  GOPATH=$(go env GOPATH)
  GOROOT=$(go env GOROOT)
  what_installed_version
  latest=$(extract_version_from "$latest_version_link")
  current=$(extract_version_from "$INSTALLED_VERSION")

  eval "$CLEAR_UP"
  echo -e "          VERSION"
  echo -e "CURRENT:   $current"
  echo -e "CHOSEN:    $latest"

  if [[ $current == "$latest" ]]; then
    echo "You already have that version of $($TEXT_COLOR $CYAN)Go${RESET} Installed!"
    echo "$($TEXT_COLOR $BLUE)Exiting, Bye!${RESET}"
    exit
  fi

  echo "Installing will remove the current installed version from '$GOROOT'"

  if [[ $1 == "update" ]]; then
    # update is used to force update for testing on travis
    # bypass read option
    option=""
  else
    echo -e "Do you want to install $($TEXT_COLOR $GREEN)Go($latest)${RESET} and remove $($TEXT_COLOR $RED)Go($current)${RESET}? [ENTER(yes)/n]: \c"
    read -r option
  fi

  case $option in
  "" | Y* | y*)
    remove && install_go
    ;;
  N* | n*)
    echo "Okay, Bye!"
    exit 0
    ;;
  *)
    echo "Wrong choice!"
    exit 1
    ;;
  esac

}

function remove_downloaded_package() {
  rm -f "$latest_version_file_name"
}

function main() {
  print_welcome

  if [[ $# == 1 ]]; then
    case $1 in
    "update")
      # do nothing, continue execution normally
      ;;
    "remove")
      remove
      exit
      ;;
    *)
      print_help
      exit
      ;;
    esac
  elif [[ $# == 2 ]]; then
    case $1 in
    "--version")
      version_regex=$2
      is_latest_version="no"
      ;;
    *)
      print_help
      exit
      ;;
    esac
  elif [[ $# > 2 ]]; then
    print_help
    exit
  fi

  what_platform
  echo_finding
  find_version_link

  if go_exists -eq 0; then
    echo "Go exists"
    update_go "$1"
  else
    install_go
  fi

  test_installation
  remove_downloaded_package
}

main "$@"
