#!/bin/bash
# run createrepo to re-generate metadata
createrepo --update /repo

# sign repo with GPG key
gpg --default-key 000CF27C --detach-sign --armor /repo/repodata/repomd.xml
