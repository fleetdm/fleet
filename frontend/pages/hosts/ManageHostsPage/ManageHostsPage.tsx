import React, { useState, useContext, useEffect, useCallback } from "react";
import { useQuery } from "react-query";
import { InjectedRouter, Params } from "react-router/lib/Router";
import { RouteProps } from "react-router/lib/Route";
import { find, isEmpty, isEqual, omit } from "lodash";
import { format } from "date-fns";
import FileSaver from "file-saver";

import enrollSecretsAPI from "services/entities/enroll_secret";
import labelsAPI, { ILabelsResponse } from "services/entities/labels";
import teamsAPI, { ILoadTeamsResponse } from "services/entities/teams";
import globalPoliciesAPI from "services/entities/global_policies";
import teamPoliciesAPI from "services/entities/team_policies";
import hostsAPI, {
  ILoadHostsOptions,
  ISortOption,
} from "services/entities/hosts";
import hostCountAPI, {
  IHostCountLoadOptions,
} from "services/entities/host_count";
import {
  getOSVersions,
  IGetOSVersionsQueryKey,
  IOSVersionsResponse,
} from "services/entities/operating_systems";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { QueryContext } from "context/query";
import { TableContext } from "context/table";
import { NotificationContext } from "context/notification";
import {
  IEnrollSecret,
  IEnrollSecretsResponse,
} from "interfaces/enroll_secret";
import { IApiError } from "interfaces/errors";
import { IHost } from "interfaces/host";
import { ILabel, ILabelFormData } from "interfaces/label";
import { IMDMSolution } from "interfaces/macadmins";
import { IOperatingSystemVersion } from "interfaces/operating_system";
import { IPolicy } from "interfaces/policy";
import { ISoftware } from "interfaces/software";
import { ITeam } from "interfaces/team";
import deepDifference from "utilities/deep_difference";
import sortUtils from "utilities/sort";
import {
  DEFAULT_CREATE_LABEL_ERRORS,
  HOSTS_SEARCH_BOX_PLACEHOLDER,
  HOSTS_SEARCH_BOX_TOOLTIP,
  PLATFORM_LABEL_DISPLAY_NAMES,
  PolicyResponse,
} from "utilities/constants";

import Button from "components/buttons/Button";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import QuerySidePanel from "components/side_panels/QuerySidePanel";
import TableContainer from "components/TableContainer";
import TableDataError from "components/DataError";
import { IActionButtonProps } from "components/TableContainer/DataTable/ActionButton";
import TeamsDropdown from "components/TeamsDropdown";
import Spinner from "components/Spinner";
import MainContent from "components/MainContent";
import SidePanelContent from "components/SidePanelContent";

import { getValidatedTeamId } from "utilities/helpers";
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
  DEFAULT_PAGE_SIZE,
  HOST_SELECT_STATUSES,
} from "./constants";
import { isAcceptableStatus, getNextLocationPath } from "./helpers";

import LabelForm from "./components/LabelForm";
import DeleteSecretModal from "../../../components/DeleteSecretModal";
import SecretEditorModal from "../../../components/SecretEditorModal";
import AddHostsModal from "../../../components/AddHostsModal";
import EnrollSecretModal from "../../../components/EnrollSecretModal";
import NoHosts from "./components/NoHosts";
import EmptyHosts from "./components/EmptyHosts";
import PoliciesFilter from "./components/PoliciesFilter";
// @ts-ignore
import EditColumnsModal from "./components/EditColumnsModal/EditColumnsModal";
import TransferHostModal from "./components/TransferHostModal";
import DeleteHostModal from "./components/DeleteHostModal";
import DeleteLabelModal from "./components/DeleteLabelModal";
import EditColumnsIcon from "../../../../assets/images/icon-edit-columns-16x16@2x.png";
import PencilIcon from "../../../../assets/images/icon-pencil-14x14@2x.png";
import TrashIcon from "../../../../assets/images/icon-trash-14x14@2x.png";
import CloseIconBlack from "../../../../assets/images/icon-close-fleet-black-16x16@2x.png";
import PolicyIcon from "../../../../assets/images/icon-policy-fleet-black-12x12@2x.png";
import DownloadIcon from "../../../../assets/images/icon-download-12x12@2x.png";
import LabelFilterSelect from "./components/LabelFilterSelect";
import FilterPill from "./components/FilterPill";

interface IManageHostsProps {
  route: RouteProps;
  router: InjectedRouter;
  params: Params;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  location: any; // no type in react-router v3
}

interface IPolicyAPIResponse {
  policy: IPolicy;
}

interface ITableQueryProps {
  pageIndex: number;
  pageSize: number;
  searchQuery: string;
  sortHeader: string;
  sortDirection: string;
}

const CSV_HOSTS_TITLE = "Hosts";
const baseClass = "manage-hosts";

const ManageHostsPage = ({
  route,
  router,
  params: routeParams,
  location,
}: IManageHostsProps): JSX.Element => {
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
    isSandboxMode,
    setCurrentTeam,
  } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  if (queryParams.team_id) {
    const teamIdParam = parseInt(queryParams.team_id, 10);
    if (
      isNaN(teamIdParam) ||
      (teamIdParam &&
        availableTeams &&
        !availableTeams.find((team) => team.id === teamIdParam))
    ) {
      router.replace({
        pathname: location.pathname,
        query: omit(queryParams, "team_id"),
      });
    }
  }

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
  const [
    mdmSolutionDetails,
    setMDMSolutionDetails,
  ] = useState<IMDMSolution | null>(null);
  const [tableQueryData, setTableQueryData] = useState<ITableQueryProps>();
  const [
    currentQueryOptions,
    setCurrentQueryOptions,
  ] = useState<ILoadHostsOptions>();
  const [labelValidator, setLabelValidator] = useState<{
    [key: string]: string;
  }>(DEFAULT_CREATE_LABEL_ERRORS);

  // ======== end states

  const isAddLabel = location.hash === NEW_LABEL_HASH;
  const isEditLabel = location.hash === EDIT_LABEL_HASH;
  const routeTemplate = route?.path ?? "";
  const policyId = queryParams?.policy_id;
  const policyResponse: PolicyResponse = queryParams?.policy_response;
  const softwareId =
    queryParams?.software_id !== undefined
      ? parseInt(queryParams?.software_id, 10)
      : undefined;
  const mdmId =
    queryParams?.mdm_id !== undefined
      ? parseInt(queryParams?.mdm_id, 10)
      : undefined;
  const mdmEnrollmentStatus = queryParams?.mdm_enrollment_status;
  const { os_id, os_name, os_version } = queryParams;
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
  const canAddNewLabels = (isGlobalAdmin || isGlobalMaintainer) ?? false;

  const { data: labels, error: labelsError, refetch: refetchLabels } = useQuery<
    ILabelsResponse,
    Error,
    ILabel[]
  >(["labels"], () => labelsAPI.loadAll(), {
    select: (data: ILabelsResponse) => data.labels,
  });

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

  const {
    data: teams,
    isLoading: isLoadingTeams,
    refetch: refetchTeams,
  } = useQuery<ILoadTeamsResponse, Error, ITeam[]>(
    ["teams"],
    () => teamsAPI.loadAll(),
    {
      enabled: !!isPremiumTier,
      select: (data: ILoadTeamsResponse) =>
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

  const { data: osVersions } = useQuery<
    IOSVersionsResponse,
    Error,
    IOperatingSystemVersion[],
    IGetOSVersionsQueryKey[]
  >([{ scope: "os_versions" }], () => getOSVersions(), {
    enabled:
      !!queryParams?.os_id ||
      (!!queryParams?.os_name && !!queryParams?.os_version),
    keepPreviousData: true,
    select: (data) => data.os_versions,
  });

  const toggleDeleteSecretModal = () => {
    // open and closes delete modal
    setShowDeleteSecretModal(!showDeleteSecretModal);
    // open and closes main enroll secret modal
    setShowEnrollSecretModal(!showEnrollSecretModal);
  };

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

  const toggleEditColumnsModal = () => {
    setShowEditColumnsModal(!showEditColumnsModal);
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
      const {
        hosts: returnedHosts,
        software,
        mobile_device_management_solution,
      } = await hostsAPI.loadHosts(options);
      setHosts(returnedHosts);
      software && setSoftwareDetails(software);
      mobile_device_management_solution &&
        setMDMSolutionDetails(mobile_device_management_solution);
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
    retrieveHostCount(omit(options, "device_mapping"));
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
      mdmId,
      mdmEnrollmentStatus,
      os_id,
      os_name,
      os_version,
      page: tableQueryData ? tableQueryData.pageIndex : 0,
      perPage: tableQueryData ? tableQueryData.pageSize : 100,
      device_mapping: true,
    };

    if (isEqual(options, currentQueryOptions)) {
      return;
    }
    if (teamSync) {
      retrieveHosts(options);
      retrieveHostCount(omit(options, "device_mapping"));
      setCurrentQueryOptions(options);
    }
  }, [availableTeams, currentTeam, location, labels]);

  const isLastPage =
    tableQueryData &&
    !!filteredHostCount &&
    DEFAULT_PAGE_SIZE * tableQueryData.pageIndex + (hosts?.length || 0) >=
      filteredHostCount;

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

  const handleClearOSFilter = () => {
    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.MANAGE_HOSTS,
        routeTemplate,
        routeParams,
        queryParams: omit(queryParams, ["os_id", "os_name", "os_version"]),
      })
    );
  };

  const handleClearSoftwareFilter = () => {
    router.replace(PATHS.MANAGE_HOSTS);
    setSoftwareDetails(null);
  };

  const handleClearMDMSolutionFilter = () => {
    router.replace(PATHS.MANAGE_HOSTS);
    setMDMSolutionDetails(null);
  };

  const handleClearMDMEnrollmentFilter = () => {
    router.replace(PATHS.MANAGE_HOSTS);
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

  const onAddLabelClick = () => {
    setLabelValidator(DEFAULT_CREATE_LABEL_ERRORS);
    router.push(`${PATHS.MANAGE_HOSTS}${NEW_LABEL_HASH}`);
  };

  const onEditLabelClick = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();

    setLabelValidator(DEFAULT_CREATE_LABEL_ERRORS);
    router.push(
      `${PATHS.MANAGE_HOSTS}/${getLabelSelected()}${EDIT_LABEL_HASH}`
    );
  };

  const onSaveColumns = (newHiddenColumns: string[]) => {
    localStorage.setItem("hostHiddenColumns", JSON.stringify(newHiddenColumns));
    setHiddenColumns(newHiddenColumns);
    setShowEditColumnsModal(false);
  };

  const onCancelLabel = () => {
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

      if (softwareId && !policyId && !mdmId && !mdmEnrollmentStatus) {
        newQueryParams.software_id = softwareId;
      }

      if (mdmId && !policyId && !softwareId && !mdmEnrollmentStatus) {
        newQueryParams.mdm_id = mdmId;
      }

      if (mdmEnrollmentStatus && !policyId && !softwareId && !mdmId) {
        newQueryParams.mdm_enrollment_status = mdmEnrollmentStatus;
      }

      if (
        (os_id || (os_name && os_version)) &&
        !softwareId &&
        !policyId &&
        !mdmEnrollmentStatus &&
        !mdmId
      ) {
        newQueryParams.os_id = os_id;
        newQueryParams.os_name = os_name;
        newQueryParams.os_version = os_version;
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
      mdmId,
      mdmEnrollmentStatus,
      os_id,
      os_name,
      os_version,
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
      renderFlash(
        "success",
        `Successfully ${selectedSecret ? "edited" : "added"} enroll secret.`
      );
    } catch (error) {
      console.error(error);
      renderFlash(
        "error",
        `Could not ${
          selectedSecret ? "edit" : "add"
        } enroll secret. Please try again.`
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
      renderFlash("success", `Successfully deleted enroll secret.`);
    } catch (error) {
      console.error(error);
      renderFlash("error", "Could not delete enroll secret. Please try again.");
    }
  };

  const onEditLabel = (formData: ILabelFormData) => {
    if (!selectedLabel) {
      console.error("Label isn't available. This should not happen.");
      return;
    }

    const updateAttrs = deepDifference(formData, selectedLabel);

    labelsAPI
      .update(selectedLabel, updateAttrs)
      .then(() => {
        refetchLabels();
        renderFlash(
          "success",
          "Label updated. Try refreshing this page in just a moment to see the updated host count for your label."
        );
        setLabelValidator({});
      })
      .catch((updateError: { data: IApiError }) => {
        if (updateError.data.errors[0].reason.includes("Duplicate")) {
          setLabelValidator({
            name: "A label with this name already exists",
          });
        } else {
          renderFlash("error", "Could not create label. Please try again.");
        }
      });
  };

  const onOsqueryTableSelect = (tableName: string) => {
    setSelectedOsqueryTable(tableName);
  };

  const onSaveAddLabel = (formData: ILabelFormData) => {
    labelsAPI
      .create(formData)
      .then(() => {
        router.push(PATHS.MANAGE_HOSTS);
        renderFlash(
          "success",
          "Label created. Try refreshing this page in just a moment to see the updated host count for your label."
        );
        setLabelValidator({});
        refetchLabels();
      })
      .catch((updateError: any) => {
        if (updateError.data.errors[0].reason.includes("Duplicate")) {
          setLabelValidator({
            name: "A label with this name already exists",
          });
        } else {
          renderFlash("error", "Could not create label. Please try again.");
        }
      });
  };

  const onClearLabelFilter = () => {
    const allHostsLabel = labels?.find((label) => label.name === "All Hosts");
    if (allHostsLabel !== undefined) {
      handleLabelChange(allHostsLabel);
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
      renderFlash("error", "Could not delete label. Please try again.");
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

      renderFlash("success", successMessage);
      setResetSelectedRows(true);
      refetchHosts({
        selectedLabels: selectedFilters,
        globalFilter: searchQuery,
        sortBy,
        teamId: currentTeam?.id,
        policyId,
        policyResponse,
        softwareId,
        mdmId,
        mdmEnrollmentStatus,
        os_id,
        os_name,
        os_version,
      });

      toggleTransferHostModal();
      setSelectedHostIds([]);
      setIsAllMatchingHostsSelected(false);
    } catch (error) {
      renderFlash("error", "Could not transfer hosts. Please try again.");
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

      renderFlash("success", successMessage);
      setResetSelectedRows(true);
      refetchHosts({
        selectedLabels: selectedFilters,
        globalFilter: searchQuery,
        sortBy,
        teamId: currentTeam?.id,
        policyId,
        policyResponse,
        softwareId,
        mdmId,
        mdmEnrollmentStatus,
        os_id,
        os_name,
        os_version,
      });

      refetchLabels();
      toggleDeleteHostModal();
      setSelectedHostIds([]);
      setIsAllMatchingHostsSelected(false);
    } catch (error) {
      renderFlash(
        "error",
        `Could not delete ${
          selectedHostIds.length === 1 ? "host" : "hosts"
        }. Please try again.`
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

  const renderLabelFilterPill = () => {
    if (selectedLabel) {
      const { description, display_text, label_type } = selectedLabel;
      const pillLabel =
        PLATFORM_LABEL_DISPLAY_NAMES[display_text] ?? display_text;

      return (
        <>
          <FilterPill
            label={pillLabel}
            tooltipDescription={description}
            onClear={onClearLabelFilter}
          />
          {label_type !== "builtin" && !isOnlyObserver && (
            <>
              <Button onClick={onEditLabelClick} variant={"text-icon"}>
                <img src={PencilIcon} alt="Edit label" />
              </Button>
              <Button onClick={toggleDeleteLabelModal} variant={"text-icon"}>
                <img src={TrashIcon} alt="Delete label" />
              </Button>
            </>
          )}
        </>
      );
    }

    return null;
  };

  const renderOSFilterBlock = () => {
    if (!os_id && !(os_name && os_version)) return null;

    let os: IOperatingSystemVersion | undefined;
    if (os_id) {
      os = osVersions?.find((v) => v.os_id === os_id);
    } else if (os_name && os_version) {
      const name: string = os_name;
      const vers: string = os_version;

      os = osVersions?.find(
        ({ name_only, version }) =>
          name_only.toLowerCase() === name.toLowerCase() &&
          version.toLowerCase() === vers.toLowerCase()
      );
    }
    if (!os) return null;

    const { name, name_only, version } = os;
    const label =
      name_only || version
        ? `${name_only || ""} ${version || ""}`
        : `${name || ""}`;

    const TooltipDescription = (
      <span className={`tooltip__tooltip-text`}>
        {`Hosts with ${name_only || name}`},<br />
        {version && `${version} installed`}
      </span>
    );

    return (
      <FilterPill
        label={label}
        tooltipDescription={TooltipDescription}
        onClear={handleClearOSFilter}
      />
    );
  };

  const renderPoliciesFilterBlock = () => (
    <>
      <PoliciesFilter
        policyResponse={policyResponse}
        onChange={handleChangePoliciesFilter}
      />
      <FilterPill
        icon={PolicyIcon}
        label={policy?.name ?? ""}
        onClear={handleClearPoliciesFilter}
        className={`${baseClass}__policies-filter-pill`}
      />
    </>
  );

  const renderSoftwareFilterBlock = () => {
    if (!softwareDetails) return null;

    const { name, version } = softwareDetails;
    const label = name && version ? `${name} ${version}` : "";
    const TooltipDescription =
      name && version ? (
        <span className={`tooltip__tooltip-text`}>
          {`Hosts with ${name}`},<br />
          {`${version} installed`}
        </span>
      ) : undefined;

    return (
      <FilterPill
        label={label}
        onClear={handleClearSoftwareFilter}
        tooltipDescription={TooltipDescription}
      />
    );
  };

  const renderMDMSolutionFilterBlock = () => {
    if (!mdmSolutionDetails) return null;

    const { name, server_url } = mdmSolutionDetails;
    const label = name ? `${name} ${server_url}` : `${server_url}`;

    const TooltipDescription = (
      <span className={`tooltip__tooltip-text`}>
        Host enrolled
        {name !== "Unknown" && ` to ${name}`}
        <br /> at {server_url}
      </span>
    );

    return (
      <FilterPill
        label={label}
        tooltipDescription={TooltipDescription}
        onClear={handleClearMDMSolutionFilter}
      />
    );
  };

  const renderMDMEnrollmentFilterBlock = () => {
    if (!mdmEnrollmentStatus) return null;

    let label: string;
    switch (mdmEnrollmentStatus) {
      case "automatic":
        label = "MDM enrolled (automatic)";
        break;
      case "manual":
        label = "MDM enrolled (manual)";
        break;
      default:
        label = "Unenrolled";
    }

    let TooltipDescription: JSX.Element;
    switch (mdmEnrollmentStatus) {
      case "automatic":
        TooltipDescription = (
          <span className={`tooltip__tooltip-text`}>
            Hosts automatically enrolled <br />
            to an MDM solution the first time <br />
            the host is used. Administrators <br />
            might have a higher level of control <br />
            over these hosts.
          </span>
        );
        break;
      case "manual":
        TooltipDescription = (
          <span className={`tooltip__tooltip-text`}>
            Hosts manually enrolled to an <br />
            MDM solution by a user or <br />
            administrator.
          </span>
        );
        break;
      default:
        TooltipDescription = (
          <span className={`tooltip__tooltip-text`}>
            Hosts not enrolled to <br /> an MDM solution.
          </span>
        );
    }

    return (
      <FilterPill
        label={label}
        tooltipDescription={TooltipDescription}
        onClear={handleClearMDMEnrollmentFilter}
      />
    );
  };

  const renderEditColumnsModal = () => {
    if (!config || !currentUser) {
      return null;
    }

    return (
      <EditColumnsModal
        columns={generateAvailableTableHeaders(
          config,
          currentUser,
          currentTeam
        )}
        hiddenColumns={hiddenColumns}
        onSaveColumns={onSaveColumns}
        onCancelColumns={toggleEditColumnsModal}
      />
    );
  };

  const renderSecretEditorModal = () => (
    <SecretEditorModal
      selectedTeam={currentTeam?.id || 0}
      teams={teams || []}
      onSaveSecret={onSaveSecret}
      toggleSecretEditorModal={toggleSecretEditorModal}
      selectedSecret={selectedSecret}
    />
  );

  const renderDeleteSecretModal = () => (
    <DeleteSecretModal
      onDeleteSecret={onDeleteSecret}
      selectedTeam={currentTeam?.id || 0}
      teams={teams || []}
      toggleDeleteSecretModal={toggleDeleteSecretModal}
    />
  );

  const renderEnrollSecretModal = () => (
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

  const renderDeleteLabelModal = () => (
    <DeleteLabelModal
      onSubmit={onDeleteLabel}
      onCancel={toggleDeleteLabelModal}
    />
  );

  const renderAddHostsModal = () => {
    const enrollSecret =
      // TODO: Currently, prepacked installers in Fleet Sandbox use the global enroll secret,
      // and Fleet Sandbox runs Fleet Free so the isSandboxMode check here is an
      // additional precaution/reminder to revisit this in connection with future changes.
      // See https://github.com/fleetdm/fleet/issues/4970#issuecomment-1187679407.
      currentTeam && !isSandboxMode
        ? teamSecrets?.[0].secret
        : globalSecrets?.[0].secret;
    return (
      <AddHostsModal
        currentTeam={currentTeam}
        enrollSecret={enrollSecret}
        isLoading={isLoadingTeams || isGlobalSecretsLoading}
        isSandboxMode={!!isSandboxMode}
        onCancel={toggleAddHostsModal}
      />
    );
  };

  const renderTransferHostModal = () => {
    if (!teams) {
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

  const renderDeleteHostModal = () => (
    <DeleteHostModal
      selectedHostIds={selectedHostIds}
      onSubmit={onDeleteHostSubmit}
      onCancel={toggleDeleteHostModal}
      isAllMatchingHostsSelected={isAllMatchingHostsSelected}
    />
  );

  const renderHeader = () => (
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

  const onExportHostsResults = async (
    evt: React.MouseEvent<HTMLButtonElement>
  ) => {
    evt.preventDefault();

    const hiddenColumnsStorage = localStorage.getItem("hostHiddenColumns");
    let currentHiddenColumns = [];
    let visibleColumns;
    if (hiddenColumnsStorage) {
      currentHiddenColumns = JSON.parse(hiddenColumnsStorage);
    }

    if (config && currentUser) {
      const tableColumns = generateVisibleTableColumns(
        currentHiddenColumns,
        config,
        currentUser,
        currentTeam
      );

      const columnAccessors = tableColumns
        .map((column) => (column.accessor ? column.accessor : ""))
        .filter((element) => element);
      visibleColumns = columnAccessors.join(",");
    }

    let options = {
      selectedLabels: selectedFilters,
      globalFilter: searchQuery,
      sortBy,
      teamId: currentTeam?.id,
      policyId,
      policyResponse,
      softwareId,
      mdmId,
      mdmEnrollmentStatus,
      os_id,
      os_name,
      os_version,
      visibleColumns,
    };

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
      const exportHostResults = await hostsAPI.exportHosts(options);

      const formattedTime = format(new Date(), "yyyy-MM-dd");
      const filename = `${CSV_HOSTS_TITLE} ${formattedTime}.csv`;
      const file = new global.window.File([exportHostResults], filename, {
        type: "text/csv",
      });

      FileSaver.saveAs(file);
    } catch (error) {
      console.error(error);
      renderFlash("error", "Could not export hosts. Please try again.");
    }
  };

  const renderHostCount = useCallback(() => {
    const count = filteredHostCount;

    return (
      <div
        className={`${baseClass}__count ${
          isHostCountLoading ? "count-loading" : ""
        }`}
      >
        {count !== undefined && (
          <span>{`${count} host${count === 1 ? "" : "s"}`}</span>
        )}
        {count ? (
          <Button
            className={`${baseClass}__export-btn`}
            onClick={onExportHostsResults}
            variant="text-link"
          >
            <>
              Export hosts <img alt="" src={DownloadIcon} />
            </>
          </Button>
        ) : (
          <></>
        )}
      </div>
    );
  }, [isHostCountLoading, filteredHostCount]);

  const renderActiveFilterBlock = () => {
    const showSelectedLabel =
      selectedLabel &&
      selectedLabel.type !== "all" &&
      selectedLabel.type !== "status";
    if (
      policyId ||
      softwareId ||
      showSelectedLabel ||
      mdmId ||
      mdmEnrollmentStatus ||
      os_id ||
      (os_name && os_version)
    ) {
      return (
        <div className={`${baseClass}__labels-active-filter-wrap`}>
          {showSelectedLabel && renderLabelFilterPill()}
          {!!policyId &&
            !softwareId &&
            !mdmId &&
            !mdmEnrollmentStatus &&
            !showSelectedLabel &&
            renderPoliciesFilterBlock()}
          {!!softwareId &&
            !policyId &&
            !mdmId &&
            !mdmEnrollmentStatus &&
            !showSelectedLabel &&
            renderSoftwareFilterBlock()}
          {!!mdmId &&
            !policyId &&
            !softwareId &&
            !mdmEnrollmentStatus &&
            !showSelectedLabel &&
            renderMDMSolutionFilterBlock()}
          {!!mdmEnrollmentStatus &&
            !policyId &&
            !softwareId &&
            !mdmId &&
            !showSelectedLabel &&
            renderMDMEnrollmentFilterBlock()}
          {(!!os_id || (!!os_name && !!os_version)) &&
            !policyId &&
            !softwareId &&
            !showSelectedLabel &&
            !mdmId &&
            !mdmEnrollmentStatus &&
            renderOSFilterBlock()}
        </div>
      );
    }
    return null;
  };

  const renderForm = () => {
    if (isAddLabel) {
      return (
        <LabelForm
          onCancel={onCancelLabel}
          onOsqueryTableSelect={onOsqueryTableSelect}
          handleSubmit={onSaveAddLabel}
          baseError={labelsError?.message || ""}
          backendValidators={labelValidator}
        />
      );
    }

    if (isEditLabel) {
      return (
        <LabelForm
          selectedLabel={selectedLabel}
          onCancel={onCancelLabel}
          onOsqueryTableSelect={onOsqueryTableSelect}
          handleSubmit={onEditLabel}
          baseError={labelsError?.message || ""}
          backendValidators={labelValidator}
          isEdit
        />
      );
    }

    return false;
  };

  const renderCustomControls = () => {
    // we filter out the status labels as we dont want to display them in the label
    // filter select dropdown.
    // TODO: seperate labels and status into different data sets.
    const selectedDropdownLabel =
      selectedLabel?.type !== "all" && selectedLabel?.type !== "status"
        ? selectedLabel
        : undefined;

    return (
      <div className={`${baseClass}__filter-dropdowns`}>
        <Dropdown
          value={getStatusSelected() || ALL_HOSTS_LABEL}
          className={`${baseClass}__status_dropdown`}
          options={HOST_SELECT_STATUSES}
          searchable={false}
          onChange={handleStatusDropdownChange}
        />
        <LabelFilterSelect
          className={`${baseClass}__label-filter-dropdown`}
          labels={labels ?? []}
          canAddNewLabels={canAddNewLabels}
          selectedLabel={selectedDropdownLabel ?? null}
          onChange={handleLabelChange}
          onAddLabel={onAddLabelClick}
        />
      </div>
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
      const { software_id, policy_id, mdm_id, mdm_enrollment_status } =
        queryParams || {};
      const includesNameCardFilter = !!(
        software_id ||
        policy_id ||
        mdm_id ||
        mdm_enrollment_status ||
        os_id ||
        os_name ||
        os_version
      );

      return (
        <NoHosts
          toggleAddHostsModal={toggleAddHostsModal}
          canEnrollHosts={canEnrollHosts}
          includesNameCardFilter={includesNameCardFilter}
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

    const tableColumns = generateVisibleTableColumns(
      hiddenColumns,
      config,
      currentUser,
      currentTeam
    );

    return (
      <TableContainer
        columns={tableColumns}
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
        inputPlaceHolder={HOSTS_SEARCH_BOX_PLACEHOLDER}
        primarySelectActionButtonText={"Delete"}
        primarySelectActionButtonIcon={"delete"}
        primarySelectActionButtonVariant={"text-icon"}
        secondarySelectActions={secondarySelectActions}
        resultsTitle={"hosts"}
        showMarkAllPages
        isAllPagesSelected={isAllMatchingHostsSelected}
        searchable
        renderCount={renderHostCount}
        searchToolTipText={HOSTS_SEARCH_BOX_TOOLTIP}
        emptyComponent={EmptyHosts}
        customControl={renderCustomControls}
        onActionButtonClick={toggleEditColumnsModal}
        onPrimarySelectActionClick={onDeleteHostsClick}
        onQueryChange={onTableQueryChange}
        toggleAllPagesSelected={toggleAllMatchingHosts}
        disableNextPage={isLastPage}
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
    <>
      <MainContent>
        <>
          {renderForm()}
          {!isAddLabel && !isEditLabel && (
            <div className={`${baseClass}`}>
              <div className="header-wrap">
                {renderHeader()}
                <div className={`${baseClass} button-wrap`}>
                  {!isSandboxMode &&
                    canEnrollHosts &&
                    !hasHostErrors &&
                    !hasHostCountErrors && (
                      <Button
                        onClick={() => setShowEnrollSecretModal(true)}
                        className={`${baseClass}__enroll-hosts button`}
                        variant="inverse"
                      >
                        <span>Manage enroll secret</span>
                      </Button>
                    )}
                  {canEnrollHosts &&
                    !hasHostErrors &&
                    !hasHostCountErrors &&
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
              {renderNoEnrollSecretBanner()}
              {renderTable()}
            </div>
          )}
        </>
      </MainContent>
      {isAddLabel && (
        <SidePanelContent>
          <QuerySidePanel
            key="query-side-panel"
            onOsqueryTableSelect={onOsqueryTableSelect}
            selectedOsqueryTable={selectedOsqueryTable}
          />
        </SidePanelContent>
      )}

      {canEnrollHosts && showDeleteSecretModal && renderDeleteSecretModal()}
      {canEnrollHosts && showSecretEditorModal && renderSecretEditorModal()}
      {canEnrollHosts && showEnrollSecretModal && renderEnrollSecretModal()}
      {showEditColumnsModal && renderEditColumnsModal()}
      {showDeleteLabelModal && renderDeleteLabelModal()}
      {showAddHostsModal && renderAddHostsModal()}
      {showTransferHostModal && renderTransferHostModal()}
      {showDeleteHostModal && renderDeleteHostModal()}
    </>
  );
};

export default ManageHostsPage;
