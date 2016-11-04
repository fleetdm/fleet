import React, { PropTypes } from 'react';
import { noop } from 'lodash';

import EllipsisMenu from '../../../components/buttons/EllipsisMenu';
import hostInterface from '../../../interfaces/host';
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
          <button className={`${baseClass}__ellipsis-child-item button button__unstyled`} onClick={onQueryClick(host)}>
            <i className={`${baseClass}__query-icon kolidecon-query`} />
            <div>Query</div>
          </button>
          <div className={`${baseClass}__vertical-separator`} />
          <button className={`${baseClass}__ellipsis-child-item button button__unstyled`} onClick={onDisableClick(host)}>
            <i className={`${baseClass}__disabled-icon kolidecon-ex`} />
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
          <i className={`${baseClass}__icon ${platformIconClass(platform)}`} />
          <span className={`${baseClass}__host-content`}>{osVersion}</span>
        </div>
        <div>
          <span className={`${baseClass}__host-content ${baseClass}__host-content--caps`}>{platform}</span>
        </div>
        <div>
          <span style={{ marginRight: '8px' }}>
            <i className={`${baseClass}__icon kolidecon-memory`} />
            <span className={`${baseClass}__host-content`}>{humanMemory(memory)}</span>
          </span>
          <i className={`${baseClass}__icon kolidecon-uptime`} />
          <span className={`${baseClass}__host-content`}>{humanUptime(uptime)}</span>
        </div>
        <div>
          <i className={`${baseClass}__icon kolidecon-mac`} />
          <span className={`${baseClass}__host-content ${baseClass}__host-content--mono`}>{mac}</span>
        </div>
        <div>
          <i className={`${baseClass}__icon kolidecon-world`} />
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
