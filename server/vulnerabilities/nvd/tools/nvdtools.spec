# Copyright (c) Facebook, Inc. and its affiliates. All Rights Reserved

Name: nvdtools
Summary: A collection of tools for working with National Vulnerability Database feeds.
Version: %{_version}
Release: 1
License: Apache License 2.0
URL: https://github.com/facebookincubator/nvdtools
Source0: %{name}-%{version}.tar.gz

%description
A set of tools to work with the feeds (vulnerabilities, CPE dictionary etc.) distributed by National Vulnerability Database (NVD).

%prep
%setup -q

%build
make GOFLAGS="-ldflags=-linkmode=external"

%install
make install DESTDIR=$RPM_BUILD_ROOT

%files
%license LICENSE
%{_bindir}/*
/usr/share/doc/nvdtools

%changelog
