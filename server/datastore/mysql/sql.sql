SELECT
  hsi.host_id AS host_id,
  hsi.execution_id AS execution_id,
  hsi.software_installer_id AS installer_id,
  si.pre_install_query AS pre_install_condition,
  is.contents AS install_script,
  pis.contents AS post_install_script
FROM
  host_software_installs hsi
INNER JOIN
  software_installers si
  ON hsi.software_installer_id = si.id
LEFT JOIN
  script_contents is
  ON is.id = si.install_script_content_id
LEFT JOIN
  script_contents pis
  ON pis.id = si.post_install_script_content_id
WHERE
  hsi.host_id = ?
AND
  hsi.install_script_exit_code IS NOT NULL
