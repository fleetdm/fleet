import React, { Component, PropTypes } from 'react';
import classnames from 'classnames';

import Icon from 'components/Icon';
import hostInterface from 'interfaces/host';
import { platformIconClass, statusIconClass } from 'utilities/icon_class';

const baseClass = 'hosts-table';

class HostsTable extends Component {
  static propTypes = {
    hosts: PropTypes.arrayOf(hostInterface),
  };

  renderHost = (host) => {
    const statusClassName = classnames(`${baseClass}__status`, `${baseClass}__status--${host.status}`);

    return (
      <tr key={`host-${host.id}-table`}>
        <td className={`${baseClass}__hostname`}>{host.hostname}</td>
        <td className={statusClassName}><Icon name={statusIconClass(host.status)} /></td>
        <td><Icon name={platformIconClass(host.platform)} /> {host.os_version}</td>
        <td>{host.osquery_version}</td>
        <td>{host.ip}</td>
        <td>{host.mac}</td>
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
