import { isFQDN, isIP, isPort } from "validator";

export default (addr: string): boolean => {
  const isValid = isFQDN(addr) || isIP(addr);

  // Previous validators will fail with the address includes a port
  // or if its [IPv6] notation
  if (!isValid) {
    const lastColonIndex = addr.lastIndexOf(":");
    // Ensure colon exists and isn't the first character
    if (lastColonIndex > 0) {
      const port = addr.substring(lastColonIndex + 1);
      let host = addr.substring(0, lastColonIndex);

      // Handle [IPv6] notation
      if (host.startsWith("[") && host.endsWith("]")) {
        host = host.slice(1, -1);
      }

      if (isPort(port) && (isFQDN(host) || isIP(host))) {
        return true;
      }
    }
  }

  return isValid;
};
