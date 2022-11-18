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
    _make_path,
    SKIP_IN_PATH,
)


class MigrationClient(NamespacedClient):
    @query_params("allow_no_indices", "expand_wildcards", "ignore_unavailable")
    def get_assistance(self, index=None, params=None):
        """
        `<https://www.elastic.co/guide/en/elasticsearch/reference/current/migration-api-assistance.html>`_

        :arg index: A comma-separated list of index names; use `_all` or empty
            string to perform the operation on all indices
        :arg allow_no_indices: Whether to ignore if a wildcard indices
            expression resolves into no concrete indices. (This includes `_all`
            string or when no indices have been specified)
        :arg expand_wildcards: Whether to expand wildcard expression to concrete
            indices that are open, closed or both., default 'open', valid
            choices are: 'open', 'closed', 'none', 'all'
        :arg ignore_unavailable: Whether specified concrete indices should be
            ignored when unavailable (missing or closed)
        """
        return self.transport.perform_request(
            "GET", _make_path("_xpack", "migration", "assistance", index), params=params
        )

    @query_params("wait_for_completion")
    def upgrade(self, index, params=None):
        """

        `<https://www.elastic.co/guide/en/elasticsearch/reference/current/migration-api-upgrade.html>`_

        :arg index: The name of the index
        :arg wait_for_completion: Should the request block until the upgrade
            operation is completed, default True
        """
        if index in SKIP_IN_PATH:
            raise ValueError("Empty value passed for a required argument 'index'.")
        return self.transport.perform_request(
            "POST", _make_path("_xpack", "migration", "upgrade", index), params=params
        )
