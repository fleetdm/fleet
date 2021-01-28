import React, { Component } from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';

import Button from 'components/buttons/Button';
import hostInterface from 'interfaces/host';

import helpers from 'kolide/helpers';

const baseClass = 'hosts-table';

class HostsTable extends Component {
  static propTypes = {
    hosts: PropTypes.arrayOf(hostInterface),
    onHostClick: PropTypes.func,
  };

  lastSeenTime = (status, seenTime) => {
    const { humanHostLastSeen } = helpers;

    if (status !== 'online') {
      return `Last Seen: ${humanHostLastSeen(seenTime)} UTC`;
    }

    return 'Online';
  };

  renderHost = (host) => {
    const { humanHostMemory, humanHostUptime } = helpers;

    const { onHostClick } = this.props;
    const statusClassName = classnames(
      `${baseClass}__status`,
      `${baseClass}__status--${host.status}`,
    );

    return (
      <tr key={`host-${host.id}-table`}>
        <td
          className={`${baseClass}__hostname`}
          title={this.lastSeenTime(host.status, host.seen_time)}
        >
          <Button onClick={() => onHostClick(host)} variant="text-link">{host.hostname}</Button>
        </td>
        <td className={statusClassName}>
          {host.status}
        </td>
        <td>{host.os_version}</td>
        <td>{host.osquery_version}</td>
        <td>{host.primary_ip}</td>
        <td>{host.primary_mac}</td>
        <td>{host.host_cpu}</td>
        <td>{humanHostMemory(host.memory)}</td>
        <td>{humanHostUptime(host.uptime)}</td>
      </tr>
    );
  };

  render() {
    const { hosts } = this.props;
    const { renderHost } = this;

    return (
      <div className={`${baseClass} ${baseClass}__wrapper`}>
        <table className={`${baseClass}__table`}>
          <thead>
            <tr>
              <th>Hostname</th>
              <th>Status</th>
              <th>OS</th>
              <th>Osquery</th>
              <th>IPv4</th>
              <th>Physical Address</th>
              <th>CPU</th>
              <th>Memory</th>
              <th>Uptime</th>
            </tr>
          </thead>
          <tbody>
            {hosts.map((host) => {
              return renderHost(host);
            })}
          </tbody>
        </table>
      </div>
    );
  }
}

export default HostsTable;
