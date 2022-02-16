import React, { useState, useContext, useEffect, useCallback } from "react";
import { useDispatch } from "react-redux";
import { useQuery } from "react-query";
import { InjectedRouter, Params } from "react-router/lib/Router";
import { RouteProps } from "react-router/lib/Route";
import { find, isEmpty, isEqual, omit } from "lodash";
import ReactTooltip from "react-tooltip";

import enrollSecretsAPI from "services/entities/enroll_secret";
import labelsAPI from "services/entities/labels";
import teamsAPI from "services/entities/teams";
import usersAPI, { IGetMeResponse } from "services/entities/users";
import globalPoliciesAPI from "services/entities/global_policies";
import teamPoliciesAPI from "services/entities/team_policies";
import hostsAPI, {
  ILoadHostsOptions,
  ISortOption,
} from "services/entities/hosts";
import hostCountAPI, {
  IHostCountLoadOptions,
} from "services/entities/host_count";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { QueryContext } from "context/query";
import { TableContext } from "context/table";
import {
  IEnrollSecret,
  IEnrollSecretsResponse,
} from "interfaces/enroll_secret";
import { IHost } from "interfaces/host";
import { ILabel, ILabelFormData } from "interfaces/label";
import { IPolicy } from "interfaces/policy";
import { ISoftware } from "interfaces/software";
import { ITeam } from "interfaces/team";
// @ts-ignore
import deepDifference from "utilities/deep_difference";
import sortUtils from "utilities/sort";
import {
  PLATFORM_LABEL_DISPLAY_NAMES,
  PolicyResponse,
} from "utilities/constants";
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";

import Button from "components/buttons/Button";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import HostSidePanel from "components/side_panels/HostSidePanel";
// @ts-ignore
import LabelForm from "components/forms/LabelForm";
import Modal from "components/Modal";
import QuerySidePanel from "components/side_panels/QuerySidePanel";
import TableContainer from "components/TableContainer";
import TableDataError from "components/TableDataError";
import { IActionButtonProps } from "components/TableContainer/DataTable/ActionButton";
import TeamsDropdown from "components/TeamsDropdown";
import Spinner from "components/Spinner";

import { getValidatedTeamId } from "fleet/helpers";
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
} from "./helpers";

import DeleteSecretModal from "../../../components/DeleteSecretModal";
import SecretEditorModal from "../../../components/SecretEditorModal";
import AddHostsModal from "../../../components/AddHostsModal";
import EnrollSecretModal from "../../../components/EnrollSecretModal";
// @ts-ignore
import NoHosts from "./components/NoHosts";
import EmptyHosts from "./components/EmptyHosts";
import PoliciesFilter from "./components/PoliciesFilter";
// @ts-ignore
import EditColumnsModal from "./components/EditColumnsModal/EditColumnsModal";
import TransferHostModal from "./components/TransferHostModal";
import DeleteHostModal from "./components/DeleteHostModal";
import SoftwareVulnerabilities from "./components/SoftwareVulnerabilities";
import EditColumnsIcon from "../../../../assets/images/icon-edit-columns-16x16@2x.png";
import PencilIcon from "../../../../assets/images/icon-pencil-14x14@2x.png";
import TrashIcon from "../../../../assets/images/icon-trash-14x14@2x.png";
import CloseIcon from "../../../../assets/images/icon-close-vibrant-blue-16x16@2x.png";
import CloseIconBlack from "../../../../assets/images/icon-close-fleet-black-16x16@2x.png";
import PolicyIcon from "../../../../assets/images/icon-policy-fleet-black-12x12@2x.png";

interface IManageHostsProps {
  route: RouteProps;
  router: InjectedRouter;
  params: Params;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  location: any; // no type in react-router v3
}

interface ILabelsResponse {
  labels: ILabel[];
}
interface IPolicyAPIResponse {
  policy: IPolicy;
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
}: IManageHostsProps): JSX.Element => {
  const dispatch = useDispatch();
  const queryParams = location.query;
  const {
    availableTeams,
    config,
    currentTeam,
    currentUser,
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamMaintainer,
    isTeamAdmin,
    isOnGlobalTeam,
    isOnlyObserver,
    isPremiumTier,
    isFreeTier,
    setAvailableTeams,
    setCurrentTeam,
    setCurrentUser,
  } = useContext(AppContext);

  useQuery(["me"], () => usersAPI.me(), {
    onSuccess: ({ user, available_teams }: IGetMeResponse) => {
      setCurrentUser(user);
      setAvailableTeams(available_teams);
      if (queryParams.team_id) {
        const teamIdParam = parseInt(queryParams.team_id, 10);
        if (
          isNaN(teamIdParam) ||
          (teamIdParam &&
            available_teams &&
            !available_teams.find((t) => t.id === teamIdParam))
        ) {
          router.replace({
            pathname: location.pathname,
            query: omit(queryParams, "team_id"),
          });
        }
      }
    },
  });

  const { selectedOsqueryTable, setSelectedOsqueryTable } = useContext(
    QueryContext
  );
  const { setResetSelectedRows } = useContext(TableContext);

  const hostHiddenColumns = localStorage.getItem("hostHiddenColumns");
  const storedHiddenColumns = hostHiddenColumns
    ? JSON.parse(hostHiddenColumns)
    : null;

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

  const initialQuery = (() => {
    let query = "";

    if (queryParams && queryParams.query) {
      query = queryParams.query;
    }

    return query;
  })();

  // ========= states
  const [selectedLabel, setSelectedLabel] = useState<ILabel>();
  const [selectedSecret, setSelectedSecret] = useState<IEnrollSecret>();
  const [
    showNoEnrollSecretBanner,
    setShowNoEnrollSecretBanner,
  ] = useState<boolean>(true);
  const [showDeleteSecretModal, setShowDeleteSecretModal] = useState<boolean>(
    false
  );
  const [showSecretEditorModal, setShowSecretEditorModal] = useState<boolean>(
    false
  );
  const [showEnrollSecretModal, setShowEnrollSecretModal] = useState<boolean>(
    false
  );
  const [showDeleteLabelModal, setShowDeleteLabelModal] = useState<boolean>(
    false
  );
  const [showEditColumnsModal, setShowEditColumnsModal] = useState<boolean>(
    false
  );
  const [showAddHostsModal, setShowAddHostsModal] = useState<boolean>(false);
  const [showTransferHostModal, setShowTransferHostModal] = useState<boolean>(
    false
  );
  const [showDeleteHostModal, setShowDeleteHostModal] = useState<boolean>(
    false
  );
  const [hiddenColumns, setHiddenColumns] = useState<string[]>(
    storedHiddenColumns || defaultHiddenColumns
  );
  const [selectedHostIds, setSelectedHostIds] = useState<number[]>([]);
  const [
    isAllMatchingHostsSelected,
    setIsAllMatchingHostsSelected,
  ] = useState<boolean>(false);
  const [searchQuery, setSearchQuery] = useState<string>(initialQuery);
  const [hosts, setHosts] = useState<IHost[]>();
  const [isHostsLoading, setIsHostsLoading] = useState<boolean>(false);
  const [hasHostErrors, setHasHostErrors] = useState<boolean>(false);
  const [filteredHostCount, setFilteredHostCount] = useState<number>();
  const [isHostCountLoading, setIsHostCountLoading] = useState<boolean>(false);
  const [hasHostCountErrors, setHasHostCountErrors] = useState<boolean>(false);
  const [sortBy, setSortBy] = useState<ISortOption[]>(initialSortBy);
  const [policy, setPolicy] = useState<IPolicy>();
  const [softwareDetails, setSoftwareDetails] = useState<ISoftware | null>(
    null
  );
  const [tableQueryData, setTableQueryData] = useState<ITableQueryProps>();
  const [
    currentQueryOptions,
    setCurrentQueryOptions,
  ] = useState<ILoadHostsOptions>();

  // ======== end states

  const isAddLabel = location.hash === NEW_LABEL_HASH;
  const isEditLabel = location.hash === EDIT_LABEL_HASH;
  const routeTemplate = route && route.path ? route.path : "";
  const policyId = queryParams?.policy_id;
  const policyResponse: PolicyResponse = queryParams?.policy_response;
  const softwareId = parseInt(queryParams?.software_id, 10);
  const { active_label: activeLabel, label_id: labelID } = routeParams;

  // ===== filter matching
  const selectedFilters: string[] = [];
  labelID && selectedFilters.push(`${LABEL_SLUG_PREFIX}${labelID}`);
  activeLabel && selectedFilters.push(activeLabel);
  !labelID && !activeLabel && selectedFilters.push(ALL_HOSTS_LABEL); // "all-hosts" should always be alone
  // ===== end filter matching

  const canEnrollHosts =
    isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer;
  const canEnrollGlobalHosts = isGlobalAdmin || isGlobalMaintainer;
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

  const {
    isLoading: isGlobalSecretsLoading,
    data: globalSecrets,
    refetch: refetchGlobalSecrets,
  } = useQuery<IEnrollSecretsResponse, Error, IEnrollSecret[]>(
    ["global secrets"],
    () => enrollSecretsAPI.getGlobalEnrollSecrets(),
    {
      enabled: !!canEnrollGlobalHosts,
      select: (data: IEnrollSecretsResponse) => data.secrets,
    }
  );

  const {
    isLoading: isTeamSecretsLoading,
    data: teamSecrets,
    refetch: refetchTeamSecrets,
  } = useQuery<IEnrollSecretsResponse, Error, IEnrollSecret[]>(
    ["team secrets", currentTeam],
    () => {
      if (currentTeam) {
        return enrollSecretsAPI.getTeamEnrollSecrets(currentTeam.id);
      }
      return { secrets: [] };
    },
    {
      enabled: !!currentTeam?.id && !!canEnrollHosts,
      select: (data: IEnrollSecretsResponse) => data.secrets,
    }
  );

  const addHostsTeam = currentTeam
    ? { name: currentTeam.name, secrets: teamSecrets || null }
    : {
        name: "No team",
        secrets: globalSecrets || null,
      };

  const {
    data: teams,
    isLoading: isLoadingTeams,
    refetch: refetchTeams,
  } = useQuery<ITeamsResponse, Error, ITeam[]>(
    ["teams"],
    () => teamsAPI.loadAll(),
    {
      enabled: !!isPremiumTier,
      select: (data: ITeamsResponse) =>
        data.teams.sort((a, b) => sortUtils.caseInsensitiveAsc(a.name, b.name)),
      onSuccess: (responseTeams: ITeam[]) => {
        if (!currentTeam && !isOnGlobalTeam && responseTeams.length) {
          setCurrentTeam(responseTeams[0]);
        }
      },
    }
  );

  useQuery<IPolicyAPIResponse, Error>(
    ["policy"],
    () => {
      const teamId = parseInt(queryParams?.team_id, 10) || 0;
      const request = teamId
        ? teamPoliciesAPI.load(teamId, policyId)
        : globalPoliciesAPI.load(policyId);
      return request;
    },
    {
      enabled: !!policyId,
      onSuccess: ({ policy: policyAPIResponse }) => {
        setPolicy(policyAPIResponse);
      },
    }
  );

  const toggleDeleteSecretModal = () => {
    // open and closes delete modal
    setShowDeleteSecretModal(!showDeleteSecretModal);
    // open and closes main enroll secret modal
    setShowEnrollSecretModal(!showEnrollSecretModal);
  };

  // this is called when we click add or edit
  const toggleSecretEditorModal = () => {
    // open and closes add/edit modal
    setShowSecretEditorModal(!showSecretEditorModal);
    // open and closes main enroll secret modall
    setShowEnrollSecretModal(!showEnrollSecretModal);
  };

  const toggleDeleteLabelModal = () => {
    setShowDeleteLabelModal(!showDeleteLabelModal);
  };

  const toggleTransferHostModal = () => {
    setShowTransferHostModal(!showTransferHostModal);
  };

  const toggleDeleteHostModal = () => {
    setShowDeleteHostModal(!showDeleteHostModal);
  };

  const toggleAddHostsModal = () => {
    setShowAddHostsModal(!showAddHostsModal);
  };

  const toggleAllMatchingHosts = (shouldSelect: boolean) => {
    if (typeof shouldSelect !== "undefined") {
      setIsAllMatchingHostsSelected(shouldSelect);
    } else {
      setIsAllMatchingHostsSelected(!isAllMatchingHostsSelected);
    }
  };

  const getLabelSelected = () => {
    return selectedFilters.find((f) => f.includes(LABEL_SLUG_PREFIX));
  };

  const getStatusSelected = () => {
    return selectedFilters.find((f) => !f.includes(LABEL_SLUG_PREFIX));
  };

  const retrieveHosts = async (options: ILoadHostsOptions = {}) => {
    setIsHostsLoading(true);

    options = {
      ...options,
      teamId: getValidatedTeamId(
        availableTeams || [],
        options.teamId as number,
        currentUser,
        isOnGlobalTeam as boolean
      ),
    };

    if (queryParams.team_id) {
      options.teamId = queryParams.team_id;
    }

    try {
      const { hosts: returnedHosts, software } = await hostsAPI.loadHosts(
        options
      );
      setHosts(returnedHosts);
      software && setSoftwareDetails(software);
    } catch (error) {
      console.error(error);
      setHasHostErrors(true);
    } finally {
      setIsHostsLoading(false);
    }
  };

  const retrieveHostCount = async (options: IHostCountLoadOptions = {}) => {
    setIsHostCountLoading(true);

    options = {
      ...options,
      teamId: getValidatedTeamId(
        availableTeams || [],
        options.teamId as number,
        currentUser,
        isOnGlobalTeam as boolean
      ),
    };

    if (queryParams.team_id) {
      options.teamId = queryParams.team_id;
    }

    try {
      const { count: returnedHostCount } = await hostCountAPI.load(options);
      setFilteredHostCount(returnedHostCount);
    } catch (error) {
      console.error(error);
      setHasHostCountErrors(true);
    } finally {
      setIsHostCountLoading(false);
    }
  };

  const refetchHosts = (options: ILoadHostsOptions) => {
    retrieveHosts(options);
    if (options.sortBy) {
      delete options.sortBy;
    }
    retrieveHostCount(options);
  };

  let teamSync = false;
  if (currentUser && availableTeams) {
    const teamIdParam = queryParams.team_id
      ? parseInt(queryParams.team_id, 10) // we don't want to parse undefined so we can differntiate non-numeric strings as NaN
      : undefined;
    if (currentTeam?.id && !teamIdParam) {
      teamSync = true;
    } else if (teamIdParam === currentTeam?.id) {
      teamSync = true;
    }
  }

  useEffect(() => {
    const teamId = parseInt(queryParams?.team_id, 10) || 0;
    const selectedTeam = find(availableTeams, ["id", teamId]);
    if (selectedTeam) {
      setCurrentTeam(selectedTeam);
    }
    setShowNoEnrollSecretBanner(true);

    const slugToFind =
      (selectedFilters.length > 0 &&
        selectedFilters.find((f) => f.includes(LABEL_SLUG_PREFIX))) ||
      selectedFilters[0];

    const selected = find(labels, ["slug", slugToFind]) as ILabel;
    setSelectedLabel(selected);

    const options: ILoadHostsOptions = {
      selectedLabels: selectedFilters,
      globalFilter: searchQuery,
      sortBy,
      teamId: selectedTeam?.id,
      policyId,
      policyResponse,
      softwareId,
      page: tableQueryData ? tableQueryData.pageIndex : 0,
      perPage: tableQueryData ? tableQueryData.pageSize : 100,
    };

    if (isEqual(options, currentQueryOptions)) {
      return;
    }
    if (teamSync) {
      retrieveHosts(options);
      retrieveHostCount(options);
      setCurrentQueryOptions(options);
    }
  }, [availableTeams, currentTeam, location, labels]);

  const handleLabelChange = ({ slug }: ILabel): boolean => {
    if (!slug) {
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

    // Non-status labels are not compatible with policies or software filters
    // so omit policies and software params from next location
    let newQueryParams = queryParams;
    if (newFilters.find((f) => f.includes(LABEL_SLUG_PREFIX))) {
      newQueryParams = omit(newQueryParams, [
        "policy_id",
        "policy_response",
        "software_id",
      ]);
    }

    router.replace(
      getNextLocationPath({
        pathPrefix: isAllHosts
          ? MANAGE_HOSTS
          : `${MANAGE_HOSTS}/${newFilters.join("/")}`,
        queryParams: newQueryParams,
      })
    );

    return true;
  };

  const handleChangePoliciesFilter = (response: PolicyResponse) => {
    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.MANAGE_HOSTS,
        routeTemplate,
        routeParams,
        queryParams: Object.assign({}, queryParams, {
          policy_id: policyId,
          policy_response: response,
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

  const handleClearSoftwareFilter = () => {
    router.replace(PATHS.MANAGE_HOSTS);
    setCurrentTeam(undefined);
    setSoftwareDetails(null);
  };

  const handleTeamSelect = (teamId: number) => {
    const { MANAGE_HOSTS } = PATHS;
    const teamIdParam = getValidatedTeamId(
      availableTeams || [],
      teamId,
      currentUser,
      isOnGlobalTeam as boolean
    );

    const slimmerParams = omit(queryParams, [
      "policy_id",
      "policy_response",
      "team_id",
    ]);

    const newQueryParams = !teamIdParam
      ? slimmerParams
      : Object.assign({}, slimmerParams, { team_id: teamIdParam });

    const nextLocation = getNextLocationPath({
      pathPrefix: MANAGE_HOSTS,
      routeTemplate,
      routeParams,
      queryParams: newQueryParams,
    });
    router.replace(nextLocation);
    const selectedTeam = find(availableTeams, ["id", teamId]);
    setCurrentTeam(selectedTeam);
  };

  const handleStatusDropdownChange = (statusName: string) => {
    // we want the full label object
    const isAll = statusName === ALL_HOSTS_LABEL;
    const selected = isAll
      ? find(labels, { type: "all" })
      : find(labels, { id: statusName });
    handleLabelChange(selected as ILabel);
  };

  const onAddLabelClick = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();
    router.push(`${PATHS.MANAGE_HOSTS}${NEW_LABEL_HASH}`);
  };

  const onEditLabelClick = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();
    router.push(
      `${PATHS.MANAGE_HOSTS}/${getLabelSelected()}${EDIT_LABEL_HASH}`
    );
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

  // NOTE: this is called once on initial render and every time the query changes
  const onTableQueryChange = useCallback(
    async (newTableQuery: ITableQueryProps) => {
      if (isEqual(newTableQuery, tableQueryData)) {
        return;
      }

      setTableQueryData({ ...newTableQuery });

      const {
        searchQuery: searchText,
        sortHeader,
        sortDirection,
      } = newTableQuery;

      let sort = sortBy;
      if (sortHeader) {
        sort = [
          {
            key: sortHeader,
            direction: sortDirection || DEFAULT_SORT_DIRECTION,
          },
        ];
      } else if (!sortBy.length) {
        sort = [
          { key: DEFAULT_SORT_HEADER, direction: DEFAULT_SORT_DIRECTION },
        ];
      }

      if (!isEqual(sort, sortBy)) {
        setSortBy([...sort]);
      }

      if (!isEqual(searchText, searchQuery)) {
        setSearchQuery(searchText);
      }

      // Rebuild queryParams to dispatch new browser location to react-router
      const newQueryParams: { [key: string]: string | number } = {};
      if (!isEmpty(searchText)) {
        newQueryParams.query = searchText;
      }

      newQueryParams.order_key = sort[0].key || DEFAULT_SORT_HEADER;
      newQueryParams.order_direction =
        sort[0].direction || DEFAULT_SORT_DIRECTION;

      if (currentTeam?.id) {
        newQueryParams.team_id = currentTeam.id;
      }

      if (policyId) {
        newQueryParams.policy_id = policyId;
      }

      if (policyResponse) {
        newQueryParams.policy_response = policyResponse;
      }

      if (softwareId && !policyId) {
        newQueryParams.software_id = softwareId;
      }

      router.replace(
        getNextLocationPath({
          pathPrefix: PATHS.MANAGE_HOSTS,
          routeTemplate,
          routeParams,
          queryParams: newQueryParams,
        })
      );
    },
    [
      availableTeams,
      currentTeam,
      currentUser,
      policyId,
      queryParams,
      softwareId,
      sortBy,
    ]
  );

  const onSaveSecret = async (enrollSecretString: string) => {
    const { MANAGE_HOSTS } = PATHS;

    // Creates new list of secrets removing selected secret and adding new secret
    const currentSecrets = currentTeam
      ? teamSecrets || []
      : globalSecrets || [];

    const newSecrets = currentSecrets.filter(
      (s) => s.secret !== selectedSecret?.secret
    );

    if (enrollSecretString) {
      newSecrets.push({ secret: enrollSecretString });
    }

    try {
      if (currentTeam?.id) {
        await enrollSecretsAPI.modifyTeamEnrollSecrets(
          currentTeam.id,
          newSecrets
        );
        refetchTeamSecrets();
      } else {
        await enrollSecretsAPI.modifyGlobalEnrollSecrets(newSecrets);
        refetchGlobalSecrets();
      }
      toggleSecretEditorModal();
      isPremiumTier && refetchTeams();

      router.push(
        getNextLocationPath({
          pathPrefix: MANAGE_HOSTS,
          routeTemplate: routeTemplate.replace("/labels/:label_id", ""),
          routeParams,
          queryParams,
        })
      );
      dispatch(
        renderFlash(
          "success",
          `Successfully ${selectedSecret ? "edited" : "added"} enroll secret.`
        )
      );
    } catch (error) {
      console.error(error);
      dispatch(
        renderFlash(
          "error",
          `Could not ${
            selectedSecret ? "edit" : "add"
          } enroll secret. Please try again.`
        )
      );
    }
  };

  const onDeleteSecret = async () => {
    const { MANAGE_HOSTS } = PATHS;

    // create new list of secrets removing selected secret
    const currentSecrets = currentTeam
      ? teamSecrets || []
      : globalSecrets || [];

    const newSecrets = currentSecrets.filter(
      (s) => s.secret !== selectedSecret?.secret
    );

    try {
      if (currentTeam?.id) {
        await enrollSecretsAPI.modifyTeamEnrollSecrets(
          currentTeam.id,
          newSecrets
        );
        refetchTeamSecrets();
      } else {
        await enrollSecretsAPI.modifyGlobalEnrollSecrets(newSecrets);
        refetchGlobalSecrets();
      }
      toggleDeleteSecretModal();
      refetchTeams();
      router.push(
        getNextLocationPath({
          pathPrefix: MANAGE_HOSTS,
          routeTemplate: routeTemplate.replace("/labels/:label_id", ""),
          routeParams,
          queryParams,
        })
      );
      dispatch(renderFlash("success", `Successfully deleted enroll secret.`));
    } catch (error) {
      console.error(error);
      dispatch(
        renderFlash(
          "error",
          "Could not delete enroll secret. Please try again."
        )
      );
    }
  };

  const onEditLabel = async (formData: ILabelFormData) => {
    if (!selectedLabel) {
      console.error("Label isn't available. This should not happen.");
      return;
    }

    const updateAttrs = deepDifference(formData, selectedLabel);
    try {
      await labelsAPI.update(selectedLabel, updateAttrs);
      refetchLabels();
      router.push(`${PATHS.MANAGE_HOSTS}/${getLabelSelected()}`);
      dispatch(
        renderFlash(
          "success",
          "Label updated. Try refreshing this page in just a moment to see the updated host count for your label."
        )
      );
    } catch (error) {
      console.error(error);
      dispatch(
        renderFlash("error", "Could not create label. Please try again.")
      );
    }
  };

  const onLabelClick = (label: ILabel) => {
    return (evt: React.MouseEvent<HTMLButtonElement>) => {
      evt.preventDefault();
      handleLabelChange(label);
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
    } catch (error) {
      console.error(error);
      dispatch(
        renderFlash("error", "Could not create label. Please try again.")
      );
    }
  };

  const onDeleteLabel = async () => {
    if (!selectedLabel) {
      console.error("Label isn't available. This should not happen.");
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
      );
    } catch (error) {
      console.error(error);
      dispatch(
        renderFlash("error", "Could not delete label. Please try again.")
      );
    }
  };

  const onTransferToTeamClick = (hostIds: number[]) => {
    toggleTransferHostModal();
    setSelectedHostIds(hostIds);
  };

  const onDeleteHostsClick = (hostIds: number[]) => {
    toggleDeleteHostModal();
    setSelectedHostIds(hostIds);
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
      setResetSelectedRows(true);
      refetchHosts({
        selectedLabels: selectedFilters,
        globalFilter: searchQuery,
        sortBy,
        teamId: currentTeam?.id,
        policyId,
        policyResponse,
        softwareId,
      });

      toggleTransferHostModal();
      setSelectedHostIds([]);
      setIsAllMatchingHostsSelected(false);
    } catch (error) {
      dispatch(
        renderFlash("error", "Could not transfer hosts. Please try again.")
      );
    }
  };

  const onDeleteHostSubmit = async () => {
    let action = hostsAPI.destroyBulk(selectedHostIds);

    if (isAllMatchingHostsSelected) {
      let status = "";
      let labelId = null;
      const teamId = currentTeam?.id || null;
      const selectedStatus = getStatusSelected();

      if (selectedStatus && isAcceptableStatus(selectedStatus)) {
        status = getStatusSelected() || "";
      } else {
        labelId = selectedLabel?.id as number;
      }

      action = hostsAPI.destroyByFilter(teamId, searchQuery, status, labelId);
    }

    try {
      await action;

      const successMessage = `${
        selectedHostIds.length === 1 ? "Host" : "Hosts"
      } successfully deleted.`;

      dispatch(renderFlash("success", successMessage));
      setResetSelectedRows(true);
      refetchHosts({
        selectedLabels: selectedFilters,
        globalFilter: searchQuery,
        sortBy,
        teamId: currentTeam?.id,
        policyId,
        policyResponse,
        softwareId,
      });

      refetchLabels();
      toggleDeleteHostModal();
      setSelectedHostIds([]);
      setIsAllMatchingHostsSelected(false);
    } catch (error) {
      dispatch(
        renderFlash(
          "error",
          `Could not delete ${
            selectedHostIds.length === 1 ? "host" : "hosts"
          }. Please try again.`
        )
      );
    }
  };

  const renderTeamsFilterDropdown = () => (
    <TeamsDropdown
      currentUserTeams={availableTeams || []}
      selectedTeamId={
        (policyId && policy?.team_id) || (currentTeam?.id as number)
      }
      isDisabled={isHostsLoading || isHostCountLoading}
      onChange={(newSelectedValue: number) =>
        handleTeamSelect(newSelectedValue)
      }
    />
  );

  const renderPoliciesFilterBlock = () => {
    return (
      <div className={`${baseClass}__policies-filter-block`}>
        <PoliciesFilter
          policyResponse={policyResponse}
          onChange={handleChangePoliciesFilter}
        />
        <div className={`${baseClass}__policies-filter-name-card`}>
          <img src={PolicyIcon} alt="Policy" />
          {policy?.name}
          <Button
            className={`${baseClass}__clear-policies-filter`}
            onClick={handleClearPoliciesFilter}
            variant={"small-text-icon"}
            title={policy?.name}
          >
            <img src={CloseIcon} alt="Remove policy filter" />
          </Button>
        </div>
      </div>
    );
  };

  const renderSoftwareFilterBlock = () => {
    if (softwareDetails) {
      const { name, version } = softwareDetails;
      const buttonText = name && version ? `${name} ${version}` : "";
      return (
        <div className={`${baseClass}__software-filter-block`}>
          <div>
            <span
              className="software-filter-tooltip"
              data-tip
              data-for="software-filter-tooltip"
              data-tip-disable={!name || !version}
            >
              <div className={`${baseClass}__software-filter-name-card`}>
                {buttonText}
                <Button
                  className={`${baseClass}__clear-policies-filter`}
                  onClick={handleClearSoftwareFilter}
                  variant={"small-text-icon"}
                  title={buttonText}
                >
                  <img src={CloseIcon} alt="Remove policy filter" />
                </Button>
              </div>
            </span>
            <ReactTooltip
              place="bottom"
              type="dark"
              effect="solid"
              backgroundColor="#3e4771"
              id="software-filter-tooltip"
              data-html
            >
              <span className={`tooltip__tooltip-text`}>
                {`Hosts with ${name}`},<br />
                {`${version} installed`}
              </span>
            </ReactTooltip>
          </div>
        </div>
      );
    }
    return null;
  };

  const renderEditColumnsModal = () => {
    if (!showEditColumnsModal || !config || !currentUser) {
      return null;
    }

    return (
      <Modal
        title="Edit columns"
        onExit={() => setShowEditColumnsModal(false)}
        className={`${baseClass}__invite-modal`}
      >
        <EditColumnsModal
          columns={generateAvailableTableHeaders(
            config,
            currentUser,
            currentTeam
          )}
          hiddenColumns={hiddenColumns}
          onSaveColumns={onSaveColumns}
          onCancelColumns={onCancelColumns}
        />
      </Modal>
    );
  };

  const renderSecretEditorModal = () => {
    if (!canEnrollHosts || !showSecretEditorModal) {
      return null;
    }

    return (
      <SecretEditorModal
        selectedTeam={currentTeam?.id || 0}
        teams={teams || []}
        onSaveSecret={onSaveSecret}
        toggleSecretEditorModal={toggleSecretEditorModal}
        selectedSecret={selectedSecret}
      />
    );
  };

  const renderDeleteSecretModal = () => {
    if (!canEnrollHosts || !showDeleteSecretModal) {
      return null;
    }

    return (
      <DeleteSecretModal
        onDeleteSecret={onDeleteSecret}
        selectedTeam={currentTeam?.id || 0}
        teams={teams || []}
        toggleDeleteSecretModal={toggleDeleteSecretModal}
      />
    );
  };

  const renderEnrollSecretModal = () => {
    if (!canEnrollHosts || !showEnrollSecretModal) {
      return null;
    }

    return (
      <EnrollSecretModal
        selectedTeam={currentTeam?.id || 0}
        teams={teams || []}
        onReturnToApp={() => setShowEnrollSecretModal(false)}
        toggleSecretEditorModal={toggleSecretEditorModal}
        toggleDeleteSecretModal={toggleDeleteSecretModal}
        setSelectedSecret={setSelectedSecret}
        globalSecrets={globalSecrets}
      />
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

  const renderAddHostsModal = () => {
    if (!showAddHostsModal) {
      return null;
    }

    return (
      <AddHostsModal
        onCancel={toggleAddHostsModal}
        selectedTeam={addHostsTeam}
      />
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

  const renderDeleteHostModal = () => {
    if (!showDeleteHostModal) {
      return null;
    }

    return (
      <DeleteHostModal
        selectedHostIds={selectedHostIds}
        onSubmit={onDeleteHostSubmit}
        onCancel={toggleDeleteHostModal}
        isAllMatchingHostsSelected={isAllMatchingHostsSelected}
      />
    );
  };

  const renderHeaderLabelBlock = () => {
    if (selectedLabel) {
      const {
        description,
        display_text: displayText,
        label_type: labelType,
      } = selectedLabel;

      return (
        <div className={`${baseClass}__label-block`}>
          <div className="title">
            <span>
              {PLATFORM_LABEL_DISPLAY_NAMES[displayText] || displayText}
            </span>
            {labelType !== "builtin" && !isOnlyObserver && (
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
    }

    return null;
  };

  const renderHeader = () => {
    return (
      <div className={`${baseClass}__header`}>
        <div className={`${baseClass}__text`}>
          <div className={`${baseClass}__title`}>
            {isFreeTier && <h1>Hosts</h1>}
            {isPremiumTier &&
              availableTeams &&
              (availableTeams.length > 1 || isOnGlobalTeam) &&
              renderTeamsFilterDropdown()}
            {isPremiumTier &&
              !isOnGlobalTeam &&
              availableTeams &&
              availableTeams.length === 1 && <h1>{availableTeams[0].name}</h1>}
          </div>
        </div>
      </div>
    );
  };

  const renderActiveFilterBlock = () => {
    const showSelectedLabel =
      selectedLabel &&
      selectedLabel.type !== "all" &&
      selectedLabel.type !== "status";
    if (policyId || softwareId || showSelectedLabel) {
      return (
        <div className={`${baseClass}__labels-active-filter-wrap`}>
          {showSelectedLabel && renderHeaderLabelBlock()}
          {!!policyId &&
            !softwareId &&
            !showSelectedLabel &&
            renderPoliciesFilterBlock()}
          {!!softwareId &&
            !policyId &&
            !showSelectedLabel &&
            renderSoftwareFilterBlock()}
        </div>
      );
    }
    return null;
  };

  const renderSoftwareVulnerabilities = () => {
    if (softwareId && softwareDetails) {
      return <SoftwareVulnerabilities software={softwareDetails} />;
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
          isLabelsLoading={isLabelsLoading}
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
    if (
      !config ||
      !currentUser ||
      !hosts ||
      selectedFilters.length === 0 ||
      selectedLabel === undefined ||
      !teamSync
    ) {
      return <Spinner />;
    }

    if (hasHostErrors || hasHostCountErrors) {
      return <TableDataError />;
    }

    // There are no hosts for this instance yet
    if (
      getStatusSelected() === ALL_HOSTS_LABEL &&
      !isHostCountLoading &&
      filteredHostCount === 0 &&
      searchQuery === "" &&
      !isHostsLoading &&
      teamSync
    ) {
      const { software_id, policy_id } = queryParams || {};
      const includesSoftwareOrPolicyFilter = !!(software_id || policy_id);

      return (
        <NoHosts
          toggleAddHostsModal={toggleAddHostsModal}
          canEnrollHosts={canEnrollHosts}
          includesSoftwareOrPolicyFilter={includesSoftwareOrPolicyFilter}
        />
      );
    }

    const secondarySelectActions: IActionButtonProps[] = [
      {
        name: "transfer",
        onActionButtonClick: onTransferToTeamClick,
        buttonText: "Transfer",
        variant: "text-icon",
        icon: "transfer",
        hideButton: !isPremiumTier || (!isGlobalAdmin && !isGlobalMaintainer),
      },
    ];

    return (
      <TableContainer
        columns={generateVisibleTableColumns(
          hiddenColumns,
          config,
          currentUser,
          currentTeam
        )}
        data={hosts}
        isLoading={isHostsLoading || isHostCountLoading}
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
        primarySelectActionButtonText={"Delete"}
        primarySelectActionButtonIcon={"delete"}
        primarySelectActionButtonVariant={"text-icon"}
        secondarySelectActions={secondarySelectActions}
        resultsTitle={"hosts"}
        showMarkAllPages
        isAllPagesSelected={isAllMatchingHostsSelected}
        searchable
        filteredCount={filteredHostCount}
        searchToolTipText={
          "Search hosts by hostname, UUID, machine serial or IP address"
        }
        emptyComponent={EmptyHosts}
        customControl={renderStatusDropdown}
        onActionButtonClick={onEditColumnsClick}
        onPrimarySelectActionClick={onDeleteHostsClick}
        onQueryChange={onTableQueryChange}
        toggleAllPagesSelected={toggleAllMatchingHosts}
      />
    );
  };

  const renderNoEnrollSecretBanner = () => {
    const noTeamEnrollSecrets =
      currentTeam &&
      !!currentTeam?.id &&
      !isTeamSecretsLoading &&
      !teamSecrets?.length;
    const noGlobalEnrollSecrets =
      (!isPremiumTier ||
        (isPremiumTier && !currentTeam?.id && !isLoadingTeams)) &&
      !isGlobalSecretsLoading &&
      !globalSecrets?.length;

    return (
      ((canEnrollHosts && noTeamEnrollSecrets) ||
        (canEnrollGlobalHosts && noGlobalEnrollSecrets)) &&
      showNoEnrollSecretBanner && (
        <div className={`${baseClass}__no-enroll-secret-banner`}>
          <div>
            <span>
              You have no enroll secrets. Manage enroll secrets to enroll hosts
              to <b>{currentTeam?.id ? currentTeam.name : "Fleet"}</b>.
            </span>
          </div>
          <div className={`dismiss-banner-button`}>
            <button
              className="button button--unstyled"
              onClick={() =>
                setShowNoEnrollSecretBanner(!showNoEnrollSecretBanner)
              }
            >
              <img alt="Dismiss no enroll secret banner" src={CloseIconBlack} />
            </button>
          </div>
        </div>
      )
    );
  };

  if (!teamSync) {
    return <Spinner />;
  }

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
                  onClick={() => setShowEnrollSecretModal(true)}
                  className={`${baseClass}__enroll-hosts button`}
                  variant="inverse"
                >
                  <span>Manage enroll secret</span>
                </Button>
              )}
              {canEnrollHosts &&
                !(
                  getStatusSelected() === ALL_HOSTS_LABEL &&
                  selectedLabel?.count === 0
                ) &&
                !(
                  getStatusSelected() === ALL_HOSTS_LABEL &&
                  filteredHostCount === 0
                ) && (
                  <Button
                    onClick={toggleAddHostsModal}
                    className={`${baseClass}__add-hosts button button--brand`}
                  >
                    <span>Add hosts</span>
                  </Button>
                )}
            </div>
          </div>
          {renderActiveFilterBlock()}
          {renderNoEnrollSecretBanner() ||
            (renderSoftwareVulnerabilities() && (
              <div className={`${baseClass}__info-banners`}>
                {renderNoEnrollSecretBanner()}
                {renderSoftwareVulnerabilities()}
              </div>
            ))}
          {renderTable()}
        </div>
      )}
      {renderSidePanel()}
      {renderDeleteSecretModal()}
      {renderSecretEditorModal()}
      {renderEnrollSecretModal()}
      {renderEditColumnsModal()}
      {renderDeleteLabelModal()}
      {renderAddHostsModal()}
      {renderTransferHostModal()}
      {renderDeleteHostModal()}
    </div>
  );
};

export default ManageHostsPage;
