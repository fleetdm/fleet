import React, { useState, useContext, useEffect } from "react";
import { useDispatch } from "react-redux";
import { useQuery } from "react-query";
import { InjectedRouter, Params } from "react-router/lib/Router";
import { RouteProps } from "react-router/lib/Route";
import { find, isEmpty, isEqual, memoize, omit } from "lodash"

import labelsAPI from "services/entities/labels";
import statusLabelsAPI from "services/entities/statusLabels";
import teamsAPI from "services/entities/teams";
import policiesAPI from "services/entities/policies";
import hostsAPI, { 
  IHostLoadOptions,
  ISortOption,
} from "services/entities/hosts";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { QueryContext } from "context/query";
import { ILabel, ILabelFormData } from "interfaces/label";
import { IStatusLabels } from "interfaces/status_labels";
import { ITeam } from "interfaces/team";
import { IHost } from "interfaces/host";
import { IPolicy } from "interfaces/policy";
import { useDeepEffect } from "utilities/hooks"; // @ts-ignore
import deepDifference from "utilities/deep_difference";
import {
  PLATFORM_LABEL_DISPLAY_NAMES,
  PolicyResponse,
} from "utilities/constants"; // @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";

import Button from "components/buttons/Button"; // @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import HostSidePanel from "components/side_panels/HostSidePanel"; // @ts-ignore
import LabelForm from "components/forms/LabelForm";
import Modal from "components/modals/Modal";
import QuerySidePanel from "components/side_panels/QuerySidePanel";
import TableContainer from "components/TableContainer";
import TableDataError from "components/TableDataError";

import {
  defaultHiddenColumns,
  generateVisibleTableColumns,
  generateAvailableTableHeaders,
} from "./HostTableConfig";
import {
  NEW_LABEL_HASH,
  EDIT_LABEL_HASH,
  ALL_HOSTS_LABEL,
  LABEL_SLUG_PREFIX,
  DEFAULT_SORT_HEADER,
  DEFAULT_SORT_DIRECTION,
  HOST_SELECT_STATUSES,
  isAcceptableStatus,
  getNextLocationPath,
  generateTeamFilterDropdownOptions,
  getValidatedTeamId,
} from "./helpers";
import EnrollSecretModal from "./components/EnrollSecretModal"; // @ts-ignore
import AddHostModal from "./components/AddHostModal";
import NoHosts from "./components/NoHosts";
import EmptyHosts from "./components/EmptyHosts";
import PoliciesFilter from "./components/PoliciesFilter"; // @ts-ignore
import EditColumnsModal from "./components/EditColumnsModal/EditColumnsModal";
import TransferHostModal from "./components/TransferHostModal";
import EditColumnsIcon from "../../../../assets/images/icon-edit-columns-16x16@2x.png";
import PencilIcon from "../../../../assets/images/icon-pencil-14x14@2x.png";
import TrashIcon from "../../../../assets/images/icon-trash-14x14@2x.png";
import CloseIcon from "../../../../assets/images/icon-close-fleet-black-16x16@2x.png";

interface IManageHostsProps {
  route: RouteProps;
  router: InjectedRouter;
  params: Params;
  location: any; // no type in react-router v3
}

interface ILabelsResponse {
  labels: ILabel[];
}

interface ITeamsResponse {
  teams: ITeam[];
}

interface ITableQueryProps {
  pageIndex: number;
  pageSize: number;
  searchQuery: string;
  sortHeader: string;
  sortDirection: string;
}

const baseClass = "manage-hosts";

const ManageHostsPage = ({
  route,
  router,
  params: routeParams,
  location,
}: IManageHostsProps) => {
  const dispatch = useDispatch();
  const queryParams = location.query;
  const {
    currentUser,
    config,
    isGlobalAdmin,
    isGlobalMaintainer,
    isAnyTeamMaintainer,
    isTeamMaintainer,
    isOnGlobalTeam,
    isPremiumTier,
    currentTeam,
    setCurrentTeam,
  } = useContext(AppContext);
  const { selectedOsqueryTable, setSelectedOsqueryTable } = useContext(
    QueryContext
  );

  const storedHiddenColumns = JSON.parse(
    localStorage.getItem("hostHiddenColumns") || ""
  );

  const initialSortBy: ISortOption[] = (() => {
    let key = DEFAULT_SORT_HEADER;
    let direction = DEFAULT_SORT_DIRECTION;

    if (queryParams) {
      const { order_key, order_direction } = queryParams;
      key = order_key || key;
      direction = order_direction || direction;
    }

    return [{ key, direction }];
  })();

  // ========= states
  const [selectedLabel, setSelectedLabel] = useState<ILabel>();
  const [statusLabels, setStatusLabels] = useState<IStatusLabels>();
  const [showAddHostModal, setShowAddHostModal] = useState<boolean>(false);
  const [showEnrollSecretModal, setShowEnrollSecretModal] = useState<boolean>(false);
  const [showDeleteLabelModal, setShowDeleteLabelModal] = useState<boolean>(false);
  const [showEditColumnsModal, setShowEditColumnsModal] = useState<boolean>(false);
  const [showTransferHostModal, setShowTransferHostModal] = useState<boolean>(false);
  const [hiddenColumns, setHiddenColumns] = useState<string[]>(storedHiddenColumns || defaultHiddenColumns);
  const [selectedHostIds, setSelectedHostIds] = useState<number[]>([]);
  const [isAllMatchingHostsSelected, setIsAllMatchingHostsSelected] = useState<boolean>(false);
  const [searchQuery, setSearchQuery] = useState<string>("");
  const [hosts, setHosts] = useState<IHost[]>();
  const [isHostsLoading, setIsHostsLoading] = useState<boolean>(false);
  const [hasHostErrors, setHasHostErrors] = useState<boolean>(false);
  const [sortBy, setSortBy] = useState<ISortOption[]>(initialSortBy);
  const [policyName, setPolicyName] = useState<string>();
  const [tableQueryData, setTableQueryData] = useState<ITableQueryProps>();
  // ======== end states

  const isAddLabel = location.hash === NEW_LABEL_HASH;
  const isEditLabel = location.hash === EDIT_LABEL_HASH;
  const routeTemplate = route && route.path ? route.path : "";
  const policyId = queryParams?.policy_id;
  const policyResponse: PolicyResponse = queryParams?.policy_response;
  const { active_label: activeLabel, label_id: labelID } = routeParams;

  // ===== filter matching
  const selectedFilters: string[] = [];
  labelID && selectedFilters.push(`${LABEL_SLUG_PREFIX}${labelID}`);
  activeLabel && selectedFilters.push(activeLabel);
  !labelID && !activeLabel && selectedFilters.push(ALL_HOSTS_LABEL); // "all-hosts" should always be alone
  // ===== end filter matching
  
  const canAddNewHosts = isGlobalAdmin || isGlobalMaintainer || isAnyTeamMaintainer;
  const canEnrollHosts = isGlobalAdmin || isGlobalMaintainer || isTeamMaintainer;
  const canAddNewLabels = isGlobalAdmin || isGlobalMaintainer;

  const {
    isLoading: isLabelsLoading,
    data: labels,
    error: labelsError,
    refetch: refetchLabels,
  } = useQuery<ILabelsResponse, Error, ILabel[]>(
    ["labels"],
    () => labelsAPI.loadAll(),
    {
      select: (data: ILabelsResponse) => data.labels,
    }
  );

  // TODO: add counts to status dropdown
  useQuery<IStatusLabels, Error>(
    ["status labels"],
    () => statusLabelsAPI.getCounts(),
    {
      onSuccess: (statusLabels) => {
        setStatusLabels(statusLabels);
      },
    }
  );

  const { data: teams } = useQuery<ITeamsResponse, Error, ITeam[]>(
    ["teams"],
    () => teamsAPI.loadAll(),
    {
      enabled: isPremiumTier,
      select: (data: ITeamsResponse) => data.teams,
    }
  );

  useQuery<IPolicy, Error>(
    ["policy"],
    () => policiesAPI.load(policyId),
    {
      enabled: !!policyId,
      onSuccess: ({ query_name: policyName }) => {
        setPolicyName(policyName);
      },
    }
  );

  // triggered every time the route is changed
  // which means every filter click and text search
  useDeepEffect(() => {
    // set the team object in context
    const teamId = parseInt(queryParams?.team_id, 10) || 0;
    const selectedTeam = find(teams, ["id", teamId]);
    setCurrentTeam(selectedTeam);

    // set selected label
    const slugToFind =
      (selectedFilters.length > 0 &&
        selectedFilters.find((f) => f.includes(LABEL_SLUG_PREFIX))) ||
      selectedFilters[0];
    
    const selected = find(labels, ["slug", slugToFind]) as ILabel;
    setSelectedLabel(selected);

    // get the hosts
    let options: IHostLoadOptions = {
      selectedLabels: selectedFilters,
      globalFilter: searchQuery,
      sortBy,
      teamId: selectedTeam?.id,
      policyId,
      policyResponse,
    };
    
    if (tableQueryData) {
      options.page = tableQueryData.pageIndex;
      options.perPage = tableQueryData.pageSize;
    }
    
    retrieveHosts(options);
  }, [location, tableQueryData, labels]);

  const onAddLabelClick = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();
    router.push(`${PATHS.MANAGE_HOSTS}${NEW_LABEL_HASH}`);
  };

  const onEditLabelClick = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();
    router.push(`${PATHS.MANAGE_HOSTS}/${getLabelSelected()}${EDIT_LABEL_HASH}`);
  };

  const onEditColumnsClick = () => {
    setShowEditColumnsModal(true);
  };

  const onCancelColumns = () => {
    setShowEditColumnsModal(false);
  };

  const onSaveColumns = (newHiddenColumns: string[]) => {
    localStorage.setItem("hostHiddenColumns", JSON.stringify(newHiddenColumns));
    setHiddenColumns(newHiddenColumns);
    setShowEditColumnsModal(false);
  };

  const onCancelAddLabel = () => {
    router.goBack();
  };

  const onCancelEditLabel = () => {
    router.goBack();
  };

  const onShowEnrollSecretClick = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();
    toggleEnrollSecretModal();
  };

  const onAddHostClick = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();
    toggleAddHostModal();
  };

  // NOTE: this is called once on the initial rendering. The initial render of
  // the TableContainer child component will call this handler.
  const onTableQueryChange = async (newTableQuery: ITableQueryProps) => {
    if (isEqual(newTableQuery, tableQueryData)) {
      return false;
    }
    
    setTableQueryData({...newTableQuery});

    const {
      searchQuery: searchText,
      sortHeader,
      sortDirection,
    } = newTableQuery;
    const teamId = getValidatedTeamId(
      teams || [],
      currentTeam?.id as number,
      currentUser,
      isOnGlobalTeam as boolean,
    );
    
    let sort = sortBy;
    if (!!sortHeader) {
      sort = [
        { key: sortHeader, direction: sortDirection || DEFAULT_SORT_DIRECTION },
      ];
    } else if (!sortBy.length) {
      sort = [
        { key: DEFAULT_SORT_HEADER, direction: DEFAULT_SORT_DIRECTION }
      ];
    }
    
    if (!isEqual(sort, sortBy)) {
      setSortBy([...sort]);
    }

    if (!isEqual(searchText, searchQuery)) {
      setSearchQuery(searchText);
    }
    
    // Rebuild queryParams to dispatch new browser location to react-router
    const newQueryParams:{[key: string]: any} = {};
    if (!isEmpty(searchQuery)) {
      newQueryParams.query = searchQuery;
    }
    if (sortBy[0] && sortBy[0].key) {
      newQueryParams.order_key = sortBy[0].key;
    } else {
      newQueryParams.order_key = DEFAULT_SORT_HEADER;
    }
    if (sortBy[0] && sortBy[0].direction) {
      newQueryParams.order_direction = sortBy[0].direction;
    } else {
      newQueryParams.order_direction = DEFAULT_SORT_DIRECTION;
    }
    if (teamId) {
      newQueryParams.team_id = teamId;
    }
    if (policyId) {
      newQueryParams.policy_id = policyId;
    }
    if (policyResponse) {
      newQueryParams.policy_response = policyResponse;
    }

    // triggers useDeepEffect using queryParams
    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.MANAGE_HOSTS,
        routeTemplate,
        routeParams,
        queryParams: newQueryParams,
      })
    );
  };

  const onEditLabel = async (formData: ILabelFormData) => {
    if (!selectedLabel) {
      console.error("Label isn't available. This should not happen.")
      return;
    }
    
    const updateAttrs = deepDifference(formData, selectedLabel);
    try {
      await labelsAPI.update(selectedLabel, updateAttrs);
      refetchLabels();
      dispatch(
        renderFlash(
          "success",
          "Label updated. Try refreshing this page in just a moment to see the updated host count for your label."
        )
      );
    } catch(error) {
      console.error(error);
      dispatch(
        renderFlash("error", "Could not create label. Please try again.")
      );
    }
  };

  const onLabelClick = (selectedLabel: ILabel) => {
    return (evt: React.MouseEvent<HTMLButtonElement>) => {
      evt.preventDefault();
      handleLabelChange(selectedLabel);
    };
  };

  const onOsqueryTableSelect = (tableName: string) => {
    setSelectedOsqueryTable(tableName);
  };

  const onSaveAddLabel = async (formData: ILabelFormData) => {
    try {
      await labelsAPI.create(formData);
      router.push(PATHS.MANAGE_HOSTS);
      refetchLabels();
      
      // TODO flash messages are not visible seemingly because of page renders
      dispatch(
        renderFlash(
          "success",
          "Label created. Try refreshing this page in just a moment to see the updated host count for your label."
        )
      );
    } catch(error) {
      console.error(error);
      dispatch(
        renderFlash("error", "Could not create label. Please try again.")
      );
    }
  };

  const onDeleteLabel = async () => {
    if (!selectedLabel) {
      console.error("Label isn't available. This should not happen.")
      return false;
    }

    const { MANAGE_HOSTS } = PATHS;
    try {
      await labelsAPI.destroy(selectedLabel);
      toggleDeleteLabelModal();
      refetchLabels();

      router.push(
        getNextLocationPath({
          pathPrefix: MANAGE_HOSTS,
          routeTemplate: routeTemplate.replace("/labels/:label_id", ""),
          routeParams,
          queryParams,
        })
      )
    } catch(error) {
      console.error(error);
      dispatch(
        renderFlash("error", "Could not delete label. Please try again.")
      );
    }
  };

  const onTransferToTeamClick = (selectedHostIds: number[]) => {
    toggleTransferHostModal();
    setSelectedHostIds(selectedHostIds);
  };

  const onTransferHostSubmit = async (team: ITeam) => {
    const teamId = typeof team.id === "number" ? team.id : null;
    let action = hostsAPI.transferToTeam(teamId, selectedHostIds);

    if (isAllMatchingHostsSelected) {
      let status = "";
      let labelId = null;
      const selectedStatus = getStatusSelected();      

      if (selectedStatus && isAcceptableStatus(selectedStatus)) {
        status = getStatusSelected() || "";
      } else {
        labelId = selectedLabel?.id as number;
      }

      action = hostsAPI.transferToTeamByFilter(
        teamId,
        searchQuery,
        status,
        labelId
      );
    }

    try {
      await action;

      const successMessage =
        teamId === null
          ? `Hosts successfully removed from teams.`
          : `Hosts successfully transferred to  ${team.name}.`;

      dispatch(renderFlash("success", successMessage));
      retrieveHosts({
        selectedLabels: selectedFilters,
        globalFilter: searchQuery,
        sortBy,
        teamId: currentTeam?.id,
        policyId,
        policyResponse,
      });

      toggleTransferHostModal();
      setSelectedHostIds([]);
      setIsAllMatchingHostsSelected(false);
    } catch(error) {
      dispatch(
        renderFlash("error", "Could not transfer hosts. Please try again.")
      );
    }
  };

  const getLabelSelected = () => {
    return selectedFilters.find((f) => f.includes(LABEL_SLUG_PREFIX));
  };

  const getStatusSelected = () => {
    return selectedFilters.find((f) => !f.includes(LABEL_SLUG_PREFIX));
  };

  const retrieveHosts = async (options: IHostLoadOptions = {}) => {
    setIsHostsLoading(true);

    options = {
      ...options,
      teamId: getValidatedTeamId(
        teams || [],
        options.teamId as number,
        currentUser,
        isOnGlobalTeam as boolean,
      ),
    };

    try {
      const { hosts } = await hostsAPI.loadAll(options);
      setHosts(hosts);
    } catch (error) {
      console.error(error);
      setHasHostErrors(true);
    } finally {
      setIsHostsLoading(false);
    }
  };

  const toggleEnrollSecretModal = () => {
    setShowEnrollSecretModal(!showEnrollSecretModal);
  };

  const toggleAddHostModal = () => {
    setShowAddHostModal(!showAddHostModal);
  };

  const toggleDeleteLabelModal = () => {
    setShowDeleteLabelModal(!showDeleteLabelModal);
  };

  const toggleTransferHostModal = () => {
    setShowTransferHostModal(!showTransferHostModal);
  };

  const toggleAllMatchingHosts = (shouldSelect: boolean) => {
    if (typeof shouldSelect !== "undefined") {
      setIsAllMatchingHostsSelected(shouldSelect);
    } else {
      setIsAllMatchingHostsSelected(!isAllMatchingHostsSelected);
    }
  };

  const handleChangePoliciesFilter = (policyResponse: PolicyResponse) => {
    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.MANAGE_HOSTS,
        routeTemplate,
        routeParams,
        queryParams: Object.assign({}, queryParams, {
          policy_id: policyId,
          policy_response: policyResponse,
        }),
      })
    );
  };

  const handleClearPoliciesFilter = () => {
    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.MANAGE_HOSTS,
        routeTemplate,
        routeParams,
        queryParams: omit(queryParams, ["policy_id", "policy_response"]),
      })
    );
  };

  // The handleChange method below is for the filter-by-team dropdown rather than the dropdown used in modals
  const handleChangeSelectedTeamFilter = (selectedTeam: number) => {
    const { MANAGE_HOSTS } = PATHS;
    const teamIdParam = getValidatedTeamId(
      teams || [],
      selectedTeam,
      currentUser,
      isOnGlobalTeam as boolean,
    );
    const newQueryParams = !teamIdParam
      ? omit(queryParams, "team_id")
      : Object.assign({}, queryParams, { team_id: teamIdParam })

    const nextLocation = getNextLocationPath({
      pathPrefix: MANAGE_HOSTS,
      routeTemplate,
      routeParams,
      queryParams: newQueryParams,
    });
    
    router.replace(nextLocation);
  };

  const handleLabelChange = ({ slug }: ILabel) => {
    if (!slug) {
      console.error("Slug was missing. This should not happen.");
      return false;
    }
    
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

    router.replace(
      getNextLocationPath({
        pathPrefix: isAllHosts
          ? MANAGE_HOSTS
          : `${MANAGE_HOSTS}/${newFilters.join("/")}`,
        queryParams: newQueryParams,
      })
    );
  };

  const handleStatusDropdownChange = (statusName: string) => {
    // we want the full label object
    const isAll = statusName === ALL_HOSTS_LABEL;
    const selected = isAll
      ? find(labels, { type: "all" })
      : find(labels, { id: statusName });
    handleLabelChange(selected as ILabel);
  };

  const renderTeamsFilterDropdown = () => {
    if (!isPremiumTier || !teams) {
      return null;
    }

    if (!isPremiumTier) {
      return <h1>Hosts</h1>;
    }

    const teamOptions = generateTeamFilterDropdownOptions(teams, currentUser, isOnGlobalTeam as boolean);
    const selectedTeamId = getValidatedTeamId(
      teams || [],
      currentTeam?.id as number,
      currentUser,
      isOnGlobalTeam as boolean,
    );

    return (
      <div>
        <Dropdown
          value={selectedTeamId}
          placeholder={"All teams"}
          className={`${baseClass}__team-dropdown`}
          options={teamOptions}
          searchable={false}
          onChange={(newSelectedValue: number) =>
            handleChangeSelectedTeamFilter(newSelectedValue)
          }
        />
      </div>
    );
  };

  const renderPoliciesFilterBlock = () => {
    return (
      <div className={`${baseClass}__policies-filter-block`}>
        <PoliciesFilter
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

  const renderEditColumnsModal = () => {
    if (!showEditColumnsModal || !config || !currentUser) {
      return null;
    }

    return (
      <Modal
        title="Edit Columns"
        onExit={() => setShowEditColumnsModal(false)}
        className={`${baseClass}__invite-modal`}
      >
        <EditColumnsModal
          columns={generateAvailableTableHeaders(config, currentUser)}
          hiddenColumns={hiddenColumns}
          onSaveColumns={onSaveColumns}
          onCancelColumns={onCancelColumns}
        />
      </Modal>
    );
  };

  const renderEnrollSecretModal = () => {
    if (!canEnrollHosts || !showEnrollSecretModal || !teams) {
      return null;
    }

    return (
      <Modal
        title="Enroll secret"
        onExit={toggleEnrollSecretModal}
        className={`${baseClass}__enroll-secret-modal`}
      >
        <EnrollSecretModal
          selectedTeam={currentTeam?.id || 0}
          teams={teams}
          onReturnToApp={toggleEnrollSecretModal}
          isPremiumTier={isPremiumTier as boolean}
        />
      </Modal>
    );
  };

  const renderAddHostModal = () => {
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
          onReturnToApp={toggleAddHostModal}
          config={config}
          currentUser={currentUser}
        />
      </Modal>
    );
  };

  const renderDeleteLabelModal = () => {
    if (!showDeleteLabelModal) {
      return false;
    }

    return (
      <Modal
        title="Delete label"
        onExit={toggleDeleteLabelModal}
        className={`${baseClass}_delete-label__modal`}
      >
        <>
          <p>Are you sure you wish to delete this label?</p>
          <div className={`${baseClass}__modal-buttons`}>
            <Button onClick={toggleDeleteLabelModal} variant="inverse-alert">
              Cancel
            </Button>
            <Button onClick={onDeleteLabel} variant="alert">
              Delete
            </Button>
          </div>
        </>
      </Modal>
    );
  };

  const renderTransferHostModal = () => {
    if (!showTransferHostModal || !teams) {
      return null;
    }

    return (
      <TransferHostModal
        isGlobalAdmin={isGlobalAdmin as boolean}
        teams={teams}
        onSubmit={onTransferHostSubmit}
        onCancel={toggleTransferHostModal}
      />
    );
  };

  const renderHeaderLabelBlock = ({
    description = "",
    display_text: displayText = "",
    label_type: labelType = "",
  }) => {
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

  const renderHeader = () => {
    return (
      <div className={`${baseClass}__header`}>
        <div className={`${baseClass}__text`}>
          {renderTeamsFilterDropdown()}
        </div>
      </div>
    );
  };

  const renderLabelOrPolicyBlock = () => {
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

  const renderForm = () => {
    if (isAddLabel) {
      return (
        <div className="body-wrap">
          <LabelForm
            onCancel={onCancelAddLabel}
            onOsqueryTableSelect={onOsqueryTableSelect}
            handleSubmit={onSaveAddLabel}
            baseError={labelsError?.message || ""}
          />
        </div>
      );
    }

    if (isEditLabel) {
      return (
        <div className="body-wrap">
          <LabelForm
            selectedLabel={selectedLabel}
            onCancel={onCancelEditLabel}
            onOsqueryTableSelect={onOsqueryTableSelect}
            handleSubmit={onEditLabel}
            baseError={labelsError?.message || ""}
            isEdit
          />
        </div>
      );
    }

    return false;
  };

  const renderSidePanel = () => {
    if (!labels) {
      return null;
    }
    
    let SidePanel;

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
          canAddNewLabel={canAddNewLabels as boolean}
        />
      );
    }

    return SidePanel;
  };

  const renderStatusDropdown = () => {
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

  const renderTable = () => {
    if (!config || !currentUser || !hosts ||
      selectedFilters.length === 0 ||
      selectedLabel === undefined
    ) {
      return null;
    }

    if (hasHostErrors) {
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
        defaultSortHeader={(sortBy[0] && sortBy[0].key) || DEFAULT_SORT_HEADER}
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
          {config &&
            (!isPremiumTier || teams) &&
            renderTable()}
        </div>
      )}
      {!isLabelsLoading && renderSidePanel()}
      {renderEnrollSecretModal()}
      {renderAddHostModal()}
      {renderEditColumnsModal()}
      {renderDeleteLabelModal()}
      {renderTransferHostModal()}
    </div>
  );
};

export default ManageHostsPage;