import { find } from 'lodash';

export const parseEntityFunc = (host) => {
  const { network_interfaces: networkInterfaces } = host;
  const networkInterface = networkInterfaces && find(networkInterfaces, { id: host.primary_ip_id });

  let clockSpeed = null;
  let clockSpeedFlt = null;
  let hostCpuOutput = null;

  if (host && host.cpu_brand) {
    clockSpeed = host.cpu_brand.split('@ ')[1] || host.cpu_brand.split('@')[1];
    clockSpeedFlt = parseFloat(clockSpeed.split('GHz')[0].trim());
    hostCpuOutput = `${host.cpu_physical_cores} x ${Math.floor(clockSpeedFlt * 10) / 10} GHz`;
  }

  const additionalAttrs = {
    host_cpu: hostCpuOutput,
    target_type: 'hosts',
  };

  if (networkInterface) {
    additionalAttrs.host_ip_address = networkInterface.address;
    additionalAttrs.host_mac = networkInterface.mac;
  }

  return {
    ...host,
    ...additionalAttrs,
  };
};

export default { parseEntityFunc };
