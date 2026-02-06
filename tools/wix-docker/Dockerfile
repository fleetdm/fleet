FROM debian:bookworm-slim@sha256:56ff6d36d4eb3db13a741b342ec466f121480b5edded42e4b7ee850ce7a418ee

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
    && apt upgrade -y \
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
    && wget https://github.com/wixtoolset/wix3/releases/download/wix3141rtm/wix314-binaries.zip -nv -O wix.zip \
    && mkdir wix \
    && unzip wix.zip -d wix \
    && rm -f wix.zip \
    && /home/wine/make-aliases.sh \
    && rm -f /home/wine/make-aliases.sh \
    && mkdir $WINEPREFIX/drive_c/temp

WORKDIR /wix
