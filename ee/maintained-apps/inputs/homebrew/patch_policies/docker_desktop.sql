SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = 'com.electron.dockerdesktop' AND path NOT LIKE '%.back' AND version_compare(bundle_short_version, '__VERSION__') < 0);
