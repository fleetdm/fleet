import React, { PropTypes } from 'react';

import Button from 'components/buttons/Button';
import hostInterface from 'interfaces/host';
import Icon from 'components/icons/Icon';
import PlatformIcon from 'components/icons/PlatformIcon';
import { humanMemory, humanUptime } from './helpers';

const baseClass = 'host-details';

export const STATUSES = {
  online: 'ONLINE',
  offline: 'OFFLINE',
};

const ActionButton = ({ host, onDestroyHost, onQueryHost }) => {
  if (host.status === 'online') {
    return (
      <Button onClick={onQueryHost(host)} variant="unstyled" title="Query this host">
        <Icon name="query" className={`${baseClass}__cta-host-icon`} />
      </Button>
    );
  }

  return (
    <Button onClick={onDestroyHost(host)} variant="unstyled" title="Delete this host">
      <Icon name="trash" className={`${baseClass}__cta-host-icon`} />
    </Button>
  );
};

const HostDetails = ({ host, onDestroyHost, onQueryHost }) => {
  const {
    host_cpu: hostCpu,
    host_mac: hostMac,
    host_ip_address: hostIpAddress,
    hostname,
    memory,
    os_version: osVersion,
    osquery_version: osqueryVersion,
    platform,
    status,
    uptime,
  } = host;

  return (
    <div className={`${baseClass} ${baseClass}--${status}`}>
      <span className={`${baseClass}__cta-host`}>
        <ActionButton host={host} onDestroyHost={onDestroyHost} onQueryHost={onQueryHost} />
      </span>

      <p className={`${baseClass}__status`}>{status}</p>

      <p className={`${baseClass}__hostname`}>{hostname}</p>

      <ul className={`${baseClass}__details-list`}>
        {!!osVersion && <li className={` ${baseClass}__detail ${baseClass}__detail--os`}>
          <PlatformIcon name={platform} className={`${baseClass}__icon`} />
          <span className={`${baseClass}__host-content`}>{osVersion}</span>
        </li>}

        {!!hostCpu && <li className={` ${baseClass}__detail ${baseClass}__detail--cpu`}>
          <Icon name="cpu" className={`${baseClass}__icon`} />
          <span className={`${baseClass}__host-content`}>{hostCpu}</span>
        </li>}

        {!!osqueryVersion && <li className={` ${baseClass}__detail ${baseClass}__detail--osquery`}>
          <Icon name="osquery" className={`${baseClass}__icon`} />
          <span className={`${baseClass}__host-content`}>{osqueryVersion}</span>
        </li>}

        {!!memory && <li className={` ${baseClass}__detail ${baseClass}__detail--memory`}>
          <Icon name="memory" className={`${baseClass}__icon`} />
          <span className={`${baseClass}__host-content`}>{humanMemory(memory)}</span>
        </li>}

        {!!uptime && <li className={` ${baseClass}__detail ${baseClass}__detail--uptime`}>
          <Icon name="uptime" className={`${baseClass}__icon`} />
          <span className={`${baseClass}__host-content`}>{humanUptime(uptime)}</span>
        </li>}

        {!!hostMac && <li className={` ${baseClass}__detail ${baseClass}__detail--mac`}>
          <Icon name="mac" className={`${baseClass}__icon`} />
          <span className={`${baseClass}__host-content ${baseClass}__host-content--mono`}>{hostMac}</span>
        </li>}

        {!!hostIpAddress && <li className={` ${baseClass}__detail ${baseClass}__detail--ip`}>
          <Icon name="world" className={`${baseClass}__icon`} />
          <span className={`${baseClass}__host-content ${baseClass}__host-content--mono`}>{hostIpAddress}</span>
        </li>}
      </ul>
    </div>
  );
};

ActionButton.propTypes = {
  host: hostInterface.isRequired,
  onDestroyHost: PropTypes.func.isRequired,
  onQueryHost: PropTypes.func.isRequired,
};

HostDetails.propTypes = {
  host: hostInterface.isRequired,
  onDestroyHost: PropTypes.func.isRequired,
  onQueryHost: PropTypes.func.isRequired,
};

export default HostDetails;
