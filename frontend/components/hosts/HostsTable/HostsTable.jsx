import React, { Component } from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';

import Button from 'components/buttons/Button';
import Icon from 'components/icons/Icon';
import PlatformIcon from 'components/icons/PlatformIcon';
import hostInterface from 'interfaces/host';
import iconClassForLabel from 'utilities/icon_class_for_label';

import { humanLastSeen } from '../HostDetails/helpers';

const baseClass = 'hosts-table';

const ActionButton = ({ host, onDestroyHost, onQueryHost }) => {
  if (host.status === 'online') {
    return (
      <Button onClick={onQueryHost(host)} variant="unstyled">
        <Icon name="query" />
      </Button>
    );
  }

  return (
    <Button onClick={onDestroyHost(host)} variant="unstyled">
      <Icon name="trash" />
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
      `${baseClass}__status--${host.status}`
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
          <Icon name={iconClassForLabel(host.status)} />
        </td>
        <td>
          <PlatformIcon name={host.platform} title={host.os_version} />{' '}
          {host.os_version}
        </td>
        <td>{host.osquery_version}</td>
        <td>{host.host_ip_address}</td>
        <td>{host.host_mac}</td>
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
