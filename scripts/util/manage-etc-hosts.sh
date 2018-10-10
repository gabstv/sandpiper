#!/bin/sh

# https://gist.github.com/irazasyed/a7b0a079e7727a4315b9

set -e

# PATH TO YOUR HOSTS FILE
ETC_HOSTS=/etc/hosts
# DEFAULT IP FOR HOSTNAME
IP="127.0.0.1"

HOSTNAME=""

function removehost () {
    if [[ -n "$(grep $HOSTNAME /etc/hosts)" ]]; then
        echo "$HOSTNAME Found in your $ETC_HOSTS, Removing now...";
        sudo sed -i".bak" "/$HOSTNAME/d" $ETC_HOSTS
    else
        echo "$HOSTNAME was not found in your $ETC_HOSTS";
    fi
}

function addhost () {
    HOSTS_LINE="$IP\t$HOSTNAME"
    if [[ -n "$(grep $HOSTNAME /etc/hosts)" ]]; then
        echo "$HOSTNAME already exists : $(grep $HOSTNAME $ETC_HOSTS)"
    else
        echo "Adding $HOSTNAME to your $ETC_HOSTS";
        sudo -- sh -c -e "echo '$HOSTS_LINE' >> /etc/hosts";
        if [[ -n "$(grep $HOSTNAME /etc/hosts)" ]]; then
            echo "$HOSTNAME was added succesfully \n $(grep $HOSTNAME /etc/hosts)";
        else
            echo "Failed to Add $HOSTNAME!"
            exit 1
        fi
    fi
}

function hostexists () {
    if [[ -n "$(grep $HOSTNAME /etc/hosts)" ]]; then
        echo "1"
    else
        echo "0"
    fi
}

# Deal with command line flags.
while [[ $# > 0 ]]
do
case "${1}" in
  -e|--exists)
  HOSTNAME=$2
  hostexists
  shift
  shift
  ;;
  -a|--add)
  HOSTNAME=$2
  addhost
  shift
  shift
  ;;
  -d|--delete)
  HOSTNAME=$2
  addhost
  shift
  shift
  ;;
  *)
  echo "${1} is not a valid flag, try running: ${0} --help"
  ;;
esac
done

exit 0