import React, { Component } from 'react';
import PropTypes from 'prop-types';

import hostInterface from 'interfaces/host';
import labelInterface from 'interfaces/label';
import HostsTable from 'components/hosts/HostsTable';
import HostDetails from 'components/hosts/HostDetails';
import LonelyHost from 'components/hosts/LonelyHost';
import Spinner from 'components/loaders/Spinner';

const baseClass = 'host-container';

class HostContainer extends Component {
  static propTypes = {
    hosts: PropTypes.arrayOf(hostInterface),
    selectedLabel: labelInterface,
    loadingHosts: PropTypes.bool.isRequired,
    displayType: PropTypes.oneOf(['Grid', 'List']),
    toggleAddHostModal: PropTypes.func,
    toggleDeleteHostModal: PropTypes.func,
    onQueryHost: PropTypes.func,
  };

  renderNoHosts = () => {
    const { selectedLabel } = this.props;
    const { type } = selectedLabel || '';
    const isCustom = type === 'custom';

    return (
      <div className={`${baseClass}  ${baseClass}--no-hosts`}>
        <h1>No matching hosts found.</h1>
        <h2>Where are the missing hosts?</h2>
        <ul>
          {isCustom && <li>Check your SQL query above to confirm there are no mistakes.</li>}
          <li>Check to confirm that your hosts are online.</li>
          <li>Confirm that your expected hosts have osqueryd installed and configured.</li>
        </ul>

        <div className={`${baseClass}__no-hosts-contact`}>
          <p>Still having trouble?</p>
          <p><a href="https://github.com/kolide/fleet/issues">File a Github issue</a>.</p>
        </div>
      </div>
    );
  }

  renderHosts = () => {
    const { displayType, hosts, toggleDeleteHostModal, onQueryHost } = this.props;

    if (displayType === 'Grid') {
      return hosts.map((host) => {
        const isLoading = !host.hostname;

        return (
          <HostDetails
            host={host}
            key={`host-${host.id}-details`}
            onDestroyHost={toggleDeleteHostModal}
            onQueryHost={onQueryHost}
            isLoading={isLoading}
          />
        );
      });
    }

    return (
      <HostsTable
        hosts={hosts}
        onDestroyHost={toggleDeleteHostModal}
        onQueryHost={onQueryHost}
      />
    );
  }

  render () {
    const { renderHosts, renderNoHosts } = this;
    const { hosts, displayType, loadingHosts, selectedLabel, toggleAddHostModal } = this.props;

    if (loadingHosts) {
      return <Spinner />;
    }

    if (hosts.length === 0) {
      if (selectedLabel && selectedLabel.type === 'all') {
        return <LonelyHost onClick={toggleAddHostModal} />;
      }

      return renderNoHosts();
    }

    return (
      <div className={`${baseClass} ${baseClass}--${displayType.toLowerCase()}`}>
        {renderHosts()}
      </div>
    );
  }
}

export default HostContainer;
