import React, { Component } from "react";
import PropTypes from "prop-types";
import ReactTooltip from "react-tooltip";

import labelInterface from "interfaces/label";
import { getHostTableData } from "redux/nodes/components/ManageHostsPage/actions";
import Button from "components/buttons/Button";
import InputField from "components/forms/fields/InputField";
import KolideIcon from "components/icons/KolideIcon";
import Modal from "components/modals/Modal";
import RoboDogImage from "../../../../../../assets/images/robo-dog-176x144@2x.png";
import EditColumnsIcon from "../../../../../../assets/images/icon-edit-columns-20x20@2x.png";

import { hostDataHeaders, defaultHiddenColumns } from "./HostTableConfig";
import DataTable from "../DataTable/DataTable";
import EditColumnsModal from "../EditColumnsModal/EditColumnsModal";

const baseClass = "host-container";

const EmptyHosts = () => {
  return (
    <div className={`${baseClass}  ${baseClass}--no-hosts`}>
      <div className={`${baseClass}--no-hosts__inner`}>
        <div className={"no-filter-results"}>
          <h1>No hosts match the current criteria</h1>
          <p>
            Expecting to see new hosts? Try again in a few seconds as the system
            catches up
          </p>
        </div>
      </div>
    </div>
  );
};

class HostContainer extends Component {
  static propTypes = {
    selectedFilter: PropTypes.string,
    selectedLabel: labelInterface,
  };

  static defaultProps = {
    selectedLabel: { count: undefined },
  };

  constructor(props) {
    super(props);

    // For now we persist using localstorage. May do server side persistence later.
    const storedHiddenColumns = JSON.parse(
      localStorage.getItem("hostHiddenColumns")
    );

    this.state = {
      searchQuery: "",
      showEditColumnsModal: false,
      hiddenColumns:
        storedHiddenColumns !== null
          ? storedHiddenColumns
          : defaultHiddenColumns,
    };
  }

  onSearchQueryChange = (newQuery) => {
    this.setState({
      searchQuery: newQuery,
    });
  };

  onEditColumnsClick = () => {
    this.setState({
      showEditColumnsModal: true,
    });
  };

  onCancelColumns = () => {
    this.setState({
      showEditColumnsModal: false,
    });
  };

  onSaveColumns = (newHiddenColumns) => {
    localStorage.setItem("hostHiddenColumns", JSON.stringify(newHiddenColumns));
    this.setState({
      hiddenColumns: newHiddenColumns,
      showEditColumnsModal: false,
    });
  };

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
  };

  render() {
    const { onSearchQueryChange, renderEditColumnsModal } = this;
    const { selectedFilter, selectedLabel } = this.props;
    const { searchQuery, hiddenColumns } = this.state;

    if (selectedFilter === "all-hosts" && selectedLabel.count === 0) {
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
                <a href="https://github.com/fleetdm/fleet/issues">
                  File a GitHub issue
                </a>
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
          <Button
            onClick={this.onEditColumnsClick}
            variant="unstyled"
            className={`${baseClass}__edit-columns-button`}
          >
            <img src={EditColumnsIcon} alt="edit columns icon" />
            Edit columns
          </Button>
          <div data-for="search" className={`${baseClass}__search-input`}>
            <InputField
              placeholder="Search hostname, UUID, serial number, or IPv4"
              name=""
              onChange={onSearchQueryChange}
              value={searchQuery}
              inputWrapperClass={`${baseClass}__input-wrapper`}
            />
            <KolideIcon name="search" />
          </div>
          <ReactTooltip
            place="bottom"
            type="dark"
            effect="solid"
            id="search"
            backgroundColor="#3e4771"
          >
            <span className={`${baseClass}__tooltip-text`}>
              Search by hostname, UUID, serial number, or IPv4
            </span>
          </ReactTooltip>
        </div>
        <DataTable
          selectedFilter={selectedFilter}
          searchQuery={searchQuery}
          tableColumns={hostDataHeaders}
          hiddenColumns={hiddenColumns}
          pageSize={100}
          defaultSortHeader={hostDataHeaders[0].accessor}
          resultsName={"hosts"}
          fetchDataAction={getHostTableData}
          entity={"hosts"}
          emptyComponent={EmptyHosts}
        />
        {renderEditColumnsModal()}
      </div>
    );
  }
}

export default HostContainer;
