import React, { PureComponent } from "react";
import PropTypes from "prop-types";
import AceEditor from "react-ace";
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
import hostInterface from "interfaces/host";
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
import {
  getLabels,
  getHosts,
} from "redux/nodes/components/ManageHostsPage/actions";
import PATHS from "router/paths";
import deepDifference from "utilities/deep_difference";

import hostActions2 from "services/entities/hosts";

import permissionUtils from "utilities/permissions";
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

const NEW_LABEL_HASH = "#new_label";
const EDIT_LABEL_HASH = "#edit_label";
const baseClass = "manage-hosts";

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
    selectedFilter: PropTypes.string,
    selectedLabel: labelInterface,
    selectedOsqueryTable: osqueryTableInterface,
    statusLabels: statusLabelsInterface,
    // hosts: PropTypes.arrayOf(hostInterface),
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
    // hosts: [],
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
    const { dispatch, selectedFilter } = this.props;
    dispatch(push(`${PATHS.MANAGE_HOSTS}/${selectedFilter}${NEW_LABEL_HASH}`));
  };

  onEditLabelClick = (evt) => {
    evt.preventDefault();
    const { dispatch, selectedFilter } = this.props;
    dispatch(push(`${PATHS.MANAGE_HOSTS}/${selectedFilter}${EDIT_LABEL_HASH}`));
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
    const { dispatch, selectedFilter } = this.props;
    dispatch(push(`${PATHS.MANAGE_HOSTS}/${selectedFilter}`));
  };

  onCancelEditLabel = () => {
    const { dispatch, selectedFilter } = this.props;
    dispatch(push(`${PATHS.MANAGE_HOSTS}/${selectedFilter}`));
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
  onTableQueryChange = async (queryData) => {
    const { selectedFilter, dispatch } = this.props;
    const {
      pageIndex,
      pageSize,
      searchQuery,
      sortHeader,
      sortDirection,
    } = queryData;
    let sortBy = [];
    if (sortHeader !== "") {
      sortBy = [{ id: sortHeader, direction: sortDirection }];
    }

    // keep track as a local state to be used later
    this.setState({ searchQuery });

    try {
      const { hosts } = await hostActions2.loadAll({
        page: pageIndex,
        perPage: pageSize,
        selectedLabel: selectedFilter,
        globalFilter: searchQuery,
        sortBy,
      });

      this.setState({ hosts });
    } catch (error) {
      dispatch(
        renderFlash("error", "Sorry, we could not retrieve your hosts.")
      );
    }
  };

  onEditLabel = (formData) => {
    const { dispatch, selectedLabel, selectedFilter } = this.props;
    const updateAttrs = deepDifference(formData, selectedLabel);

    return dispatch(labelActions.update(selectedLabel, updateAttrs))
      .then(() => {
        dispatch(push(`${PATHS.MANAGE_HOSTS}/${selectedFilter}`));
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
      const { dispatch } = this.props;
      const { MANAGE_HOSTS } = PATHS;
      const { slug, type } = selectedLabel;
      const nextLocation =
        type === "all" ? MANAGE_HOSTS : `${MANAGE_HOSTS}/${slug}`;
      dispatch(push(nextLocation));
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
    const { toggleTransferHostModal, isAcceptableStatus } = this;
    const { dispatch, selectedFilter, selectedLabel } = this.props;
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

      if (isAcceptableStatus(selectedFilter)) {
        status = selectedFilter;
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
        dispatch(getHosts({ selectedLabel: selectedFilter, searchQuery }));
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
    // shouldSelect?: boolean
    const { isAllMatchingHostsSelected } = this.state;

    if (shouldSelect !== undefined) {
      this.setState({ isAllMatchingHostsSelected: shouldSelect });
    } else {
      this.setState({
        isAllMatchingHostsSelected: !isAllMatchingHostsSelected,
      });
    }
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
          <Button onClick={toggleDeleteLabelModal} variant="inverse">
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

  renderDeleteButton = () => {
    const { toggleDeleteLabelModal, onEditLabelClick } = this;
    const {
      selectedLabel: { type },
    } = this.props;

    if (type !== "custom") {
      return false;
    }

    return (
      <div className={`${baseClass}__label-actions`}>
        <Button onClick={onEditLabelClick} variant="inverse">
          Edit
        </Button>
        <Button onClick={toggleDeleteLabelModal} variant="inverse">
          Delete
        </Button>
      </div>
    );
  };

  renderQuery = () => {
    const { selectedLabel } = this.props;
    const {
      slug,
      label_type: labelType,
      label_membership_type: membershipType,
      query,
    } = selectedLabel;

    if (membershipType === "manual" && labelType !== "builtin") {
      return (
        <h4 title="Manage manual labels with fleetctl">Manually managed</h4>
      );
    }

    if (!query || slug === "all-hosts") {
      return false;
    }

    return (
      <AceEditor
        editorProps={{ $blockScrolling: Infinity }}
        mode="fleet"
        minLines={1}
        maxLines={20}
        name="label-header"
        readOnly
        setOptions={{ wrap: true }}
        showGutter={false}
        showPrintMargin={false}
        theme="fleet"
        value={query}
        width="100%"
        fontSize={14}
      />
    );
  };

  renderHeader = () => {
    const { renderDeleteButton } = this;
    const { isAddLabel, selectedLabel } = this.props;

    if (!selectedLabel || isAddLabel) {
      return false;
    }

    const { description, display_text: displayText } = selectedLabel;

    const defaultDescription = "No description available.";

    return (
      <div className={`${baseClass}__header`}>
        <div className={`${baseClass}__text`}>
          <h1 className={`${baseClass}__title`}>
            <span>{displayText}</span>
          </h1>
          <div className={`${baseClass}__description`}>
            <p>{description || <em>{defaultDescription}</em>}</p>
          </div>
        </div>
        {renderDeleteButton()}
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
      selectedFilter,
      selectedOsqueryTable,
      statusLabels,
      canAddNewLabels,
    } = this.props;
    const { onAddLabelClick, onLabelClick, onOsqueryTableSelect } = this;

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
          selectedFilter={selectedFilter}
          statusLabels={statusLabels}
          canAddNewLabel={canAddNewLabels}
        />
      );
    }

    return SidePanel;
  };

  renderTable = () => {
    const {
      config,
      currentUser,
      selectedFilter,
      selectedLabel,
      // hosts,
      loadingHosts,
    } = this.props;
    const { hiddenColumns, isAllMatchingHostsSelected, hosts } = this.state;
    const {
      onTableQueryChange,
      onEditColumnsClick,
      onTransferToTeamClick,
      toggleAllMatchingHosts,
    } = this;

    // The data has not been fetched yet.
    if (selectedFilter === undefined || selectedLabel === undefined)
      return null;

    // Hosts have not been set up for this instance yet.
    if (selectedFilter === "all-hosts" && selectedLabel.count === 0) {
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
        isLoading={loadingHosts}
        sortByAPI={true}
        defaultSortHeader={"hostname"}
        defaultSortDirection={"asc"}
        actionButtonText={"Edit columns"}
        actionButtonIcon={EditColumnsIcon}
        actionButtonVariant={"text-icon"}
        additionalQueries={JSON.stringify([selectedFilter])}
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
      renderQuery,
      renderTable,
      renderEditColumnsModal,
      renderTransferHostModal,
      onAddHostClick,
    } = this;
    const {
      isAddLabel,
      isEditLabel,
      loadingLabels,
      selectedLabel,
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
            {selectedLabel && renderQuery()}
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
  const activeLabelSlug = activeLabel || "all-hosts";
  const selectedFilter = labelID ? `labels/${labelID}` : activeLabelSlug;

  const { status_labels: statusLabels } = state.components.ManageHostsPage;
  const labelEntities = entityGetter(state).get("labels");
  const { entities: labels } = labelEntities;
  const isAddLabel = location.hash === NEW_LABEL_HASH;
  const isEditLabel = location.hash === EDIT_LABEL_HASH;
  const selectedLabel = labelEntities.findBy(
    { slug: selectedFilter },
    { ignoreCase: true }
  );
  const { selectedOsqueryTable } = state.components.QueryPages;
  const { errors: labelErrors, loading: loadingLabels } = state.entities.labels;
  const enrollSecret = state.app.enrollSecret;
  const config = state.app.config;

  // NOTE: good opportunity for performance optimisation here later. This currently
  // always generates a new array of hosts, when it could memoized version of the list.
  // const { entities: hosts } = entityGetter(state).get("hosts");

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
    selectedFilter,
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
    // hosts,
    loadingHosts,
    canAddNewHosts,
    canAddNewLabels,
    isGlobalAdmin,
    isBasicTier,
    teams,
  };
};

export default connect(mapStateToProps)(ManageHostsPage);
