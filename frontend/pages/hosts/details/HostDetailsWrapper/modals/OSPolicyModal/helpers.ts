/**
 * parseOsVersion accepts an `os_version` string (e.g., "Ubuntu 16.04.7" or "CentOS 8.3.2011") and
 * returns the label text and SQL strings expected by the `OSPolicyModal`.
 */
export const parseOsVersion = (os_version = ""): string[] => {
  let name = "";
  let version = "";

  if (os_version.startsWith("Ubuntu")) {
    // Ubuntu `os_version` may contain additional text after the point release (e.g., "Ubuntu
    // 16.04.7 LTS")
    name = "Ubuntu";
    version = os_version
      .replace("Ubuntu ", "")
      .slice(0, os_version.indexOf(" ") + 1)
      .trim();
  } else {
    name = os_version.slice(0, os_version.lastIndexOf(" "));
    version = os_version.slice(os_version.lastIndexOf(" ") + 1);
  }

  const policyLabel = `Is ${name}, version ${version} or later, installed?`;
  let policyQuery = "";

  if (name.includes("Windows")) {
    // Windows query is different from Darwin and Linux
    policyQuery = `SELECT 1 from os_version WHERE instr(lower(name), '${name.toLowerCase()}') AND (SELECT data FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\DisplayVersion' LIMIT 1) >= '${version}'`;

    return [policyLabel, policyQuery];
  }

  // Each component of the point release must be compared as a number because simple string comparisons
  // yield unexpected results (e.g., the string "10.0.0" is considered less than "9.0.0")
  const [major, minor, patch] = version
    .split(".")
    .map((str) => parseInt(str, 10) || 0);

  policyQuery = `SELECT 1 from os_version WHERE instr(lower(name), '${name.toLowerCase()}') AND (major > ${major} OR (major = ${major} AND (minor > ${minor} OR (minor = ${minor} AND ${
    // For Ubuntu, the osquery `patch` field is not updated so we need to parse the `version` string
    // using more complicated SQLite dialect
    name !== "Ubuntu"
      ? `patch >= ${patch}`
      : `cast(replace(substr(substr(version, instr(version, '.')+1), instr(substr(version, instr(version, '.')+1), '.')+1), substr(version, instr(version, ' ')), '') as integer) >= ${patch}`
  }))));`;

  return [policyLabel, policyQuery];
};

export default parseOsVersion;
