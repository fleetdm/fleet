import React, { PureComponent } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import { push, goBack } from "react-router-redux";
import { find, isEmpty, isEqual, memoize, omit } from "lodash";

import Button from "components/buttons/Button";
import Dropdown from "components/forms/fields/Dropdown";
import configInterface from "interfaces/config";
import HostSidePanel from "components/side_panels/HostSidePanel";
import LabelForm from "components/forms/LabelForm";
import Modal from "components/modals/Modal";
import QuerySidePanel from "components/side_panels/QuerySidePanel";
import TableContainer from "components/TableContainer";
import TableDataError from "components/TableDataError";
import labelInterface from "interfaces/label";
import teamInterface from "interfaces/team";
import userInterface from "interfaces/user";
import statusLabelsInterface from "interfaces/status_labels";
import { renderFlash } from "redux/nodes/notifications/actions";
import labelActions from "redux/nodes/entities/labels/actions";
import teamActions from "redux/nodes/entities/teams/actions";
import hostActions from "redux/nodes/entities/hosts/actions";
import entityGetter, { memoizedGetEntity } from "redux/utilities/entityGetter";
import { getLabels } from "redux/nodes/components/ManageHostsPage/actions";
import PATHS from "router/paths";
import deepDifference from "utilities/deep_difference";
import { QueryContext } from "context/query";

import hostAPI from "services/entities/hosts";
import policiesClient from "services/entities/policies";

import permissionUtils from "utilities/permissions";
import sortUtils from "utilities/sort";

import {
  PLATFORM_LABEL_DISPLAY_NAMES,
  PolicyResponse,
} from "utilities/constants";
import { getNextLocationPath } from "./helpers";
import {
  defaultHiddenColumns,
  generateVisibleTableColumns,
  generateAvailableTableHeaders,
} from "./HostTableConfig";
import EnrollSecretModal from "./components/EnrollSecretModal";
import AddHostModal from "./components/AddHostModal";
import NoHosts from "./components/NoHosts";
import EmptyHosts from "./components/EmptyHosts";
import PoliciesFilter from "./components/PoliciesFilter";
import EditColumnsModal from "./components/EditColumnsModal/EditColumnsModal";
import TransferHostModal from "./components/TransferHostModal";
import EditColumnsIcon from "../../../../assets/images/icon-edit-columns-16x16@2x.png";
import PencilIcon from "../../../../assets/images/icon-pencil-14x14@2x.png";
import TrashIcon from "../../../../assets/images/icon-trash-14x14@2x.png";
import CloseIcon from "../../../../assets/images/icon-close-fleet-black-16x16@2x.png";

const baseClass = "manage-hosts";

const NEW_LABEL_HASH = "#new_label";
const EDIT_LABEL_HASH = "#edit_label";
const ALL_HOSTS_LABEL = "all-hosts";
const LABEL_SLUG_PREFIX = "labels/";

const DEFAULT_SORT_HEADER = "hostname";
const DEFAULT_SORT_DIRECTION = "asc";

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
    queryParams: PropTypes.objectOf(
      PropTypes.oneOfType([PropTypes.string, PropTypes.number])
    ),
    routeTemplate: PropTypes.string,
    routeParams: PropTypes.objectOf(
      PropTypes.oneOfType([PropTypes.string, PropTypes.number])
    ),
    selectedFilters: PropTypes.arrayOf(PropTypes.string),
    selectedLabel: labelInterface,
    selectedTeam: PropTypes.oneOfType([PropTypes.number, PropTypes.string]),
    statusLabels: statusLabelsInterface,
    canAddNewHosts: PropTypes.bool,
    canEnrollHosts: PropTypes.bool,
    canAddNewLabels: PropTypes.bool,
    teams: PropTypes.arrayOf(teamInterface),
    isGlobalAdmin: PropTypes.bool,
    isOnGlobalTeam: PropTypes.bool,
    isPremiumTier: PropTypes.bool,
    currentUser: userInterface,
    policyId: PropTypes.oneOfType([PropTypes.number, PropTypes.string]),
    policyResponse: PropTypes.oneOf([
      PolicyResponse.PASSING,
      PolicyResponse.FAILING,
    ]),
  };

  static defaultProps = {
    loadingLabels: false,
  };

  static contextType = QueryContext; // eslint-disable-line react/sort-comp
  
  constructor(props) {
    super(props);

    // For now we persist hidden columns using localstorage. May do server side
    // persistence later.
    const storedHiddenColumns = JSON.parse(
      localStorage.getItem("hostHiddenColumns")
    );

    // Unpack sort params from url query string and use to initialize state of sortBy
    // so that table component sort defaults will not override url params
    const initialSortBy = (() => {
      let id = DEFAULT_SORT_HEADER;
      let direction = DEFAULT_SORT_DIRECTION;

      if (this.props.queryParams) {
        const { order_key, order_direction } = this.props.queryParams;
        id = order_key || id;
        direction = order_direction || direction;
      }

      return [{ id, direction }];
    })();

    this.state = {
      labelQueryText: "",
      showAddHostModal: false,
      showEnrollSecretModal: false,
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
      isHostsLoading: true,
      hostErrors: false,
      sortBy: initialSortBy,
      isConfigLoaded: !isEmpty(this.props.config),
      isTeamsLoaded: !isEmpty(this.props.teams),
      isTeamsLoading: false,
      policyName: null,
      selectedOsqueryTable: null,
      setSelectedOsqueryTable: null,
    };
  }

  componentDidMount() {
    const { dispatch, policyId } = this.props;
    dispatch(getLabels());

    if (policyId) {
      policiesClient
        .load(policyId)
        .then((response) => {
          const { query_name: policyName } = response.policy;
          this.setState({ policyName });
        })
        .catch((err) => {
          console.log(err);
          // dispatch(
          //   renderFlash(
          //     "error",
          //     "Sorry, we could not retrieve the policy name."
          //   )
          // );
        });
    }

    // TODO: Very temporary until this component becomes functional
    // this was so we could remove redux for selectedOsqueryTable - 8/31/21 - MP
    /* eslint-disable react/no-did-mount-set-state */
    const { selectedOsqueryTable, setSelectedOsqueryTable } = this.context;
    this.setState({ selectedOsqueryTable, setSelectedOsqueryTable });
    /* eslint-enable no-alert, react/no-did-mount-set-state */
  }

  componentWillReceiveProps() {
    const { config, dispatch, isPremiumTier } = this.props;
    const { isConfigLoaded, isTeamsLoaded, isTeamsLoading } = this.state;
    if (!isConfigLoaded && !isEmpty(config)) {
      this.setState({ isConfigLoaded: true });
    }
    if (isConfigLoaded && isPremiumTier && !isTeamsLoaded && !isTeamsLoading) {
      this.setState({ isTeamsLoading: true });
      dispatch(teamActions.loadAll({}))
        .then(() => {
          this.setState({
            isTeamsLoaded: true,
          });
        })
        .catch((error) => {
          renderFlash(
            "error",
            "An error occured loading teams. Please try again."
          );
          console.log(error);
          this.setState({
            isTeamsLoaded: false,
          });
        })
        .finally(() => {
          this.setState({
            isTeamsLoading: false,
          });
        });
    }
  }

  // TODO: Very temporary until this component becomes functional
  // this was so we could remove redux for selectedOsqueryTable - 8/31/21 - MP
  /* eslint-disable react/no-did-mount-set-state */
  componentDidUpdate() {
    if (
      !isEqual(
        this.context.selectedOsqueryTable,
        this.state.selectedOsqueryTable
      )
    ) {
      const { selectedOsqueryTable } = this.context;
      this.setState({ selectedOsqueryTable });
    }
  }
  /* eslint-enable react/no-did-mount-set-state */

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
    const { dispatch } = this.props;
    dispatch(goBack());
  };

  onCancelEditLabel = () => {
    const { dispatch } = this.props;
    dispatch(goBack());
  };

  onShowEnrollSecretClick = (evt) => {
    evt.preventDefault();
    const { toggleEnrollSecretModal } = this;
    toggleEnrollSecretModal();
  };

  onAddHostClick = (evt) => {
    evt.preventDefault();
    const { toggleAddHostModal } = this;
    toggleAddHostModal();
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
    const { getValidatedTeamId, retrieveHosts } = this;
    const {
      dispatch,
      policyId,
      policyResponse,
      routeTemplate,
      routeParams,
      selectedFilters,
      selectedTeam,
    } = this.props;

    const teamId = getValidatedTeamId(selectedTeam);

    let sortBy = this.state.sortBy;
    if (sortHeader !== "") {
      sortBy = [
        { id: sortHeader, direction: sortDirection || DEFAULT_SORT_DIRECTION },
      ];
    } else if (!sortBy.length) {
      sortBy = [{ id: DEFAULT_SORT_HEADER, direction: DEFAULT_SORT_DIRECTION }];
    }
    this.setState({
      sortBy,
    });

    // keep track as a local state to be used later
    this.setState({ searchQuery });

    retrieveHosts({
      page: pageIndex,
      perPage: pageSize,
      selectedLabels: selectedFilters,
      globalFilter: searchQuery,
      sortBy,
      teamId: selectedTeam,
      policyId,
      policyResponse,
    });

    // Rebuild queryParams to dispatch new browser location to react-router
    const queryParams = {};
    if (!isEmpty(searchQuery)) {
      queryParams.query = searchQuery;
    }
    if (sortBy[0] && sortBy[0].id) {
      queryParams.order_key = sortBy[0].id;
    } else {
      queryParams.order_key = DEFAULT_SORT_HEADER;
    }
    if (sortBy[0] && sortBy[0].direction) {
      queryParams.order_direction = sortBy[0].direction;
    } else {
      queryParams.order_direction = DEFAULT_SORT_DIRECTION;
    }
    if (teamId) {
      queryParams.team_id = teamId;
    }
    if (policyId) {
      queryParams.policy_id = policyId;
    }
    if (policyResponse) {
      queryParams.policy_response = policyResponse;
    }

    dispatch(
      push(
        getNextLocationPath({
          pathPrefix: PATHS.MANAGE_HOSTS,
          routeTemplate,
          routeParams,
          queryParams,
        })
      )
    );
  };

  onEditLabel = (formData) => {
    const { dispatch, selectedLabel } = this.props;
    const updateAttrs = deepDifference(formData, selectedLabel);

    return dispatch(labelActions.update(selectedLabel, updateAttrs))
      .then(() => {
        dispatch(goBack());

        // TODO flash messages are not visible seemingly because of page renders
        dispatch(
          renderFlash(
            "success",
            "Label updated. Try refreshing this page in just a moment to see the updated host count for your label."
          )
        );
        return false;
      })
      .catch((err) => {
        console.log(err);
        dispatch(
          renderFlash("error", "Could not create label. Please try again.")
        );
      });
  };

  onLabelClick = (selectedLabel) => {
    return (evt) => {
      evt.preventDefault();

      const { handleLabelChange } = this;
      handleLabelChange(selectedLabel);
    };
  };

  onOsqueryTableSelect = (tableName) => {
    if (this.state.setSelectedOsqueryTable) {
      this.state.setSelectedOsqueryTable(tableName);
    }
    return false;
  };

  onSaveAddLabel = (formData) => {
    const { dispatch } = this.props;

    return dispatch(labelActions.create(formData))
      .then(() => {
        dispatch(push(PATHS.MANAGE_HOSTS));

        // TODO flash messages are not visible seemingly because of page renders
        dispatch(
          renderFlash(
            "success",
            "Label created. Try refreshing this page in just a moment to see the updated host count for your label."
          )
        );
        return false;
      })
      .catch((err) => {
        console.log(err);
        dispatch(
          renderFlash("error", "Could not create label. Please try again.")
        );
      });
  };

  onDeleteLabel = () => {
    const { toggleDeleteLabelModal } = this;
    const {
      dispatch,
      routeTemplate,
      routeParams,
      queryParams,
      selectedLabel,
    } = this.props;
    const { MANAGE_HOSTS } = PATHS;

    return dispatch(labelActions.destroy(selectedLabel))
      .then(() => {
        toggleDeleteLabelModal();
        dispatch(
          push(
            getNextLocationPath({
              pathPrefix: MANAGE_HOSTS,
              routeTemplate: routeTemplate.replace("/labels/:label_id", ""),
              routeParams,
              queryParams,
            })
          )
        );
        return false;
      })
      .catch((err) => {
        console.log(err);
        dispatch(
          renderFlash("error", "Could not delete label. Please try again.")
        );
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
    const {
      dispatch,
      selectedFilters,
      selectedLabel,
      selectedTeam: selectedTeamFilter,
      policyId,
      policyResponse,
    } = this.props;
    const {
      selectedHostIds,
      isAllMatchingHostsSelected,
      searchQuery,
      sortBy,
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
          sortBy,
          teamId: selectedTeamFilter,
          policyId,
          policyResponse,
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

  getSortedTeamOptions = memoize((teams) =>
    teams
      .map((team) => {
        return {
          disabled: false,
          label: team.name,
          value: team.id,
        };
      })
      .sort((a, b) => sortUtils.caseInsensitiveAsc(b.label, a.label))
  );

  getValidatedTeamId = (teamId) => {
    const { currentUser, isOnGlobalTeam, teams } = this.props;

    teamId = parseInt(teamId, 10);

    let currentUserTeams = [];
    if (isOnGlobalTeam) {
      currentUserTeams = teams;
    } else if (currentUser && currentUser.teams) {
      currentUserTeams = currentUser.teams;
    }

    const currentUserTeamIds = currentUserTeams.map((t) => t.id);

    const validatedTeamId =
      !isNaN(teamId) && teamId > 0 && currentUserTeamIds.includes(teamId)
        ? teamId
        : 0;

    return validatedTeamId;
  };

  generateTeamFilterDropdownOptions = (teams) => {
    const { currentUser, isOnGlobalTeam } = this.props;
    const { getSortedTeamOptions } = this;

    let currentUserTeams = [];
    if (isOnGlobalTeam) {
      currentUserTeams = teams;
    } else if (currentUser && currentUser.teams) {
      currentUserTeams = currentUser.teams;
    }

    const allTeamsOption = [
      {
        disabled: false,
        label: "All teams",
        value: 0,
      },
    ];

    const sortedCurrentUserTeamOptions = getSortedTeamOptions(currentUserTeams);

    return allTeamsOption.concat(sortedCurrentUserTeamOptions);
  };

  retrieveHosts = async (options = {}) => {
    const { dispatch } = this.props;
    const { getValidatedTeamId } = this;

    this.setState({ isHostsLoading: true });

    options = {
      ...options,
      teamId: getValidatedTeamId(options.teamId),
    };

    try {
      const { hosts } = await hostAPI.loadAll(options);
      this.setState({ hosts });
    } catch (error) {
      console.log(error);
      this.setState({ hostErrors: true });
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

  clearHostUpdates = () => {
    if (this.timeout) {
      global.window.clearTimeout(this.timeout);
      this.timeout = null;
    }
  };

  toggleEnrollSecretModal = () => {
    const { showEnrollSecretModal } = this.state;
    this.setState({ showEnrollSecretModal: !showEnrollSecretModal });
  };

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

  handleChangePoliciesFilter = (policyResponse) => {
    const {
      dispatch,
      policyId,
      routeTemplate,
      routeParams,
      queryParams,
      selectedFilters,
      selectedTeam,
    } = this.props;
    const { searchQuery, sortBy } = this.state;
    const { retrieveHosts } = this;

    retrieveHosts({
      globalFilter: searchQuery,
      policyId,
      policyResponse,
      selectedLabels: selectedFilters,
      sortBy,
      teamId: selectedTeam,
    });

    dispatch(
      push(
        getNextLocationPath({
          pathPrefix: PATHS.MANAGE_HOSTS,
          routeTemplate,
          routeParams,
          queryParams: Object.assign({}, queryParams, {
            policy_id: policyId,
            policy_response: policyResponse,
          }),
        })
      )
    );
  };

  handleClearPoliciesFilter = () => {
    const {
      dispatch,
      routeTemplate,
      routeParams,
      queryParams,
      selectedFilters,
      selectedTeam,
    } = this.props;
    const { searchQuery, sortBy } = this.state;
    const { retrieveHosts } = this;

    retrieveHosts({
      globalFilter: searchQuery,
      selectedLabels: selectedFilters,
      sortBy,
      teamId: selectedTeam,
    });
    dispatch(
      push(
        getNextLocationPath({
          pathPrefix: PATHS.MANAGE_HOSTS,
          routeTemplate,
          routeParams,
          queryParams: omit(queryParams, ["policy_id", "policy_response"]),
        })
      )
    );
  };

  // The handleChange method below is for the filter-by-team dropdown rather than the dropdown used in modals
  handleChangeSelectedTeamFilter = (selectedTeam) => {
    const {
      dispatch,
      policyId,
      policyResponse,
      selectedFilters,
      routeTemplate,
      routeParams,
      queryParams,
    } = this.props;
    const { searchQuery, sortBy } = this.state;
    const { getValidatedTeamId, retrieveHosts } = this;
    const { MANAGE_HOSTS } = PATHS;

    const teamIdParam = getValidatedTeamId(selectedTeam);

    const hostsOptions = {
      teamId: teamIdParam,
      selectedLabels: selectedFilters,
      globalFilter: searchQuery,
      sortBy,
      policyId,
      policyResponse,
    };
    retrieveHosts(hostsOptions);

    const nextLocation = getNextLocationPath({
      pathPrefix: MANAGE_HOSTS,
      routeTemplate,
      routeParams,
      queryParams: !teamIdParam
        ? omit(queryParams, "team_id")
        : Object.assign({}, queryParams, { team_id: teamIdParam }),
    });
    dispatch(push(nextLocation));
  };

  handleLabelChange = ({ slug }) => {
    const { dispatch, queryParams, selectedFilters } = this.props;
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

    //  Non-status labels are not compatible with policies so omit policy params from next location
    let newQueryParams = queryParams;
    if (newFilters.find((f) => f.includes(LABEL_SLUG_PREFIX))) {
      newQueryParams = omit(newQueryParams, ["policy_id", "policy_response"]);
    }

    dispatch(
      push(
        getNextLocationPath({
          pathPrefix: isAllHosts
            ? MANAGE_HOSTS
            : `${MANAGE_HOSTS}/${newFilters.join("/")}`,
          queryParams: newQueryParams,
        })
      )
    );
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

  renderTeamsFilterDropdown = () => {
    const { isPremiumTier, selectedTeam, teams } = this.props;
    const { isConfigLoaded, isTeamsLoaded } = this.state;
    const {
      generateTeamFilterDropdownOptions,
      getValidatedTeamId,
      handleChangeSelectedTeamFilter,
    } = this;

    if (!isConfigLoaded || (isPremiumTier && !isTeamsLoaded)) {
      return null;
    }

    if (!isPremiumTier) {
      return <h1>Hosts</h1>;
    }

    const teamOptions = generateTeamFilterDropdownOptions(teams);
    const selectedTeamId = getValidatedTeamId(selectedTeam);

    return (
      <div>
        <Dropdown
          value={selectedTeamId}
          placeholder={"All teams"}
          className={`${baseClass}__team-dropdown`}
          options={teamOptions}
          searchable={false}
          onChange={(newSelectedValue) =>
            handleChangeSelectedTeamFilter(newSelectedValue)
          }
        />
      </div>
    );
  };

  renderPoliciesFilterBlock = () => {
    const { policyId, policyResponse } = this.props;
    const { policyName } = this.state;
    const { handleClearPoliciesFilter, handleChangePoliciesFilter } = this;

    return (
      <div className={`${baseClass}__policies-filter-block`}>
        <PoliciesFilter
          policyId={policyId}
          policyResponse={policyResponse}
          onChange={handleChangePoliciesFilter}
        />
        <p>{policyName}</p>
        <Button onClick={handleClearPoliciesFilter} variant={"text-icon"}>
          <img src={CloseIcon} alt="Remove policy filter" />
        </Button>
      </div>
    );
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

  renderEnrollSecretModal = () => {
    const { toggleEnrollSecretModal } = this;
    const { showEnrollSecretModal } = this.state;
    const { canEnrollHosts, teams, selectedTeam, isPremiumTier } = this.props;

    if (!canEnrollHosts || !showEnrollSecretModal) {
      return null;
    }

    return (
      <Modal
        title="Enroll secret"
        onExit={toggleEnrollSecretModal}
        className={`${baseClass}__enroll-secret-modal`}
      >
        <EnrollSecretModal
          selectedTeam={selectedTeam}
          teams={teams}
          onReturnToApp={toggleEnrollSecretModal}
          isPremiumTier={isPremiumTier}
        />
      </Modal>
    );
  };

  renderAddHostModal = () => {
    const { toggleAddHostModal, onChangeTeam } = this;
    const { showAddHostModal } = this.state;
    const { config, currentUser, canAddNewHosts, teams } = this.props;

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
          config={config}
          currentUser={currentUser}
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
    description = "",
    display_text: displayText = "",
    label_type: labelType = "",
  }) => {
    const { onEditLabelClick, toggleDeleteLabelModal } = this;

    displayText = PLATFORM_LABEL_DISPLAY_NAMES[displayText] || displayText;

    return (
      <div className={`${baseClass}__label-block`}>
        <div className="title">
          <span>{displayText}</span>
          {labelType !== "builtin" && (
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
    const { renderTeamsFilterDropdown } = this;

    return (
      <div className={`${baseClass}__header`}>
        <div className={`${baseClass}__text`}>
          {renderTeamsFilterDropdown()}
        </div>
      </div>
    );
  };

  renderLabelOrPolicyBlock = () => {
    const { renderHeaderLabelBlock, renderPoliciesFilterBlock } = this;
    const { policyId, selectedLabel } = this.props;
    const type = selectedLabel?.type;

    if (policyId || selectedLabel) {
      return (
        <div className={`${baseClass}__labels-policies-wrap`}>
          {policyId && renderPoliciesFilterBlock()}
          {!policyId &&
            type !== "all" &&
            type !== "status" &&
            selectedLabel &&
            renderHeaderLabelBlock(selectedLabel)}
        </div>
      );
    }
    return null;
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
    const { isAddLabel, labels, statusLabels, canAddNewLabels } = this.props;
    const { selectedOsqueryTable } = this.state;
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
      hostErrors,
      isConfigLoaded,
      sortBy,
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
    if (
      !isConfigLoaded ||
      selectedFilters.length === 0 ||
      selectedLabel === undefined
    ) {
      return null;
    }

    if (hostErrors) {
      return <TableDataError />;
    }

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
        defaultSortHeader={(sortBy[0] && sortBy[0].id) || DEFAULT_SORT_HEADER}
        defaultSortDirection={
          (sortBy[0] && sortBy[0].direction) || DEFAULT_SORT_DIRECTION
        }
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
      renderLabelOrPolicyBlock,
      renderSidePanel,
      renderAddHostModal,
      renderEnrollSecretModal,
      renderDeleteLabelModal,
      renderTable,
      renderEditColumnsModal,
      renderTransferHostModal,
      onShowEnrollSecretClick,
      onAddHostClick,
    } = this;
    const {
      isAddLabel,
      isEditLabel,
      loadingLabels,
      canAddNewHosts,
      isPremiumTier,
      canEnrollHosts,
    } = this.props;
    const { isConfigLoaded, isTeamsLoaded } = this.state;

    return (
      <div className="has-sidebar">
        {renderForm()}
        {!isAddLabel && !isEditLabel && (
          <div className={`${baseClass} body-wrap`}>
            <div className="header-wrap">
              {renderHeader()}
              <div className={`${baseClass} button-wrap`}>
                {canEnrollHosts && (
                  <Button
                    onClick={onShowEnrollSecretClick}
                    className={`${baseClass}__enroll-hosts button`}
                    variant="inverse"
                  >
                    <span>Show enroll secret</span>
                  </Button>
                )}
                {canAddNewHosts && (
                  <Button
                    onClick={onAddHostClick}
                    className={`${baseClass}__add-hosts button button--brand`}
                  >
                    <span>Add new host</span>
                  </Button>
                )}
              </div>
            </div>
            {renderLabelOrPolicyBlock()}
            {isConfigLoaded &&
              (!isPremiumTier || isTeamsLoaded) &&
              renderTable()}
          </div>
        )}
        {!loadingLabels && renderSidePanel()}
        {renderEnrollSecretModal()}
        {renderAddHostModal()}
        {renderEditColumnsModal()}
        {renderDeleteLabelModal()}
        {renderTransferHostModal()}
      </div>
    );
  }
}

const mapStateToProps = (state, ownProps) => {
  const { location, params, route, routeParams } = ownProps;
  const locationPath = location.path;
  const queryParams = location.query;
  const routeTemplate = route && route.path ? route.path : "";

  const policyId = queryParams?.policy_id;
  const policyResponse = queryParams?.policy_response;

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

  const { errors: labelErrors, loading: loadingLabels } = state.entities.labels;
  const config = state.app.config;

  const teams = memoizedGetEntity(state.entities.teams.data);

  // If there is no team_id, set selectedTeam to 0 so dropdown defaults to "All teams"
  const selectedTeam = location.query?.team_id || 0;

  const currentUser = state.auth.user;
  const canAddNewHosts =
    permissionUtils.isGlobalAdmin(currentUser) ||
    permissionUtils.isGlobalMaintainer(currentUser) ||
    permissionUtils.isAnyTeamMaintainer(currentUser);
  const canEnrollHosts =
    permissionUtils.isGlobalAdmin(currentUser) ||
    permissionUtils.isGlobalMaintainer(currentUser) ||
    (permissionUtils.isAnyTeamMaintainer(currentUser) && selectedTeam !== 0);
  const canAddNewLabels =
    permissionUtils.isGlobalAdmin(currentUser) ||
    permissionUtils.isGlobalMaintainer(currentUser);
  const isGlobalAdmin = permissionUtils.isGlobalAdmin(currentUser);
  const isOnGlobalTeam = permissionUtils.isOnGlobalTeam(currentUser);
  const isPremiumTier = permissionUtils.isPremiumTier(config);

  return {
    selectedFilters,
    locationPath,
    queryParams,
    routeParams,
    routeTemplate,
    isAddLabel,
    isEditLabel,
    labelErrors,
    labels,
    loadingLabels,
    selectedLabel,
    statusLabels,
    config,
    currentUser,
    canAddNewHosts,
    canEnrollHosts,
    canAddNewLabels,
    isGlobalAdmin,
    isOnGlobalTeam,
    isPremiumTier,
    teams,
    selectedTeam,
    policyId,
    policyResponse,
  };
};

export default connect(mapStateToProps)(ManageHostsPage);
