import React, { PureComponent } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import { push } from "react-router-redux";

import Button from "components/buttons/Button";
import configInterface from "interfaces/config";
import HostSidePanel from "components/side_panels/HostSidePanel";
import LabelForm from "components/forms/LabelForm";
import Modal from "components/modals/Modal";
import QuerySidePanel from "components/side_panels/QuerySidePanel";
import TableContainer from "components/TableContainer";
import labelInterface from "interfaces/label";
import teamInterface from "interfaces/team";
import userInterface from "interfaces/user";
import osqueryTableInterface from "interfaces/osquery_table";
import statusLabelsInterface from "interfaces/status_labels";
import enrollSecretInterface from "interfaces/enroll_secret";
import { selectOsqueryTable } from "redux/nodes/components/QueryPages/actions";
import { renderFlash } from "redux/nodes/notifications/actions";
import labelActions from "redux/nodes/entities/labels/actions";
import teamActions from "redux/nodes/entities/teams/actions";
import hostActions from "redux/nodes/entities/hosts/actions";
import entityGetter, { memoizedGetEntity } from "redux/utilities/entityGetter";
import { getLabels } from "redux/nodes/components/ManageHostsPage/actions";
import PATHS from "router/paths";
import deepDifference from "utilities/deep_difference";
import { find } from "lodash";

import hostClient from "services/entities/hosts";

import permissionUtils from "utilities/permissions";
import Dropdown from "components/forms/fields/Dropdown";
import {
  defaultHiddenColumns,
  generateVisibleTableColumns,
  generateAvailableTableHeaders,
} from "./HostTableConfig";
import AddHostModal from "./components/AddHostModal";
import NoHosts from "./components/NoHosts";
import EmptyHosts from "./components/EmptyHosts";
import EditColumnsModal from "./components/EditColumnsModal/EditColumnsModal";
import TransferHostModal from "./components/TransferHostModal";
import EditColumnsIcon from "../../../../assets/images/icon-edit-columns-16x12@2x.png";
import PencilIcon from "../../../../assets/images/icon-pencil-14x14@2x.png";
import TrashIcon from "../../../../assets/images/icon-trash-14x14@2x.png";

const NEW_LABEL_HASH = "#new_label";
const EDIT_LABEL_HASH = "#edit_label";
const ALL_HOSTS_LABEL = "all-hosts";
const baseClass = "manage-hosts";
const LABEL_SLUG_PREFIX = "labels/";

const HOST_SELECT_STATUSES = [
  {
    disabled: false,
    label: "All hosts",
    value: ALL_HOSTS_LABEL,
    helpText: "All hosts which have enrolled to Fleet.",
  },
  {
    disabled: false,
    label: "Online hosts",
    value: "online",
    helpText: "Hosts that have recently checked-in to Fleet.",
  },
  {
    disabled: false,
    label: "Offline hosts",
    value: "offline",
    helpText: "Hosts that have not checked-in to Fleet recently.",
  },
  {
    disabled: false,
    label: "New hosts",
    value: "new",
    helpText: "Hosts that have been enrolled to Fleet in the last 24 hours.",
  },
  {
    disabled: false,
    label: "MIA hosts",
    value: "mia",
    helpText: "Hosts that have not been seen by Fleet in more than 30 days.",
  },
];

export class ManageHostsPage extends PureComponent {
  static propTypes = {
    config: configInterface,
    dispatch: PropTypes.func,
    isAddLabel: PropTypes.bool,
    isEditLabel: PropTypes.bool,
    labelErrors: PropTypes.shape({
      base: PropTypes.string,
    }),
    labels: PropTypes.arrayOf(labelInterface),
    loadingLabels: PropTypes.bool.isRequired,
    enrollSecret: enrollSecretInterface,
    selectedFilters: PropTypes.arrayOf(PropTypes.string),
    selectedLabel: labelInterface,
    selectedOsqueryTable: osqueryTableInterface,
    statusLabels: statusLabelsInterface,
    loadingHosts: PropTypes.bool,
    canAddNewHosts: PropTypes.bool,
    canAddNewLabels: PropTypes.bool,
    teams: PropTypes.arrayOf(teamInterface),
    isGlobalAdmin: PropTypes.bool,
    isBasicTier: PropTypes.bool,
    currentUser: userInterface,
  };

  static defaultProps = {
    loadingLabels: false,
  };

  constructor(props) {
    super(props);

    // For now we persist hidden columns using localstorage. May do server side
    // persistence later.
    const storedHiddenColumns = JSON.parse(
      localStorage.getItem("hostHiddenColumns")
    );

    this.state = {
      labelQueryText: "",
      showAddHostModal: false,
      selectedHost: null,
      showDeleteLabelModal: false,
      showEditColumnsModal: false,
      showTransferHostModal: false,
      hiddenColumns:
        storedHiddenColumns !== null
          ? storedHiddenColumns
          : defaultHiddenColumns,
      selectedHostIds: [],
      isAllMatchingHostsSelected: false,
      searchQuery: "",
      hosts: [],
      isHostsLoading: false,
    };
  }

  componentDidMount() {
    const { dispatch } = this.props;
    dispatch(getLabels());
  }

  componentDidUpdate(prevProps) {
    const { dispatch, isBasicTier, canAddNewHosts } = this.props;
    if (
      isBasicTier !== prevProps.isBasicTier &&
      isBasicTier &&
      canAddNewHosts
    ) {
      dispatch(teamActions.loadAll({}));
    }
  }

  componentWillUnmount() {
    this.clearHostUpdates();
  }

  onAddLabelClick = (evt) => {
    evt.preventDefault();
    const { dispatch } = this.props;
    dispatch(push(`${PATHS.MANAGE_HOSTS}${NEW_LABEL_HASH}`));
  };

  onEditLabelClick = (evt) => {
    evt.preventDefault();
    const { getLabelSelected } = this;
    const { dispatch } = this.props;
    dispatch(
      push(`${PATHS.MANAGE_HOSTS}/${getLabelSelected()}${EDIT_LABEL_HASH}`)
    );
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

  onCancelAddLabel = () => {
    const { dispatch, selectedFilters } = this.props;
    dispatch(push(`${PATHS.MANAGE_HOSTS}/${selectedFilters.join("/")}`));
  };

  onCancelEditLabel = () => {
    const { dispatch, selectedFilters } = this.props;
    dispatch(push(`${PATHS.MANAGE_HOSTS}/${selectedFilters.join("/")}`));
  };

  onAddHostClick = (evt) => {
    evt.preventDefault();
    const { toggleAddHostModal } = this;
    toggleAddHostModal();
  };

  onChangeTeam = (team) => {
    const { dispatch } = this.props;
    dispatch(teamActions.getEnrollSecrets(team));
  };

  // NOTE: this is called once on the initial rendering. The initial render of
  // the TableContainer child component will call this handler.
  onTableQueryChange = async ({
    pageIndex,
    pageSize,
    searchQuery,
    sortHeader,
    sortDirection,
  }) => {
    const { retrieveHosts } = this;
    const { selectedFilters } = this.props;

    let sortBy = [];
    if (sortHeader !== "") {
      sortBy = [{ id: sortHeader, direction: sortDirection }];
    }

    // keep track as a local state to be used later
    this.setState({ searchQuery });

    retrieveHosts({
      page: pageIndex,
      perPage: pageSize,
      selectedLabels: selectedFilters,
      globalFilter: searchQuery,
      sortBy,
    });
  };

  onEditLabel = (formData) => {
    const { getLabelSelected } = this;
    const { dispatch, selectedLabel } = this.props;
    const updateAttrs = deepDifference(formData, selectedLabel);

    return dispatch(labelActions.update(selectedLabel, updateAttrs))
      .then(() => {
        dispatch(push(`${PATHS.MANAGE_HOSTS}/${getLabelSelected()}`));
        dispatch(
          renderFlash(
            "success",
            "Label updated. Try refreshing this page in just a moment to see the updated host count for your label."
          )
        );
        return false;
      })
      .catch(() => false);
  };

  onLabelClick = (selectedLabel) => {
    return (evt) => {
      evt.preventDefault();

      const { handleLabelChange } = this;
      handleLabelChange(selectedLabel);
    };
  };

  onOsqueryTableSelect = (tableName) => {
    const { dispatch } = this.props;
    dispatch(selectOsqueryTable(tableName));

    return false;
  };

  onSaveAddLabel = (formData) => {
    const { dispatch } = this.props;

    return dispatch(labelActions.create(formData)).then(() => {
      dispatch(push(PATHS.MANAGE_HOSTS));
      dispatch(
        renderFlash(
          "success",
          "Label created. Try refreshing this page in just a moment to see the updated host count for your label."
        )
      );
      return false;
    });
  };

  onDeleteLabel = () => {
    const { toggleDeleteLabelModal } = this;
    const { dispatch, selectedLabel } = this.props;
    const { MANAGE_HOSTS } = PATHS;

    return dispatch(labelActions.destroy(selectedLabel)).then(() => {
      toggleDeleteLabelModal();
      dispatch(push(MANAGE_HOSTS));
      return false;
    });
  };

  onTransferToTeamClick = (selectedHostIds) => {
    const { toggleTransferHostModal } = this;
    toggleTransferHostModal();
    this.setState({ selectedHostIds });
  };

  onTransferHostSubmit = (team) => {
    const {
      toggleTransferHostModal,
      isAcceptableStatus,
      getStatusSelected,
      retrieveHosts,
    } = this;
    const { dispatch, selectedFilters, selectedLabel } = this.props;
    const {
      selectedHostIds,
      isAllMatchingHostsSelected,
      searchQuery,
    } = this.state;
    const teamId = team.id === "no-team" ? null : team.id;
    let action = hostActions.transferToTeam(teamId, selectedHostIds);

    if (isAllMatchingHostsSelected) {
      let status = "";
      let labelId = null;

      if (isAcceptableStatus(getStatusSelected())) {
        status = getStatusSelected();
      } else {
        labelId = selectedLabel.id;
      }

      action = hostActions.transferToTeamByFilter(
        teamId,
        searchQuery,
        status,
        labelId
      );
    }

    dispatch(action)
      .then(() => {
        const successMessage =
          teamId === null
            ? `Hosts successfully removed from teams.`
            : `Hosts successfully transferred to  ${team.name}.`;
        dispatch(renderFlash("success", successMessage));
        retrieveHosts({
          selectedLabels: selectedFilters,
          globalFilter: searchQuery,
        });
      })
      .catch(() => {
        dispatch(
          renderFlash("error", "Could not transfer hosts. Please try again.")
        );
      });

    toggleTransferHostModal();
    this.setState({ selectedHostIds: [] });
    this.setState({ isAllMatchingHostsSelected: false });
  };

  getLabelSelected = () => {
    const { selectedFilters } = this.props;
    return selectedFilters.find((f) => f.includes(LABEL_SLUG_PREFIX));
  };

  getStatusSelected = () => {
    const { selectedFilters } = this.props;
    return selectedFilters.find((f) => !f.includes(LABEL_SLUG_PREFIX));
  };

  retrieveHosts = async (options) => {
    const { dispatch } = this.props;
    this.setState({ isHostsLoading: true });

    try {
      const { hosts } = await hostClient.loadAll(options);
      this.setState({ hosts });
    } catch (error) {
      console.log(error);
      dispatch(
        renderFlash("error", "Sorry, we could not retrieve your hosts.")
      );
    } finally {
      this.setState({ isHostsLoading: false });
    }
  };

  isAcceptableStatus = (filter) => {
    return (
      filter === "new" ||
      filter === "online" ||
      filter === "offline" ||
      filter === "mia"
    );
  };

  clearHostUpdates() {
    if (this.timeout) {
      global.window.clearTimeout(this.timeout);
      this.timeout = null;
    }
  }

  toggleAddHostModal = () => {
    const { showAddHostModal } = this.state;
    this.setState({ showAddHostModal: !showAddHostModal });
  };

  toggleDeleteLabelModal = () => {
    const { showDeleteLabelModal } = this.state;
    this.setState({ showDeleteLabelModal: !showDeleteLabelModal });
  };

  toggleTransferHostModal = () => {
    const { showTransferHostModal } = this.state;
    this.setState({ showTransferHostModal: !showTransferHostModal });
  };

  toggleAllMatchingHosts = (shouldSelect = undefined) => {
    const { isAllMatchingHostsSelected } = this.state;

    if (shouldSelect !== undefined) {
      this.setState({ isAllMatchingHostsSelected: shouldSelect });
    } else {
      this.setState({
        isAllMatchingHostsSelected: !isAllMatchingHostsSelected,
      });
    }
  };

  handleLabelChange = ({ slug, type }) => {
    const { dispatch, selectedFilters } = this.props;
    const { MANAGE_HOSTS } = PATHS;
    const isAllHosts = slug === ALL_HOSTS_LABEL;
    const newFilters = [...selectedFilters];

    if (!isAllHosts) {
      // always remove "all-hosts" from the filters first because we don't want
      // something like ["label/8", "all-hosts"]
      const allIndex = newFilters.findIndex((f) => f.includes(ALL_HOSTS_LABEL));
      allIndex > -1 && newFilters.splice(allIndex, 1);

      // replace slug for new params
      let index;
      if (slug.includes(LABEL_SLUG_PREFIX)) {
        index = newFilters.findIndex((f) => f.includes(LABEL_SLUG_PREFIX));
      } else {
        index = newFilters.findIndex((f) => !f.includes(LABEL_SLUG_PREFIX));
      }

      if (index > -1) {
        newFilters.splice(index, 1, slug);
      } else {
        newFilters.push(slug);
      }
    }

    const nextLocation = isAllHosts
      ? MANAGE_HOSTS
      : `${MANAGE_HOSTS}/${newFilters.join("/")}`;
    dispatch(push(nextLocation));
  };

  handleStatusDropdownChange = (statusName) => {
    const { handleLabelChange } = this;
    const { labels } = this.props;

    // we want the full label object
    const isAll = statusName === ALL_HOSTS_LABEL;
    const selected = isAll
      ? find(labels, { type: "all" })
      : find(labels, { id: statusName });
    handleLabelChange(selected);
  };

  renderEditColumnsModal = () => {
    const { config, currentUser } = this.props;
    const { showEditColumnsModal, hiddenColumns } = this.state;

    if (!showEditColumnsModal) return null;

    return (
      <Modal
        title="Edit Columns"
        onExit={() => this.setState({ showEditColumnsModal: false })}
        className={`${baseClass}__invite-modal`}
      >
        <EditColumnsModal
          columns={generateAvailableTableHeaders(config, currentUser)}
          hiddenColumns={hiddenColumns}
          onSaveColumns={this.onSaveColumns}
          onCancelColumns={this.onCancelColumns}
        />
      </Modal>
    );
  };

  renderAddHostModal = () => {
    const { toggleAddHostModal, onChangeTeam } = this;
    const { showAddHostModal } = this.state;
    const { enrollSecret, config, canAddNewHosts, teams } = this.props;

    if (!canAddNewHosts || !showAddHostModal) {
      return null;
    }

    return (
      <Modal
        title="New host"
        onExit={toggleAddHostModal}
        className={`${baseClass}__invite-modal`}
      >
        <AddHostModal
          teams={teams}
          onChangeTeam={onChangeTeam}
          onReturnToApp={toggleAddHostModal}
          enrollSecret={enrollSecret}
          config={config}
        />
      </Modal>
    );
  };

  renderDeleteLabelModal = () => {
    const { showDeleteLabelModal } = this.state;
    const { toggleDeleteLabelModal, onDeleteLabel } = this;

    if (!showDeleteLabelModal) {
      return false;
    }

    return (
      <Modal
        title="Delete label"
        onExit={toggleDeleteLabelModal}
        className={`${baseClass}_delete-label__modal`}
      >
        <p>Are you sure you wish to delete this label?</p>
        <div className={`${baseClass}__modal-buttons`}>
          <Button onClick={toggleDeleteLabelModal} variant="inverse-alert">
            Cancel
          </Button>
          <Button onClick={onDeleteLabel} variant="alert">
            Delete
          </Button>
        </div>
      </Modal>
    );
  };

  renderTransferHostModal = () => {
    const { toggleTransferHostModal, onTransferHostSubmit } = this;
    const { teams, isGlobalAdmin } = this.props;
    const { showTransferHostModal } = this.state;

    if (!showTransferHostModal) return null;

    return (
      <TransferHostModal
        isGlobalAdmin={isGlobalAdmin}
        teams={teams}
        onSubmit={onTransferHostSubmit}
        onCancel={toggleTransferHostModal}
      />
    );
  };

  renderHeaderLabelBlock = ({
    description,
    display_text: displayText,
    type,
  }) => {
    const { onEditLabelClick, toggleDeleteLabelModal } = this;

    return (
      <div className={`${baseClass}__label-block`}>
        <div className="title">
          <span>{displayText}</span>
          {type !== "platform" && (
            <>
              <Button onClick={onEditLabelClick} variant={"text-icon"}>
                <img src={PencilIcon} alt="Edit label" />
              </Button>
              <Button onClick={toggleDeleteLabelModal} variant={"text-icon"}>
                <img src={TrashIcon} alt="Delete label" />
              </Button>
            </>
          )}
        </div>
        <div className="description">
          <span>{description}</span>
        </div>
      </div>
    );
  };

  renderHeader = () => {
    const { renderHeaderLabelBlock } = this;
    const { isAddLabel, selectedLabel } = this.props;

    if (!selectedLabel || isAddLabel) {
      return false;
    }

    const { type } = selectedLabel;
    return (
      <div className={`${baseClass}__header`}>
        <div className={`${baseClass}__text`}>
          <h1 className={`${baseClass}__title`}>
            <span>Hosts</span>
          </h1>
          {type !== "all" &&
            type !== "status" &&
            renderHeaderLabelBlock(selectedLabel)}
        </div>
      </div>
    );
  };

  renderForm = () => {
    const { isAddLabel, isEditLabel, labelErrors, selectedLabel } = this.props;
    const {
      onCancelAddLabel,
      onCancelEditLabel,
      onEditLabel,
      onOsqueryTableSelect,
      onSaveAddLabel,
    } = this;

    if (isAddLabel) {
      return (
        <div className="body-wrap">
          <LabelForm
            onCancel={onCancelAddLabel}
            onOsqueryTableSelect={onOsqueryTableSelect}
            handleSubmit={onSaveAddLabel}
            serverErrors={labelErrors}
          />
        </div>
      );
    }

    if (isEditLabel) {
      return (
        <div className="body-wrap">
          <LabelForm
            formData={selectedLabel}
            onCancel={onCancelEditLabel}
            onOsqueryTableSelect={onOsqueryTableSelect}
            handleSubmit={onEditLabel}
            isEdit
            serverErrors={labelErrors}
          />
        </div>
      );
    }

    return false;
  };

  renderSidePanel = () => {
    let SidePanel;
    const {
      isAddLabel,
      labels,
      selectedOsqueryTable,
      statusLabels,
      canAddNewLabels,
    } = this.props;
    const {
      onAddLabelClick,
      onLabelClick,
      onOsqueryTableSelect,
      getLabelSelected,
      getStatusSelected,
    } = this;

    if (isAddLabel) {
      SidePanel = (
        <QuerySidePanel
          key="query-side-panel"
          onOsqueryTableSelect={onOsqueryTableSelect}
          selectedOsqueryTable={selectedOsqueryTable}
        />
      );
    } else {
      SidePanel = (
        <HostSidePanel
          key="hosts-side-panel"
          labels={labels}
          onAddLabelClick={onAddLabelClick}
          onLabelClick={onLabelClick}
          selectedFilter={getLabelSelected() || getStatusSelected()}
          statusLabels={statusLabels}
          canAddNewLabel={canAddNewLabels}
        />
      );
    }

    return SidePanel;
  };

  renderStatusDropdown = () => {
    const { handleStatusDropdownChange, getStatusSelected } = this;

    return (
      <Dropdown
        value={getStatusSelected() || ALL_HOSTS_LABEL}
        className={`${baseClass}__status_dropdown`}
        options={HOST_SELECT_STATUSES}
        searchable={false}
        onChange={handleStatusDropdownChange}
      />
    );
  };

  renderTable = () => {
    const { config, currentUser, selectedFilters, selectedLabel } = this.props;
    const {
      hiddenColumns,
      isAllMatchingHostsSelected,
      hosts,
      isHostsLoading,
    } = this.state;
    const {
      onTableQueryChange,
      onEditColumnsClick,
      onTransferToTeamClick,
      toggleAllMatchingHosts,
      renderStatusDropdown,
      getStatusSelected,
    } = this;

    // The data has not been fetched yet.
    if (selectedFilters.length === 0 || selectedLabel === undefined)
      return null;

    // Hosts have not been set up for this instance yet.
    if (getStatusSelected() === ALL_HOSTS_LABEL && selectedLabel.count === 0) {
      return <NoHosts />;
    }

    return (
      <TableContainer
        columns={generateVisibleTableColumns(
          hiddenColumns,
          config,
          currentUser
        )}
        data={hosts}
        isLoading={isHostsLoading}
        manualSortBy
        defaultSortHeader={"hostname"}
        defaultSortDirection={"asc"}
        actionButtonText={"Edit columns"}
        actionButtonIcon={EditColumnsIcon}
        actionButtonVariant={"text-icon"}
        additionalQueries={JSON.stringify(selectedFilters)}
        inputPlaceHolder={"Search hostname, UUID, serial number, or IPv4"}
        onActionButtonClick={onEditColumnsClick}
        onPrimarySelectActionClick={onTransferToTeamClick}
        primarySelectActionButtonText={"Transfer to team"}
        onQueryChange={onTableQueryChange}
        resultsTitle={"hosts"}
        emptyComponent={EmptyHosts}
        showMarkAllPages
        isAllPagesSelected={isAllMatchingHostsSelected}
        toggleAllPagesSelected={toggleAllMatchingHosts}
        searchable
        customControl={renderStatusDropdown}
      />
    );
  };

  render() {
    const {
      renderForm,
      renderHeader,
      renderSidePanel,
      renderAddHostModal,
      renderDeleteLabelModal,
      renderTable,
      renderEditColumnsModal,
      renderTransferHostModal,
      onAddHostClick,
    } = this;
    const {
      isAddLabel,
      isEditLabel,
      loadingLabels,
      canAddNewHosts,
    } = this.props;

    return (
      <div className="has-sidebar">
        {renderForm()}
        {!isAddLabel && !isEditLabel && (
          <div className={`${baseClass} body-wrap`}>
            <div className="header-wrap">
              {renderHeader()}
              {canAddNewHosts ? (
                <Button
                  onClick={onAddHostClick}
                  className={`${baseClass}__add-hosts button button--brand`}
                >
                  <span>Add new host</span>
                </Button>
              ) : null}
            </div>
            {renderTable()}
          </div>
        )}
        {!loadingLabels && renderSidePanel()}
        {renderAddHostModal()}
        {renderEditColumnsModal()}
        {renderDeleteLabelModal()}
        {renderTransferHostModal()}
      </div>
    );
  }
}

const mapStateToProps = (state, { location, params }) => {
  const { active_label: activeLabel, label_id: labelID } = params;
  const selectedFilters = [];

  labelID && selectedFilters.push(`${LABEL_SLUG_PREFIX}${labelID}`);
  activeLabel && selectedFilters.push(activeLabel);
  // "all-hosts" should always be alone
  !labelID && !activeLabel && selectedFilters.push(ALL_HOSTS_LABEL);

  const { status_labels: statusLabels } = state.components.ManageHostsPage;
  const labelEntities = entityGetter(state).get("labels");
  const { entities: labels } = labelEntities;

  // eqivalent to old way => const selectedFilter = labelID ? `labels/${labelID}` : activeLabelSlug;
  const slugToFind =
    (selectedFilters.length > 0 &&
      selectedFilters.find((f) => f.includes(LABEL_SLUG_PREFIX))) ||
    selectedFilters[0];
  const selectedLabel = labelEntities.findBy(
    { slug: slugToFind },
    { ignoreCase: true }
  );

  const isAddLabel = location.hash === NEW_LABEL_HASH;
  const isEditLabel = location.hash === EDIT_LABEL_HASH;

  const { selectedOsqueryTable } = state.components.QueryPages;
  const { errors: labelErrors, loading: loadingLabels } = state.entities.labels;
  const enrollSecret = state.app.enrollSecret;
  const config = state.app.config;

  const { loading: loadingHosts } = state.entities.hosts;

  const currentUser = state.auth.user;
  const canAddNewHosts =
    permissionUtils.isGlobalAdmin(currentUser) ||
    permissionUtils.isGlobalMaintainer(currentUser) ||
    permissionUtils.isAnyTeamMaintainer(currentUser);
  const canAddNewLabels =
    permissionUtils.isGlobalAdmin(currentUser) ||
    permissionUtils.isGlobalMaintainer(currentUser);
  const isGlobalAdmin = permissionUtils.isGlobalAdmin(currentUser);
  const isBasicTier = permissionUtils.isBasicTier(config);
  const teams = memoizedGetEntity(state.entities.teams.data);

  return {
    selectedFilters,
    isAddLabel,
    isEditLabel,
    labelErrors,
    labels,
    loadingLabels,
    enrollSecret,
    selectedLabel,
    selectedOsqueryTable,
    statusLabels,
    config,
    currentUser,
    loadingHosts,
    canAddNewHosts,
    canAddNewLabels,
    isGlobalAdmin,
    isBasicTier,
    teams,
  };
};

export default connect(mapStateToProps)(ManageHostsPage);
