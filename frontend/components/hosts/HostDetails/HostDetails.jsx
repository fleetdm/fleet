import React, { PropTypes } from 'react';
import { noop } from 'lodash';
import radium from 'radium';

import componentStyles from './styles';
import ElipsisMenu from '../../../components/buttons/ElipsisMenu';
import hostInterface from '../../../interfaces/host';
import { humanMemory, humanUptime, platformIconClass } from './helpers';

const {
  containerStyles,
  contentSeparatorStyles,
  disableIconStyles,
  elipsisChildItemStyles,
  elipsisChidrenWrapperStyles,
  elipsisPositionStyles,
  hostContentItemStyles,
  hostnameStyles,
  iconStyles,
  monoStyles,
  queryIconStyles,
  statusStyles,
  verticleRuleStyles,
} = componentStyles;
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
    <div style={containerStyles(status)}>
      <ElipsisMenu positionStyles={elipsisPositionStyles}>
        <div style={elipsisChidrenWrapperStyles}>
          <button className="btn--unstyled" onClick={onQueryClick(host)} style={elipsisChildItemStyles}>
            <i className="kolidecon-query" style={queryIconStyles} />
            <div>Query</div>
          </button>
          <div style={verticleRuleStyles} />
          <button className="btn--unstyled" onClick={onDisableClick(host)} style={elipsisChildItemStyles}>
            <i className="kolidecon-ex" style={disableIconStyles} />
            <div>Disable</div>
          </button>
        </div>
      </ElipsisMenu>
      <div style={statusStyles(status)}>
        {status}
      </div>
      <p style={hostnameStyles}>{hostname}</p>
      <div style={contentSeparatorStyles}>
        <div>
          <i className={platformIconClass(platform)} style={iconStyles} />
          <span style={hostContentItemStyles}>{osVersion}</span>
        </div>
        <div>
          <span style={[hostContentItemStyles, { textTransform: 'capitalize' }]}>{platform}</span>
        </div>
        <div>
          <span style={{ marginRight: '8px' }}>
            <i className="kolidecon-memory" style={iconStyles} />
            <span style={hostContentItemStyles}>{humanMemory(memory)}</span>
          </span>
          <i className="kolidecon-uptime" style={iconStyles} />
          <span style={hostContentItemStyles}>{humanUptime(uptime)}</span>
        </div>
        <div>
          <i className="kolidecon-mac" style={iconStyles} />
          <span style={[hostContentItemStyles, monoStyles]}>{mac}</span>
        </div>
        <div>
          <i className="kolidecon-world" style={iconStyles} />
          <span style={[hostContentItemStyles, monoStyles]}>{ip}</span>
        </div>
      </div>
      <div style={contentSeparatorStyles}>
        <div>
          <span style={[hostContentItemStyles, { textTransform: 'capitalize' }]}>Tags go here</span>
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

export default radium(HostDetails);
