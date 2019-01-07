import React from 'react';
import PropTypes from 'prop-types';

import Button from 'components/buttons/Button';
import hostInterface from 'interfaces/host';
import Icon from 'components/icons/Icon';
import PlatformIcon from 'components/icons/PlatformIcon';
import CircleLoader from 'components/loaders/Circle';
import { humanMemory, humanUptime, humanLastSeen } from './helpers';

const baseClass = 'host-details';

export const STATUSES = {
  online: 'ONLINE',
  offline: 'OFFLINE',
};

const ActionButton = ({ host, onDestroyHost, onQueryHost }) => {
  if (host.status === 'online') {
    return (
      <Button
        onClick={onQueryHost(host)}
        variant="unstyled"
        title="Query this host"
      >
        <Icon name="query" className={`${baseClass}__cta-host-icon`} />
      </Button>
    );
  }

  return (
    <Button
      onClick={onDestroyHost(host)}
      variant="unstyled"
      title="Delete this host"
    >
      <Icon name="trash" className={`${baseClass}__cta-host-icon`} />
    </Button>
  );
};

const lastSeenTime = (status, seenTime) => {
  if (status !== 'online') {
    return `Last Seen: ${humanLastSeen(seenTime)} UTC`;
  }

  return 'Online';
};

const HostDetails = ({ host, onDestroyHost, onQueryHost, isLoading }) => {
  return (
    <div className={`${baseClass} ${baseClass}--${host.status}`}>
      <header className={`${baseClass}__header`}>
        {!isLoading && (
          <span className={`${baseClass}__cta-host`}>
            <ActionButton
              host={host}
              onDestroyHost={onDestroyHost}
              onQueryHost={onQueryHost}
            />
          </span>
        )}

        <p
          className={`${baseClass}__hostname`}
          title={lastSeenTime(host.status, host.seen_time)}
        >
          {host.hostname || 'incoming host'}
        </p>
      </header>

      {isLoading && (
        <div className={`${baseClass}__loader`}>
          <CircleLoader />
        </div>
      )}
      {!isLoading && (
        <ul className={`${baseClass}__details-list`}>
          <li className={` ${baseClass}__detail ${baseClass}__detail--os`}>
            <PlatformIcon
              name={host.platform}
              className={`${baseClass}__icon`}
              title="Operating System & Version"
            />
            <span className={`${baseClass}__host-content`}>
              {host.os_version || '--'}
            </span>
          </li>

          <li className={` ${baseClass}__detail ${baseClass}__detail--osquery`}>
            <Icon
              name="osquery"
              className={`${baseClass}__icon`}
              title="Osquery Version"
            />
            <span className={`${baseClass}__host-content`}>
              {host.osquery_version || '--'}
            </span>
          </li>

          <li className={` ${baseClass}__detail ${baseClass}__detail--cpu`}>
            <Icon
              name="cpu"
              className={`${baseClass}__icon`}
              title="CPU Cores and Speed"
            />
            <span className={`${baseClass}__host-content`}>
              {host.host_cpu || '--'}
            </span>
          </li>

          <li className={` ${baseClass}__detail ${baseClass}__detail--memory`}>
            <Icon
              name="memory"
              className={`${baseClass}__icon`}
              title="Memory / RAM"
            />
            <span className={`${baseClass}__host-content`}>
              {humanMemory(host.memory) || '--'}
            </span>
          </li>

          <li className={` ${baseClass}__detail ${baseClass}__detail--uptime`}>
            <Icon
              name="uptime"
              className={`${baseClass}__icon`}
              title="Uptime"
            />
            <span className={`${baseClass}__host-content`}>
              {humanUptime(host.uptime) || '--'}
            </span>
          </li>

          <li className={` ${baseClass}__detail ${baseClass}__detail--mac`}>
            <Icon
              name="mac"
              className={`${baseClass}__icon`}
              title="MAC Address"
            />
            <span
              className={`${baseClass}__host-content ${baseClass}__host-content--mono`}
            >
              {host.host_mac ? host.host_mac.toUpperCase() : '--'}
            </span>
          </li>

          <li className={` ${baseClass}__detail ${baseClass}__detail--ip`}>
            <Icon
              name="world"
              className={`${baseClass}__icon`}
              title="IP Address"
            />
            <span
              className={`${baseClass}__host-content ${baseClass}__host-content--mono`}
            >
              {host.host_ip_address || '--'}
            </span>
          </li>
        </ul>
      )}
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
  isLoading: PropTypes.bool.isRequired,
};

export default HostDetails;
