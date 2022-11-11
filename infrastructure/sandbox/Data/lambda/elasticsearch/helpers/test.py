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

import time
import os

try:
    # python 2.6
    from unittest2 import TestCase, SkipTest
except ImportError:
    from unittest import TestCase, SkipTest

from elasticsearch import Elasticsearch
from elasticsearch.exceptions import ConnectionError


def get_test_client(nowait=False, **kwargs):
    # construct kwargs from the environment
    kw = {"timeout": 5}
    if "TEST_ES_CONNECTION" in os.environ:
        from elasticsearch import connection

        kw["connection_class"] = getattr(connection, os.environ["TEST_ES_CONNECTION"])

    kw.update(kwargs)
    client = Elasticsearch([os.environ.get("TEST_ES_SERVER", {})], **kw)

    # wait for yellow status
    for _ in range(1 if nowait else 1):
        try:
            client.cluster.health(wait_for_status="yellow")
            return client
        except ConnectionError:
            time.sleep(0.1)
    else:
        # timeout
        raise SkipTest("Elasticsearch failed to start.")


def _get_version(version_string):
    if "." not in version_string:
        return ()
    version = version_string.strip().split(".")
    return tuple(int(v) if v.isdigit() else 999 for v in version)


class ElasticsearchTestCase(TestCase):
    @staticmethod
    def _get_client():
        return get_test_client()

    @classmethod
    def setUpClass(cls):
        super(ElasticsearchTestCase, cls).setUpClass()
        cls.client = cls._get_client()

    def tearDown(self):
        super(ElasticsearchTestCase, self).tearDown()
        self.client.indices.delete(index="*", ignore=404)
        self.client.indices.delete_template(name="*", ignore=404)

    @property
    def es_version(self):
        if not hasattr(self, "_es_version"):
            version_string = self.client.info()["version"]["number"]
            self._es_version = _get_version(version_string)
        return self._es_version
