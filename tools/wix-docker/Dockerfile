FROM debian:bullseye-slim

ARG wine_uid
ARG wine_gid

RUN true \
    && dpkg --add-architecture i386 \
    && apt update \
    && apt install -y --no-install-recommends \
    ca-certificates \
    wine \
    wine32 \
    wget \
    unzip \
    osslsigncode \
    # Create a separate user for Wine
    && if [ -n "${wine_gid}" ] ; \
    then addgroup --system wine -g ${wine_gid} ; \
    else addgroup --system wine ; fi \
    && if [ -n "${wine_uid}" ] ; \
    then \
    adduser \
    --home /home/wine \
    --disabled-password \
    --shell /bin/bash \
    --gecos "non-root user for Wine" \
    --ingroup wine \
    --u ${wine_uid} \
    wine ; \
    else \
    adduser \
    --home /home/wine \
    --disabled-password \
    --shell /bin/bash \
    --gecos "non-root user for Wine" \
    --ingroup wine \
    wine ;\
    fi \
    && mkdir /wix \
    && chown wine:wine /wix \
    && rm -rf /var/lib/apt/lists/* 

USER wine
WORKDIR /home/wine
ENV HOME=/home/wine WINEPREFIX=/home/wine/.wine WINEARCH=win32 PATH="/home/wine/bin:$PATH" WINEDEBUG=-all

COPY make-aliases.sh /home/wine/make-aliases.sh

# Install .NET framework and WiX Toolset binaries
RUN wine wineboot && \
    wget https://dl.winehq.org/wine/wine-mono/6.4.0/wine-mono-6.4.0-x86.msi -nv -O mono.msi \
    && wine msiexec /i mono.msi \
    && rm -f mono.msi \
    && wget https://github.com/wixtoolset/wix3/releases/download/wix3112rtm/wix311-binaries.zip -nv -O wix.zip \
    && mkdir wix \
    && unzip wix.zip -d wix \
    && rm -f wix.zip \
    && /home/wine/make-aliases.sh \
    && rm -f /home/wine/make-aliases.sh \
    && mkdir $WINEPREFIX/drive_c/temp

WORKDIR /wix