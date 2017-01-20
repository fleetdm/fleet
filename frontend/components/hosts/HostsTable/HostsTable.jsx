import React, { Component, PropTypes } from 'react';
import classnames from 'classnames';

import Button from 'components/buttons/Button';
import Icon from 'components/icons/Icon';
import PlatformIcon from 'components/icons/PlatformIcon';
import hostInterface from 'interfaces/host';
import iconClassForLabel from 'utilities/icon_class_for_label';

const baseClass = 'hosts-table';

class HostsTable extends Component {
  static propTypes = {
    hosts: PropTypes.arrayOf(hostInterface),
    onDestroyHost: PropTypes.func,
  };

  renderHost = (host) => {
    const { onDestroyHost } = this.props;
    const statusClassName = classnames(`${baseClass}__status`, `${baseClass}__status--${host.status}`);

    return (
      <tr key={`host-${host.id}-table`}>
        <td className={`${baseClass}__hostname`}>{host.hostname}</td>
        <td className={statusClassName}><Icon name={iconClassForLabel(host.status)} /></td>
        <td><PlatformIcon name={host.platform} /> {host.os_version}</td>
        <td>{host.osquery_version}</td>
        <td>{host.host_ip_address}</td>
        <td>{host.host_mac}</td>
        <td><Button onClick={onDestroyHost(host)} variant="unstyled"><Icon name="trash" /></Button></td>
      </tr>
    );
  }

  render () {
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
