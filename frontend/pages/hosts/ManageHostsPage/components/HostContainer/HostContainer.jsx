import React, { Component } from 'react';
import PropTypes from 'prop-types';

import labelInterface from 'interfaces/label';
import Button from 'components/buttons/Button';
import InputField from 'components/forms/fields/InputField';
import KolideIcon from 'components/icons/KolideIcon';
import Modal from 'components/modals/Modal';
import RoboDogImage from '../../../../../../assets/images/robo-dog-176x144@2x.png';

import { hostDataHeaders, defaultHiddenColumns } from './HostTableConfig';
import HostsDataTable from '../HostsDataTable/HostsDataTable';
import EditColumnsModal from '../EditColumnsModal/EditColumnsModal';

const baseClass = 'host-container';

// TODO: come back and define table columns here

class HostContainer extends Component {
  static propTypes = {
    selectedFilter: PropTypes.string,
    selectedLabel: labelInterface,
  };

  static defaultProps = {
    selectedLabel: { count: undefined },
  }

  constructor (props) {
    super(props);

    // For now we persist using localstorage. May do server side persistence later.
    const storedHiddenColumns = JSON.parse(localStorage.getItem('hostHiddenColumns'));

    this.state = {
      searchQuery: '',
      showEditColumnsModal: false,
      hiddenColumns: storedHiddenColumns !== null ? storedHiddenColumns : defaultHiddenColumns,
    };
  }

  onSearchQueryChange = (newQuery) => {
    this.setState({
      searchQuery: newQuery,
    });
  }

  onEditColumnsClick = () => {
    this.setState({
      showEditColumnsModal: true,
    });
  }

  onCancelColumns = () => {
    this.setState({
      showEditColumnsModal: false,
    });
  }

  onSaveColumns = (newHiddenColumns) => {
    localStorage.setItem('hostHiddenColumns', JSON.stringify(newHiddenColumns));
    this.setState({
      hiddenColumns: newHiddenColumns,
      showEditColumnsModal: false,
    });
  }

  renderEditColumnsModal = () => {
    const { showEditColumnsModal, hiddenColumns } = this.state;

    if (!showEditColumnsModal) return null;

    return (
      <Modal
        title="Edit Columns"
        onExit={() => this.setState({ showEditColumnsModal: false })}
        className={`${baseClass}__invite-modal`}
      >
        <EditColumnsModal
          columns={hostDataHeaders}
          hiddenColumns={hiddenColumns}
          onSaveColumns={this.onSaveColumns}
          onCancelColumns={this.onCancelColumns}
        />
      </Modal>
    );
  }

  render () {
    const { onSearchQueryChange, renderEditColumnsModal } = this;
    const { selectedFilter, selectedLabel } = this.props;
    const { searchQuery, hiddenColumns } = this.state;

    if (selectedFilter === 'all-hosts' && selectedLabel.count === 0) {
      return (
        <div className={`${baseClass} ${baseClass}--no-hosts`}>
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

    return (
      <div className={`${baseClass}`}>
        {/* TODO: find a way to move these controls into the table component */}
        <div className={`${baseClass}__table-controls`}>
          <Button onClick={this.onEditColumnsClick}>Edit columns</Button>
          <div className={`${baseClass}__search-input`}>
            <InputField
              placeholder="Search hosts by hostname"
              name=""
              onChange={onSearchQueryChange}
              value={searchQuery}
              inputWrapperClass={`${baseClass}__input-wrapper`}
            />
            <KolideIcon name="search" />
          </div>
        </div>
        <HostsDataTable
          selectedFilter={selectedFilter}
          searchQuery={searchQuery}
          tableColumns={hostDataHeaders}
          hiddenColumns={hiddenColumns}
        />
        {renderEditColumnsModal()}
      </div>
    );
  }
}

export default HostContainer;
