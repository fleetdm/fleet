import { find } from 'lodash';

export const parseEntityFunc = (host) => {
  const { network_interfaces: networkInterfaces } = host;
  const networkInterface = networkInterfaces && find(networkInterfaces, { id: host.primary_ip_id });
  const clockSpeed = host.cpu_brand.split('@ ')[1] || host.cpu_brand.split('@')[1];

  const additionalAttrs = {
    host_cpu: `${host.cpu_physical_cores} x ${clockSpeed}`,
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
