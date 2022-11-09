#  Licensed to Elasticsearch B.V. under one or more contributor
#  license agreements. See the NOTICE file distributed with
#  this work for additional information regarding copyright
#  ownership. Elasticsearch B.V. licenses this file to you under
#  the Apache License, Version 2.0 (the "License"); you may
#  not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
# 	http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing,
#  software distributed under the License is distributed on an
#  "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
#  KIND, either express or implied.  See the License for the
#  specific language governing permissions and limitations
#  under the License.

from ..utils import NamespacedClient, query_params

from .graph import GraphClient
from .license import LicenseClient
from .monitoring import MonitoringClient
from .security import SecurityClient
from .watcher import WatcherClient
from .ml import MlClient
from .migration import MigrationClient
from .deprecation import DeprecationClient


class XPackClient(NamespacedClient):
    namespace = "xpack"

    def __init__(self, *args, **kwargs):
        super(XPackClient, self).__init__(*args, **kwargs)
        self.graph = GraphClient(self.client)
        self.license = LicenseClient(self.client)
        self.monitoring = MonitoringClient(self.client)
        self.security = SecurityClient(self.client)
        self.watcher = WatcherClient(self.client)
        self.ml = MlClient(self.client)
        self.migration = MigrationClient(self.client)
        self.deprecation = DeprecationClient(self.client)

    @query_params("categories", "human")
    def info(self, params=None):
        """
        Retrieve information about xpack, including build number/timestamp and license status
        `<https://www.elastic.co/guide/en/elasticsearch/reference/current/info-api.html>`_

        :arg categories: Comma-separated list of info categories. Can be any of:
            build, license, features
        :arg human: Presents additional info for humans (feature descriptions
            and X-Pack tagline)
        """
        return self.transport.perform_request("GET", "/_xpack", params=params)

    @query_params("master_timeout")
    def usage(self, params=None):
        """
        Retrieve information about xpack features usage

        :arg master_timeout: Specify timeout for watch write operation
        """
        return self.transport.perform_request("GET", "/_xpack/usage", params=params)
