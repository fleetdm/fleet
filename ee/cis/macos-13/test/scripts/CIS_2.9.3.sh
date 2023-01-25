#!/bin/bash

if [[ $(uname -m) == 'arm64' ]]; then
	# Apple silicon
	/usr/bin/sudo /usr/bin/pmset -a standby 900
	/usr/bin/sudo /usr/bin/pmset -a destroyfvkeyonstandby 1
	/usr/bin/sudo /usr/bin/pmset -a hibernatemode 25
else
	# Intel
	/usr/bin/sudo /usr/bin/pmset -a standbydelaylow 900
	/usr/bin/sudo /usr/bin/pmset -a standbydelayhigh 900
	/usr/bin/sudo /usr/bin/pmset -a highstandbythreshold 90
	/usr/bin/sudo /usr/bin/pmset -a destroyfvkeyonstandby 1
	/usr/bin/sudo /usr/bin/pmset -a hibernatemode 25
fi
