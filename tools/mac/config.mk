# Kolide hostname. Make sure omit https:// or the path
KOLIDE_HOSTNAME=kolide.acme.co

# Osquery enroll secret. Replace with the secret set in Kolide.
ENROLL_SECRET=CHANGEME

# Paste your kolide certificate chain below.
define KOLIDE_TLS_CERTIFICATE
CHANGEME
endef

# Osquery flag file. No need to modify.
define KOLIDE_FLAGS
--force=true
--host_identifier=instance
--verbose=true
--debug
--tls_dump=true

--tls_hostname=$(KOLIDE_HOSTNAME)
--tls_server_certs=/etc/osquery/kolide.crt
--enroll_secret_path=/etc/osquery/kolide_secret

--enroll_tls_endpoint=/api/v1/osquery/enroll

--config_plugin=tls
--config_tls_endpoint=/api/v1/osquery/config
--config_refresh=10

--disable_distributed=false
--distributed_plugin=tls
--distributed_interval=10
--distributed_tls_max_attempts=3
--distributed_tls_read_endpoint=/api/v1/osquery/distributed/read
--distributed_tls_write_endpoint=/api/v1/osquery/distributed/write

--logger_plugin=tls
--logger_tls_endpoint=/api/v1/osquery/log
--logger_tls_period=10
endef
