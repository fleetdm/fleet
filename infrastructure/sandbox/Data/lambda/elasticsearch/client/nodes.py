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

from .utils import NamespacedClient, query_params, _make_path


class NodesClient(NamespacedClient):
    @query_params("flat_settings", "timeout")
    def info(self, node_id=None, metric=None, params=None):
        """
        The cluster nodes info API allows to retrieve one or more (or all) of
        the cluster nodes information.
        `<https://www.elastic.co/guide/en/elasticsearch/reference/current/cluster-nodes-info.html>`_

        :arg node_id: A comma-separated list of node IDs or names to limit the
            returned information; use `_local` to return information from the
            node you're connecting to, leave empty to get information from all
            nodes
        :arg metric: A comma-separated list of metrics you wish returned. Leave
            empty to return all.
        :arg flat_settings: Return settings in flat format (default: false)
        :arg timeout: Explicit operation timeout
        """
        return self.transport.perform_request(
            "GET", _make_path("_nodes", node_id, metric), params=params
        )

    @query_params(
        "completion_fields",
        "fielddata_fields",
        "fields",
        "groups",
        "include_segment_file_sizes",
        "level",
        "timeout",
        "types",
    )
    def stats(self, node_id=None, metric=None, index_metric=None, params=None):
        """
        The cluster nodes stats API allows to retrieve one or more (or all) of
        the cluster nodes statistics.
        `<https://www.elastic.co/guide/en/elasticsearch/reference/current/cluster-nodes-stats.html>`_

        :arg node_id: A comma-separated list of node IDs or names to limit the
            returned information; use `_local` to return information from the
            node you're connecting to, leave empty to get information from all
            nodes
        :arg metric: Limit the information returned to the specified metrics
        :arg index_metric: Limit the information returned for `indices` metric
            to the specific index metrics. Isn't used if `indices` (or `all`)
            metric isn't specified.
        :arg completion_fields: A comma-separated list of fields for `fielddata`
            and `suggest` index metric (supports wildcards)
        :arg fielddata_fields: A comma-separated list of fields for `fielddata`
            index metric (supports wildcards)
        :arg fields: A comma-separated list of fields for `fielddata` and
            `completion` index metric (supports wildcards)
        :arg groups: A comma-separated list of search groups for `search` index
            metric
        :arg include_segment_file_sizes: Whether to report the aggregated disk
            usage of each one of the Lucene index files (only applies if segment
            stats are requested), default False
        :arg level: Return indices stats aggregated at index, node or shard
            level, default 'node', valid choices are: 'indices', 'node',
            'shards'
        :arg timeout: Explicit operation timeout
        :arg types: A comma-separated list of document types for the `indexing`
            index metric
        """
        return self.transport.perform_request(
            "GET",
            _make_path("_nodes", node_id, "stats", metric, index_metric),
            params=params,
        )

    @query_params(
        "type", "ignore_idle_threads", "interval", "snapshots", "threads", "timeout"
    )
    def hot_threads(self, node_id=None, params=None):
        """
        An API allowing to get the current hot threads on each node in the cluster.
        `<https://www.elastic.co/guide/en/elasticsearch/reference/current/cluster-nodes-hot-threads.html>`_

        :arg node_id: A comma-separated list of node IDs or names to limit the
            returned information; use `_local` to return information from the
            node you're connecting to, leave empty to get information from all
            nodes
        :arg type: The type to sample (default: cpu), valid choices are:
            'cpu', 'wait', 'block'
        :arg ignore_idle_threads: Don't show threads that are in known-idle
            places, such as waiting on a socket select or pulling from an empty
            task queue (default: true)
        :arg interval: The interval for the second sampling of threads
        :arg snapshots: Number of samples of thread stacktrace (default: 10)
        :arg threads: Specify the number of threads to provide information for
            (default: 3)
        :arg timeout: Explicit operation timeout
        """
        # avoid python reserved words
        if params and "type_" in params:
            params["type"] = params.pop("type_")
        return self.transport.perform_request(
            "GET", _make_path("_cluster", "nodes", node_id, "hotthreads"), params=params
        )

    @query_params("human", "timeout")
    def usage(self, node_id=None, metric=None, params=None):
        """
        The cluster nodes usage API allows to retrieve information on the usage
        of features for each node.
        `<http://www.elastic.co/guide/en/elasticsearch/reference/master/cluster-nodes-usage.html>`_

        :arg node_id: A comma-separated list of node IDs or names to limit the
            returned information; use `_local` to return information from the
            node you're connecting to, leave empty to get information from all
            nodes
        :arg metric: Limit the information returned to the specified metrics
        :arg human: Whether to return time and byte values in human-readable
            format., default False
        :arg timeout: Explicit operation timeout
        """
        return self.transport.perform_request(
            "GET", _make_path("_nodes", node_id, "usage", metric), params=params
        )

    @query_params("timeout")
    def reload_secure_settings(self, node_id=None, params=None):
        """
        `<https://www.elastic.co/guide/en/elasticsearch/reference/6.x/secure-settings.html#reloadable-secure-settings>`_

        :arg node_id: A comma-separated list of node IDs to span the
            reload/reinit call. Should stay empty because reloading usually
            involves all cluster nodes.
        :arg timeout: Explicit operation timeout
        """
        return self.transport.perform_request(
            "POST",
            _make_path("_nodes", node_id, "reload_secure_settings"),
            params=params,
        )
