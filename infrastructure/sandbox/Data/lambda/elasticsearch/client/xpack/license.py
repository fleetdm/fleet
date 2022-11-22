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

from ..utils import (
    NamespacedClient,
    query_params,
)


class LicenseClient(NamespacedClient):
    @query_params()
    def delete(self, params=None):
        """

        `<https://www.elastic.co/guide/en/x-pack/current/license-management.html>`_
        """
        return self.transport.perform_request(
            "DELETE", "/_xpack/license", params=params
        )

    @query_params("local")
    def get(self, params=None):
        """

        `<https://www.elastic.co/guide/en/x-pack/current/license-management.html>`_

        :arg local: Return local information, do not retrieve the state from
            master node (default: false)
        """
        return self.transport.perform_request("GET", "/_xpack/license", params=params)

    @query_params("acknowledge")
    def post(self, body=None, params=None):
        """

        `<https://www.elastic.co/guide/en/x-pack/current/license-management.html>`_

        :arg body: licenses to be installed
        :arg acknowledge: whether the user has acknowledged acknowledge messages
            (default: false)
        """
        return self.transport.perform_request(
            "PUT", "/_xpack/license", params=params, body=body
        )
