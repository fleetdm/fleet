import React, { PropTypes } from 'react';
import { noop } from 'lodash';

import EllipsisMenu from 'components/buttons/EllipsisMenu';
import hostInterface from 'interfaces/host';
import Icon from 'components/icons/Icon';
import { humanMemory, humanUptime, platformIconClass } from './helpers';

const baseClass = 'host-details';

export const STATUSES = {
  online: 'ONLINE',
  offline: 'OFFLINE',
};

const HostDetails = ({ host, onQueryClick = noop, onDisableClick = noop }) => {
  const {
    hostname,
    ip,
    mac,
    memory,
    os_version: osVersion,
    platform,
    status,
    uptime,
  } = host;

  return (
    <div className={`${baseClass} ${baseClass}--${status}`}>
      <EllipsisMenu positionStyles={{ top: '-3px', right: '10px' }}>
        <div className={`${baseClass}__ellipsis-children`}>
          <button className={`${baseClass}__ellipsis-child-item button button--unstyled`} onClick={onQueryClick(host)}>
            <Icon name="query" className={`${baseClass}__query-icon`} />
            <div>Query</div>
          </button>
          <div className={`${baseClass}__vertical-separator`} />
          <button className={`${baseClass}__ellipsis-child-item button button--unstyled`} onClick={onDisableClick(host)}>
            <Icon name="offline" className={`${baseClass}__disabled-icon`} />
            <div>Disable</div>
          </button>
        </div>
      </EllipsisMenu>
      <div className={`${baseClass}__status ${baseClass}__status--${status}`}>
        {status}
      </div>
      <p className={`${baseClass}__hostname`}>{hostname}</p>
      <div className={`${baseClass}__separator`}>
        <div>
          <Icon name={platformIconClass(platform)} className={`${baseClass}__icon`} />
          <span className={`${baseClass}__host-content`}>{osVersion}</span>
        </div>
        <div>
          <span className={`${baseClass}__host-content ${baseClass}__host-content--caps`}>{platform}</span>
        </div>
        <div>
          <span style={{ marginRight: '8px' }}>
            <Icon name="memory" className={`${baseClass}__icon`} />
            <span className={`${baseClass}__host-content`}>{humanMemory(memory)}</span>
          </span>
          <Icon name="uptime" className={`${baseClass}__icon`} />
          <span className={`${baseClass}__host-content`}>{humanUptime(uptime)}</span>
        </div>
        <div>
          <Icon name="mac" className={`${baseClass}__icon`} />
          <span className={`${baseClass}__host-content ${baseClass}__host-content--mono`}>{mac}</span>
        </div>
        <div>
          <Icon name="world" className={`${baseClass}__icon`} />
          <span className={`${baseClass}__host-content ${baseClass}__host-content--mono`}>{ip}</span>
        </div>
      </div>
      <div className={`${baseClass}__separator`}>
        <div>
          <span className={`${baseClass}__host-content ${baseClass}__host-content--caps`}>Tags go here</span>
        </div>
      </div>
    </div>
  );
};

HostDetails.propTypes = {
  host: hostInterface.isRequired,
  onDisableClick: PropTypes.func,
  onQueryClick: PropTypes.func,
};

export default HostDetails;
