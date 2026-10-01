- Added a "Retrying" status to a host's OS settings for Android certificates that Fleet is
  automatically retrying after a failed install. The status shows the error reported by the host
  and which attempt Fleet is on. Previously these certificates showed as "Enforcing" with no sign
  that an attempt had failed.
- Fixed an Android certificate up for renewal reporting the error from its previous install. The
  renewal now starts clean, so a host's OS settings no longer show an old error against a renewal
  that is proceeding normally.
