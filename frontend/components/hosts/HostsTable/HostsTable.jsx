import React, { Component } from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';

import Button from 'components/buttons/Button';
import KolideIcon from 'components/icons/KolideIcon';
import hostInterface from 'interfaces/host';

import { humanMemory, humanUptime, humanLastSeen } from './helpers';

const baseClass = 'hosts-table';

const ActionButton = ({ host, onDestroyHost, onQueryHost }) => {
  if (host.status === 'online') {
    return (
      <Button onClick={onQueryHost(host)} variant="unstyled">
        <KolideIcon name="query" />
      </Button>
    );
  }

  return (
    <Button onClick={onDestroyHost(host)} variant="unstyled">
      <KolideIcon name="trash" />
    </Button>
  );
};

ActionButton.propTypes = {
  host: hostInterface,
  onDestroyHost: PropTypes.func,
  onQueryHost: PropTypes.func,
};

class HostsTable extends Component {
  static propTypes = {
    hosts: PropTypes.arrayOf(hostInterface),
    onDestroyHost: PropTypes.func,
    onQueryHost: PropTypes.func,
  };

  lastSeenTime = (status, seenTime) => {
    if (status !== 'online') {
      return `Last Seen: ${humanLastSeen(seenTime)} UTC`;
    }

    return 'Online';
  };

  renderHost = (host) => {
    const { onDestroyHost, onQueryHost } = this.props;
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
          {host.hostname}
        </td>
        <td className={statusClassName}>
          {host.status}
        </td>
        <td>{host.os_version}</td>
        <td>{host.osquery_version}</td>
        <td>{host.primary_ip}</td>
        <td>{host.primary_mac}</td>
        <td>{host.host_cpu}</td>
        <td>{humanMemory(host.memory)}</td>
        <td>{humanUptime(host.uptime)}</td>
        <td>
          <ActionButton
            host={host}
            onDestroyHost={onDestroyHost}
            onQueryHost={onQueryHost}
          />
        </td>
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
              <th />
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
