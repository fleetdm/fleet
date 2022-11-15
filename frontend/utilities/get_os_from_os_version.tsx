const get_os_from_os_version = (
  os_version: string
): "mac" | "windows" | "linux" => {
  if (os_version.includes("Windows") || os_version.includes("windows")) {
    return "windows";
  } else if (os_version.includes("Mac") || os_version.includes("mac")) {
    return "mac";
  }
  return "linux";
};

export default get_os_from_os_version;
