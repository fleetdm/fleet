FROM ubuntu:22.04

RUN apt-get -y update \
    && apt-get -y install build-essential intltool \
    libglib2.0-dev libtool-bin libgsf-1-dev gobject-introspection \
    valac libgcab-dev uuid-dev libxml2-dev \
    libmsi-dev gir1.2-libmsi-1.0 \
    valabind bison libglib2.0-0 wget \
    && wget https://download.gnome.org/sources/msitools/0.103/msitools-0.103.tar.xz -nv -O  msitools-0.103.tar.xz \
    && tar -xvf msitools-0.103.tar.xz \
    && cd msitools-0.103 \
    && ./autogen.sh && ./configure && make && make install 