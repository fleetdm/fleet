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
import warnings

from .base import Connection
from ..exceptions import (
    ConnectionError,
    ImproperlyConfigured,
    ConnectionTimeout,
    SSLError,
)
from ..compat import urlencode, string_types
from ..utils import _client_meta_version

try:
    import requests

    REQUESTS_AVAILABLE = True
    _REQUESTS_META_VERSION = _client_meta_version(requests.__version__)
except ImportError:
    REQUESTS_AVAILABLE = False
    _REQUESTS_META_VERSION = ""


class RequestsHttpConnection(Connection):
    """
    Connection using the `requests` library.

    :arg http_auth: optional http auth information as either ':' separated
        string or a tuple. Any value will be passed into requests as `auth`.
    :arg use_ssl: use ssl for the connection if `True`
    :arg verify_certs: whether to verify SSL certificates
    :arg ca_certs: optional path to CA bundle. By default standard requests'
        bundle will be used.
    :arg client_cert: path to the file containing the private key and the
        certificate, or cert only if using client_key
    :arg client_key: path to the file containing the private key if using
        separate cert and key files (client_cert will contain only the cert)
    :arg headers: any custom http headers to be add to requests
    :arg http_compress: Use gzip compression
    :arg cloud_id: The Cloud ID from ElasticCloud. Convenient way to connect to cloud instances.
        Other host connection params will be ignored.
    """

    HTTP_CLIENT_META = ("rq", _REQUESTS_META_VERSION)

    def __init__(
        self,
        host="localhost",
        port=None,
        http_auth=None,
        use_ssl=False,
        verify_certs=True,
        ca_certs=None,
        client_cert=None,
        client_key=None,
        headers=None,
        http_compress=None,
        cloud_id=None,
        **kwargs
    ):
        if not REQUESTS_AVAILABLE:
            raise ImproperlyConfigured(
                "Please install requests to use RequestsHttpConnection."
            )

        # Initialize Session so .headers works before calling super().__init__().
        self.session = requests.Session()
        for key in list(self.session.headers):
            self.session.headers.pop(key)

        super(RequestsHttpConnection, self).__init__(
            host=host,
            port=port,
            use_ssl=use_ssl,
            headers=headers,
            http_compress=http_compress,
            cloud_id=cloud_id,
            **kwargs
        )

        if not self.http_compress:
            # Need to set this to 'None' otherwise Requests adds its own.
            self.session.headers["accept-encoding"] = None

        if http_auth is not None:
            if isinstance(http_auth, (tuple, list)):
                http_auth = tuple(http_auth)
            elif isinstance(http_auth, string_types):
                http_auth = tuple(http_auth.split(":", 1))
            self.session.auth = http_auth

        self.base_url = "%s%s" % (
            self.host,
            self.url_prefix,
        )
        self.session.verify = verify_certs
        if not client_key:
            self.session.cert = client_cert
        elif client_cert:
            # cert is a tuple of (certfile, keyfile)
            self.session.cert = (client_cert, client_key)
        if ca_certs:
            if not verify_certs:
                raise ImproperlyConfigured(
                    "You cannot pass CA certificates when verify SSL is off."
                )
            self.session.verify = ca_certs

        if self.use_ssl and not verify_certs:
            warnings.warn(
                "Connecting to %s using SSL with verify_certs=False is insecure."
                % self.host
            )

    def perform_request(
        self, method, url, params=None, body=None, timeout=None, ignore=(), headers=None
    ):
        url = self.base_url + url
        headers = headers or {}
        if params:
            url = "%s?%s" % (url, urlencode(params))

        orig_body = body
        if self.http_compress and body:
            body = self._gzip_compress(body)
            headers["content-encoding"] = "gzip"

        start = time.time()
        request = requests.Request(method=method, headers=headers, url=url, data=body)
        prepared_request = self.session.prepare_request(request)
        settings = self.session.merge_environment_settings(
            prepared_request.url, {}, None, None, None
        )
        send_kwargs = {"timeout": timeout or self.timeout}
        send_kwargs.update(settings)
        try:
            response = self.session.send(prepared_request, **send_kwargs)
            duration = time.time() - start
            raw_data = response.content.decode("utf-8", "surrogatepass")
        except Exception as e:
            self.log_request_fail(
                method,
                url,
                prepared_request.path_url,
                body,
                time.time() - start,
                exception=e,
            )
            if isinstance(e, requests.exceptions.SSLError):
                raise SSLError("N/A", str(e), e)
            if isinstance(e, requests.Timeout):
                raise ConnectionTimeout("TIMEOUT", str(e), e)
            raise ConnectionError("N/A", str(e), e)

        # raise errors based on http status codes, let the client handle those if needed
        if (
            not (200 <= response.status_code < 300)
            and response.status_code not in ignore
        ):
            self.log_request_fail(
                method,
                url,
                response.request.path_url,
                orig_body,
                duration,
                response.status_code,
                raw_data,
            )
            self._raise_error(response.status_code, raw_data)

        self.log_request_success(
            method,
            url,
            response.request.path_url,
            orig_body,
            response.status_code,
            raw_data,
            duration,
        )

        return response.status_code, response.headers, raw_data

    @property
    def headers(self):
        return self.session.headers

    def close(self):
        """
        Explicitly closes connections
        """
        self.session.close()
