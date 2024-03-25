ARG FLEET_IMAGE

FROM ${FLEET_IMAGE}

ARG LICENSE_KEY
USER root

RUN mkdir -p /opt/GeoLite2 && cd /opt/GeoLite2 &&\
    wget --max-redirect 10 "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=${LICENSE_KEY}&suffix=tar.gz" -O GeoLite2-City.tar.gz &&\
    wget --max-redirect 10 "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=${LICENSE_KEY}&suffix=tar.gz.sha256" -O GeoLite2-City.tar.gz.sha256 &&\
    [ "$(awk '{ print $1 }' GeoLite2-City.tar.gz.sha256)" == "$(sha256sum GeoLite2-City.tar.gz | awk '{ print $1 }')" ] &&\
    (tar -xzvf GeoLite2-City.tar.gz "*/GeoLite2-City.mmdb" --strip-components 1 2>/dev/null || true) &&\
    rm -f GeoLite2-City.tar.gz*

USER fleet
# Might not be needed again, but keep it just in case
CMD ["fleet", "serve"]
