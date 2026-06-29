#!/bin/bash

/usr/bin/sudo /usr/sbin/chown -R root:wheel /etc/security/audit_control
/usr/bin/sudo /bin/chmod -R o-rw /etc/security/audit_control
/usr/bin/sudo /usr/sbin/chown -R root:wheel /var/audit/
/usr/bin/sudo /bin/chmod -R o-rw /var/audit/

