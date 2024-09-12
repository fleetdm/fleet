FROM debian:stable-slim@sha256:90128f59a7c6f6fdcb6493f587ea352d5c7507f52a6ddfba66fc56cd3d99dc2b

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
    && mkdir /wix \
    && rm -rf /var/lib/apt/lists/* 

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