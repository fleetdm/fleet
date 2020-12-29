import React, { Component } from 'react';
import PropTypes from 'prop-types';

import hostInterface from 'interfaces/host';
import labelInterface from 'interfaces/label';
import HostsTable from 'components/hosts/HostsTable';
import Spinner from 'components/loaders/Spinner';
import RoboDogImage from '../../../../assets/images/robo-dog-176x144@2x.png';

const baseClass = 'host-container';

class HostContainer extends Component {
  static propTypes = {
    hosts: PropTypes.arrayOf(hostInterface),
    selectedLabel: labelInterface,
    loadingHosts: PropTypes.bool.isRequired,
    toggleDeleteHostModal: PropTypes.func,
    onQueryHost: PropTypes.func,
  };

  renderNoHosts = () => {
    const { selectedLabel } = this.props;
    const { type } = selectedLabel || '';
    const isCustom = type === 'custom';

    return (
      <div className={`${baseClass}  ${baseClass}--no-hosts`}>
        <div className={`${baseClass}--no-hosts__inner`}>
          <img src={RoboDogImage} alt="robo dog" />
          <div>
            <h1>No matching hosts found.</h1>
            <h2>Where are the missing hosts?</h2>
            <ul>
              {isCustom && <li>Check your SQL query above to confirm there are no mistakes.</li>}
              <li>Check to confirm that your hosts are online.</li>
              <li>Confirm that your expected hosts have osqueryd installed and configured.</li>
            </ul>

            <div className={`${baseClass}__no-hosts-contact`}>
              <p>Still having trouble?</p>
              <a href="https://github.com/fleetdm/fleet/issues">File a GitHub issue</a>
            </div>
          </div>
        </div>
      </div>
    );
  }

  renderHosts = () => {
    const { hosts, toggleDeleteHostModal, onQueryHost } = this.props;

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
    const { hosts, loadingHosts, selectedLabel } = this.props;

    if (loadingHosts) {
      return <Spinner />;
    }

    if (hosts.length === 0) {
      if (selectedLabel && selectedLabel.type === 'all') {
        return (
          <div className={`${baseClass}  ${baseClass}--no-hosts`}>
            <div className={`${baseClass}--no-hosts__inner`}>
              <img src={RoboDogImage} alt="No Hosts" />
              <div>
                <h1>It&#39;s kinda empty in here...</h1>
                <h2>Get started adding hosts to Fleet.</h2>
                <p>Add your laptops and servers to securely monitor them.</p>
                <div className={`${baseClass}__no-hosts-contact`}>
                  <p>Still having trouble?</p>
                  <a href="https://github.com/fleetdm/fleet/issues">File a GitHub issue</a>
                </div>
              </div>
            </div>
          </div>
        );
      }

      return renderNoHosts();
    }

    return (
      <div className={`${baseClass}`}>
        {renderHosts()}
      </div>
    );
  }
}

export default HostContainer;
