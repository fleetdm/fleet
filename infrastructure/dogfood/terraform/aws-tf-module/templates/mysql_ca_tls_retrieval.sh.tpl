#!/bin/bash
apk add coreutils openssl
        
wget --quiet  https://truststore.pki.rds.amazonaws.com/${aws_region}/${aws_region}-bundle.pem -O ${aws_region}-bundle.dl.pem
csplit -z -k -f cert. -b '%02d.pem' ${aws_region}-bundle.dl.pem '/-----BEGIN CERTIFICATE-----/' '{*}'

for filename in cert.*;
do 
  thumbprint=$(openssl x509 -in $${filename} -noout -fingerprint | cut -c 18- | sed 's/\://g' | awk '{print tolower($0)}')
  if [[ "${ca_cert_thumbprint}" = "$${thumbprint}" ]];
  then
    mv $${filename} ${container_path}/${aws_region}.pem
  fi 
done
