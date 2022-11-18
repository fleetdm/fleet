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

try:
    import queue
except ImportError:
    import Queue as queue
from .base import Connection


class PoolingConnection(Connection):
    """
    Base connection class for connections that use libraries without thread
    safety and no capacity for connection pooling. To use this just implement a
    ``_make_connection`` method that constructs a new connection and returns
    it.
    """

    def __init__(self, *args, **kwargs):
        self._free_connections = queue.Queue()
        super(PoolingConnection, self).__init__(*args, **kwargs)

    def _get_connection(self):
        try:
            return self._free_connections.get_nowait()
        except queue.Empty:
            return self._make_connection()

    def _release_connection(self, con):
        self._free_connections.put(con)

    def close(self):
        """
        Explicitly close connection
        """
        pass
