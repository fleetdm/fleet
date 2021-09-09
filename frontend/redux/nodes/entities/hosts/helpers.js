export const parseEntityFunc = (host) => {
  let hostCpuOutput = null;
  if (host) {
    let clockSpeedOutput = null;
    try {
      const clockSpeed =
        host.cpu_brand.split("@ ")[1] || host.cpu_brand.split("@")[1];
      const clockSpeedFlt = parseFloat(clockSpeed.split("GHz")[0].trim());
      clockSpeedOutput = Math.floor(clockSpeedFlt * 10) / 10;
    } catch (e) {
      // Some CPU brand strings do not fit this format and we can't parse the
      // clock speed. Leave it set to 'Unknown'.
      console.log(
        `Unable to parse clock speed from cpu_brand: ${host.cpu_brand}`
      );
    }
    if (host.cpu_physical_cores || clockSpeedOutput) {
      hostCpuOutput = `${host.cpu_physical_cores || "Unknown"} x ${
        clockSpeedOutput || "Unknown"
      } GHz`;
    }
  }

  const additionalAttrs = {
    cpu_type: hostCpuOutput,
    target_type: "hosts",
  };

  return {
    ...host,
    ...additionalAttrs,
  };
};

export default { parseEntityFunc };
