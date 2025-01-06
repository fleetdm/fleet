import React, {
  useState,
  useContext,
  useEffect,
  useCallback,
  useMemo,
} from "react";
import { useQuery } from "react-query";
import { InjectedRouter, Params } from "react-router/lib/Router";
import { RouteProps } from "react-router/lib/Route";
import { find, isEmpty, isEqual, omit } from "lodash";
import { format } from "date-fns";
import FileSaver from "file-saver";

import enrollSecretsAPI from "services/entities/enroll_secret";
import usersAPI from "services/entities/users";
import labelsAPI, { ILabelsResponse } from "services/entities/labels";
import teamsAPI, { ILoadTeamsResponse } from "services/entities/teams";
import globalPoliciesAPI from "services/entities/global_policies";
import hostsAPI, {
  HOSTS_QUERY_PARAMS as PARAMS,
  ILoadHostsQueryKey,
  ILoadHostsResponse,
  ISortOption,
  MacSettingsStatusQueryParam,
  HOSTS_QUERY_PARAMS,
} from "services/entities/hosts";
import hostCountAPI, {
  IHostsCountQueryKey,
  IHostsCountResponse,
} from "services/entities/host_count";
import {
  getOSVersions,
  IGetOSVersionsQueryKey,
  IOSVersionsResponse,
} from "services/entities/operating_systems";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { TableContext } from "context/table";
import { NotificationContext } from "context/notification";

import useTeamIdParam from "hooks/useTeamIdParam";

import {
  IEnrollSecret,
  IEnrollSecretsResponse,
} from "interfaces/enroll_secret";
import { ILabel } from "interfaces/label";
import { IOperatingSystemVersion } from "interfaces/operating_system";
import { IPolicy, IStoredPolicyResponse } from "interfaces/policy";
import {
  isValidSoftwareAggregateStatus,
  SoftwareAggregateStatus,
} from "interfaces/software";
import { API_ALL_TEAMS_ID, ITeam } from "interfaces/team";
import { IEmptyTableProps } from "interfaces/empty_table";
import {
  DiskEncryptionStatus,
  BootstrapPackageStatus,
  MdmProfileStatus,
} from "interfaces/mdm";

import sortUtils from "utilities/sort";
import {
  HOSTS_SEARCH_BOX_PLACEHOLDER,
  HOSTS_SEARCH_BOX_TOOLTIP,
  PolicyResponse,
} from "utilities/constants";
import { getNextLocationPath } from "utilities/helpers";

import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import TableContainer from "components/TableContainer";
import InfoBanner from "components/InfoBanner/InfoBanner";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import TableDataError from "components/DataError";
import { IActionButtonProps } from "components/TableContainer/DataTable/ActionButton/ActionButton";
import TeamsDropdown from "components/TeamsDropdown";
import Spinner from "components/Spinner";
import MainContent from "components/MainContent";
import EmptyTable from "components/EmptyTable";
import {
  defaultHiddenColumns,
  generateVisibleTableColumns,
  generateAvailableTableHeaders,
} from "./HostTableConfig";
import {
  LABEL_SLUG_PREFIX,
  DEFAULT_SORT_HEADER,
  DEFAULT_SORT_DIRECTION,
  DEFAULT_PAGE_SIZE,
  DEFAULT_PAGE_INDEX,
  getHostSelectStatuses,
  MANAGE_HOSTS_PAGE_FILTER_KEYS,
  MANAGE_HOSTS_PAGE_LABEL_INCOMPATIBLE_QUERY_PARAMS,
} from "./HostsPageConfig";
import { getDeleteLabelErrorMessages, isAcceptableStatus } from "./helpers";

import DeleteSecretModal from "../../../components/EnrollSecrets/DeleteSecretModal";
import SecretEditorModal from "../../../components/EnrollSecrets/SecretEditorModal";
import AddHostsModal from "../../../components/AddHostsModal";
import EnrollSecretModal from "../../../components/EnrollSecrets/EnrollSecretModal";
// @ts-ignore
import EditColumnsModal from "./components/EditColumnsModal/EditColumnsModal";
import TransferHostModal from "../components/TransferHostModal";
import DeleteHostModal from "../components/DeleteHostModal";
import DeleteLabelModal from "./components/DeleteLabelModal";
import LabelFilterSelect from "./components/LabelFilterSelect";
import HostsFilterBlock from "./components/HostsFilterBlock";

interface IManageHostsProps {
  route: RouteProps;
  router: InjectedRouter;
  params: Params;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  location: any; // no type in react-router v3 TODO: Improve this type
}

const CSV_HOSTS_TITLE = "Hosts";
const baseClass = "manage-hosts";

const ManageHostsPage = ({
  route,
  router,
  params: routeParams,
  location,
}: IManageHostsProps): JSX.Element => {
  const routeTemplate = route?.path ?? "";
  const queryParams = location.query;
  const {
    config,
    currentUser,
    filteredHostsPath,
    isGlobalAdmin,
    isGlobalMaintainer,
    isOnGlobalTeam,
    isOnlyObserver,
    isPremiumTier,
    isFreeTier,
    isSandboxMode,
    userSettings,
    setFilteredHostsPath,
    setFilteredPoliciesPath,
    setFilteredQueriesPath,
    setFilteredSoftwarePath,
  } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const { setResetSelectedRows } = useContext(TableContext);

  const {
    currentTeamId,
    currentTeamName,
    isAnyTeamSelected,
    isRouteOk,
    isTeamAdmin,
    isTeamMaintainer,
    isTeamMaintainerOrTeamAdmin,
    teamIdForApi,
    userTeams,
    handleTeamChange,
  } = useTeamIdParam({
    location,
    router,
    includeAllTeams: true,
    includeNoTeam: true,
    overrideParamsOnTeamChange: {
      // remove the software status filter when selecting All teams
      [HOSTS_QUERY_PARAMS.SOFTWARE_STATUS]: (newTeamId?: number) =>
        newTeamId === API_ALL_TEAMS_ID,
    },
  });

  // Functions to avoid race conditions
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
  const initialQuery = (() => queryParams.query ?? "")();
  const initialPage = (() =>
    queryParams && queryParams.page ? parseInt(queryParams?.page, 10) : 0)();

  // ========= states
  const [selectedLabel, setSelectedLabel] = useState<ILabel>();
  const [selectedSecret, setSelectedSecret] = useState<IEnrollSecret>();
  const [showNoEnrollSecretBanner, setShowNoEnrollSecretBanner] = useState(
    true
  );
  const [showDeleteSecretModal, setShowDeleteSecretModal] = useState(false);
  const [showSecretEditorModal, setShowSecretEditorModal] = useState(false);
  const [showEnrollSecretModal, setShowEnrollSecretModal] = useState(false);
  const [showDeleteLabelModal, setShowDeleteLabelModal] = useState(false);
  const [showEditColumnsModal, setShowEditColumnsModal] = useState(false);
  const [showAddHostsModal, setShowAddHostsModal] = useState(false);
  const [showTransferHostModal, setShowTransferHostModal] = useState(false);
  const [showDeleteHostModal, setShowDeleteHostModal] = useState(false);
  const [hiddenColumns, setHiddenColumns] = useState<string[]>(
    userSettings?.hidden_hosts_table_columns || defaultHiddenColumns
  );
  const [selectedHostIds, setSelectedHostIds] = useState<number[]>([]);
  const [isAllMatchingHostsSelected, setIsAllMatchingHostsSelected] = useState(
    false
  );
  const [searchQuery, setSearchQuery] = useState(initialQuery);
  const [page, setPage] = useState(initialPage);
  const [sortBy, setSortBy] = useState<ISortOption[]>(initialSortBy);
  const [tableQueryData, setTableQueryData] = useState<ITableQueryData>();
  const [resetPageIndex, setResetPageIndex] = useState<boolean>(false);
  const [isUpdatingLabel, setIsUpdatingLabel] = useState<boolean>(false);
  const [isUpdatingSecret, setIsUpdatingSecret] = useState<boolean>(false);
  const [isUpdatingHosts, setIsUpdatingHosts] = useState<boolean>(false);

  // ========= queryParams
  const policyId = queryParams?.policy_id;
  const policyResponse: PolicyResponse = queryParams?.policy_response;
  const macSettingsStatus = queryParams?.macos_settings;
  const softwareId =
    queryParams?.software_id !== undefined
      ? parseInt(queryParams.software_id, 10)
      : undefined;
  const softwareVersionId =
    queryParams?.software_version_id !== undefined
      ? parseInt(queryParams.software_version_id, 10)
      : undefined;
  const softwareTitleId =
    queryParams?.software_title_id !== undefined
      ? parseInt(queryParams.software_title_id, 10)
      : undefined;
  const softwareStatus = isValidSoftwareAggregateStatus(
    queryParams?.[HOSTS_QUERY_PARAMS.SOFTWARE_STATUS]
  )
    ? (queryParams[
        HOSTS_QUERY_PARAMS.SOFTWARE_STATUS
      ] as SoftwareAggregateStatus)
    : undefined;
  const status = isAcceptableStatus(queryParams?.status)
    ? queryParams?.status
    : undefined;
  const mdmId =
    queryParams?.mdm_id !== undefined
      ? parseInt(queryParams.mdm_id, 10)
      : undefined;
  const mdmEnrollmentStatus = queryParams?.mdm_enrollment_status;
  const {
    os_version_id: osVersionId,
    os_name: osName,
    os_version: osVersion,
  } = queryParams;
  const vulnerability = queryParams?.vulnerability;
  const munkiIssueId =
    queryParams?.munki_issue_id !== undefined
      ? parseInt(queryParams.munki_issue_id, 10)
      : undefined;
  const lowDiskSpaceHosts =
    queryParams?.low_disk_space !== undefined
      ? parseInt(queryParams.low_disk_space, 10)
      : undefined;
  const missingHosts = queryParams?.status === "missing";
  const osSettingsStatus = queryParams?.[PARAMS.OS_SETTINGS];
  const diskEncryptionStatus: DiskEncryptionStatus | undefined =
    queryParams?.[PARAMS.DISK_ENCRYPTION];
  const bootstrapPackageStatus: BootstrapPackageStatus | undefined =
    queryParams?.bootstrap_package;

  // ========= routeParams
  const { active_label: activeLabel, label_id: labelID } = routeParams;
  const selectedFilters = useMemo(() => {
    const filters: string[] = [];
    labelID && filters.push(`${LABEL_SLUG_PREFIX}${labelID}`);
    activeLabel && filters.push(activeLabel);
    return filters;
  }, [activeLabel, labelID]);

  // ========= derived permissions
  const canEnrollHosts =
    isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer;
  const canEnrollGlobalHosts = isGlobalAdmin || isGlobalMaintainer;
  const canAddNewLabels = (isGlobalAdmin || isGlobalMaintainer) ?? false;

  const { data: labels, refetch: refetchLabels } = useQuery<
    ILabelsResponse,
    Error,
    ILabel[]
  >(["labels"], () => labelsAPI.loadAll(), {
    enabled: isRouteOk,
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
      enabled: isRouteOk && !!canEnrollGlobalHosts,
      select: (data: IEnrollSecretsResponse) => data.secrets,
    }
  );

  const {
    isLoading: isTeamSecretsLoading,
    data: teamSecrets,
    refetch: refetchTeamSecrets,
  } = useQuery<IEnrollSecretsResponse, Error, IEnrollSecret[]>(
    ["team secrets", currentTeamId],
    () => {
      if (isAnyTeamSelected) {
        return enrollSecretsAPI.getTeamEnrollSecrets(currentTeamId);
      }
      return { secrets: [] };
    },
    {
      enabled: isRouteOk && isAnyTeamSelected && canEnrollHosts,
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
      enabled: isRouteOk && !!isPremiumTier,
      select: (data: ILoadTeamsResponse) =>
        data.teams.sort((a, b) => sortUtils.caseInsensitiveAsc(a.name, b.name)),
    }
  );

  const {
    data: policy,
    isLoading: isLoadingPolicy,
    error: errorPolicy,
  } = useQuery<IStoredPolicyResponse, Error, IPolicy>(
    ["policy", policyId],
    () => globalPoliciesAPI.load(policyId),
    {
      enabled: isRouteOk && !!policyId,
      select: (data) => data.policy,
    }
  );

  const { data: osVersions } = useQuery<
    IOSVersionsResponse,
    Error,
    IOperatingSystemVersion[],
    IGetOSVersionsQueryKey[]
  >([{ scope: "os_versions" }], () => getOSVersions(), {
    enabled:
      isRouteOk &&
      (!!queryParams?.os_version_id ||
        (!!queryParams?.os_name && !!queryParams?.os_version)),
    keepPreviousData: true,
    select: (data) => data.os_versions,
  });

  const {
    data: hostsData,
    error: errorHosts,
    isFetching: isLoadingHosts,
    refetch: refetchHostsAPI,
  } = useQuery<
    ILoadHostsResponse,
    Error,
    ILoadHostsResponse,
    ILoadHostsQueryKey[]
  >(
    [
      {
        scope: "hosts",
        selectedLabels: selectedFilters,
        globalFilter: searchQuery,
        sortBy,
        teamId: teamIdForApi,
        policyId,
        policyResponse,
        softwareId,
        softwareTitleId,
        softwareVersionId,
        softwareStatus,
        status,
        mdmId,
        mdmEnrollmentStatus,
        munkiIssueId,
        lowDiskSpaceHosts,
        osVersionId,
        osName,
        osVersion,
        vulnerability,
        page: tableQueryData ? tableQueryData.pageIndex : DEFAULT_PAGE_INDEX,
        perPage: tableQueryData ? tableQueryData.pageSize : DEFAULT_PAGE_SIZE,
        device_mapping: true,
        osSettings: osSettingsStatus,
        diskEncryptionStatus,
        bootstrapPackageStatus,
        macSettingsStatus,
      },
    ],
    ({ queryKey }) => hostsAPI.loadHosts(queryKey[0]),
    {
      enabled: isRouteOk,
      keepPreviousData: true,
      staleTime: 10000, // stale time can be adjusted if fresher data is desired
    }
  );

  const {
    data: hostsCount,
    error: errorHostsCount,
    isFetching: isLoadingHostsCount,
    refetch: refetchHostsCountAPI,
  } = useQuery<IHostsCountResponse, Error, number, IHostsCountQueryKey[]>(
    [
      {
        scope: "hosts_count",
        selectedLabels: selectedFilters,
        globalFilter: searchQuery,
        teamId: teamIdForApi,
        policyId,
        policyResponse,
        softwareId,
        softwareTitleId,
        softwareVersionId,
        softwareStatus,
        status,
        mdmId,
        mdmEnrollmentStatus,
        munkiIssueId,
        lowDiskSpaceHosts,
        osVersionId,
        osName,
        osVersion,
        vulnerability,
        osSettings: osSettingsStatus,
        diskEncryptionStatus,
        bootstrapPackageStatus,
        macSettingsStatus,
      },
    ],
    ({ queryKey }) => hostCountAPI.load(queryKey[0]),
    {
      enabled: isRouteOk,
      keepPreviousData: true,
      staleTime: 10000, // stale time can be adjusted if fresher data is desired
      select: (data) => data.count,
    }
  );

  const refetchHosts = () => {
    refetchHostsAPI();
    refetchHostsCountAPI();
  };

  const hasErrors = !!errorHosts || !!errorHostsCount || !!errorPolicy;

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

  // TODO: cleanup this effect
  useEffect(() => {
    setShowNoEnrollSecretBanner(true);
  }, [teamIdForApi]);

  // TODO: cleanup this effect
  useEffect(() => {
    const slugToFind =
      (selectedFilters.length > 0 &&
        selectedFilters.find((f) => f.includes(LABEL_SLUG_PREFIX))) ||
      selectedFilters[0];
    const validLabel = find(labels, ["slug", slugToFind]) as ILabel;
    if (selectedLabel !== validLabel) {
      setSelectedLabel(validLabel);
    }
  }, [labels, selectedFilters, selectedLabel]);

  // TODO: cleanup this effect
  useEffect(() => {
    if (
      location.search.match(
        /software_id|software_version_id|software_title_id|software_status/gi
      )
    ) {
      // regex matches any of "software_id", "software_version_id", "software_title_id", or "software_status"
      // so we don't set the filtered hosts path in those cases
      return;
    }
    const path = location.pathname + location.search;
    if (filteredHostsPath !== path) {
      setFilteredHostsPath(path);
    }
  }, [filteredHostsPath, location, setFilteredHostsPath]);

  const isLastPage =
    tableQueryData &&
    !!hostsCount &&
    DEFAULT_PAGE_SIZE * tableQueryData.pageIndex +
      (hostsData?.hosts?.length || 0) >=
      hostsCount;

  const handleLabelChange = ({ slug, id: newLabelId }: ILabel): boolean => {
    const { MANAGE_HOSTS } = PATHS;

    const isDeselectingLabel = newLabelId && newLabelId === selectedLabel?.id;

    let newQueryParams = queryParams;
    if (slug) {
      // some filters are incompatible with non-status labels so omit those params from next location
      newQueryParams = omit(
        newQueryParams,
        MANAGE_HOSTS_PAGE_LABEL_INCOMPATIBLE_QUERY_PARAMS
      );
    }

    router.replace(
      getNextLocationPath({
        pathPrefix: isDeselectingLabel
          ? MANAGE_HOSTS
          : `${MANAGE_HOSTS}/${slug}`,
        queryParams: newQueryParams,
      })
    );

    return true;
  };

  // NOTE: Solution also used on ManagePoliciesPage.tsx
  // NOTE: used to reset page number to 0 when modifying filters
  useEffect(() => {
    setResetPageIndex(false);
  }, [queryParams, page]);

  // NOTE: used to reset page number to 0 when modifying filters
  const handleResetPageIndex = () => {
    setTableQueryData(
      (prevState) =>
        ({
          ...prevState,
          pageIndex: 0,
        } as ITableQueryData)
    );
    setResetPageIndex(true);
  };

  const handleChangePoliciesFilter = (response: PolicyResponse) => {
    handleResetPageIndex();

    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.MANAGE_HOSTS,
        routeTemplate,
        routeParams,
        queryParams: {
          ...queryParams,
          policy_id: policyId,
          policy_response: response,
          page: 0, // resets page index
        },
      })
    );
  };

  const handleChangeDiskEncryptionStatusFilter = (
    newStatus: DiskEncryptionStatus
  ) => {
    handleResetPageIndex();

    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.MANAGE_HOSTS,
        routeTemplate,
        routeParams,
        queryParams: {
          ...queryParams,
          [PARAMS.DISK_ENCRYPTION]: newStatus,
          page: 0, // resets page index
        },
      })
    );
  };

  const handleChangeOsSettingsFilter = (newStatus: MdmProfileStatus) => {
    handleResetPageIndex();

    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.MANAGE_HOSTS,
        routeTemplate,
        routeParams,
        queryParams: {
          ...queryParams,
          [PARAMS.OS_SETTINGS]: newStatus,
          page: 0, // resets page index
        },
      })
    );
  };

  const handleChangeBootstrapPackageStatusFilter = (
    newStatus: BootstrapPackageStatus
  ) => {
    handleResetPageIndex();

    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.MANAGE_HOSTS,
        routeTemplate,
        routeParams,
        queryParams: { ...queryParams, bootstrap_package: newStatus },
      })
    );
  };

  const handleClearRouteParam = () => {
    handleResetPageIndex();

    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.MANAGE_HOSTS,
        routeTemplate,
        routeParams: undefined,
        queryParams: {
          ...queryParams,
          page: 0, // resets page index
        },
      })
    );
  };

  const handleClearFilter = (omitParams: string[]) => {
    handleResetPageIndex();

    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.MANAGE_HOSTS,
        routeTemplate,
        routeParams,
        queryParams: {
          ...omit(queryParams, omitParams),
          page: 0, // resets page index
        },
      })
    );
  };

  const handleStatusDropdownChange = (statusName: string) => {
    handleResetPageIndex();

    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.MANAGE_HOSTS,
        routeTemplate,
        routeParams,
        queryParams: {
          ...queryParams,
          status: statusName,
          page: 0, // resets page index
        },
      })
    );
  };

  const handleMacSettingsStatusDropdownChange = (
    newMacSettingsStatus: MacSettingsStatusQueryParam
  ) => {
    handleResetPageIndex();

    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.MANAGE_HOSTS,
        routeTemplate,
        routeParams,
        queryParams: {
          ...queryParams,
          macos_settings: newMacSettingsStatus,
          page: 0, // resets page index
        },
      })
    );
  };

  const handleSoftwareInstallStatusChange = (
    newStatus: SoftwareAggregateStatus
  ) => {
    handleResetPageIndex();

    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.MANAGE_HOSTS,
        routeTemplate,
        routeParams,
        queryParams: {
          ...queryParams,
          [HOSTS_QUERY_PARAMS.SOFTWARE_STATUS]: newStatus,
          page: 0, // resets page index
        },
      })
    );
  };

  const onAddLabelClick = () => {
    router.push(`${PATHS.NEW_LABEL}`);
  };

  const onEditLabelClick = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();
    router.push(`${PATHS.EDIT_LABEL(parseInt(labelID, 10))}`);
  };

  const onSaveColumns = async (newHiddenColumns: string[]) => {
    if (!currentUser) {
      return;
    }
    try {
      await usersAPI.update(currentUser.id, {
        settings: { hidden_hosts_table_columns: newHiddenColumns },
      });
      // No success renderFlash, to make column setting more seamless
    } catch (response) {
      renderFlash("error", "Couldn't save column settings. Please try again.");
    }

    setHiddenColumns(newHiddenColumns);
    setShowEditColumnsModal(false);
  };

  // NOTE: this is called once on initial render and every time the query changes
  const onTableQueryChange = useCallback(
    async (newTableQuery: ITableQueryData) => {
      if (!isRouteOk || isEqual(newTableQuery, tableQueryData)) {
        return;
      }

      setTableQueryData({ ...newTableQuery });

      const {
        searchQuery: searchText,
        sortHeader,
        sortDirection,
        pageIndex,
      } = newTableQuery;

      let sort = sortBy;
      if (sortHeader) {
        let direction = sortDirection;
        if (sortHeader === "last_restarted_at") {
          if (sortDirection === "asc") {
            direction = "desc";
          } else {
            direction = "asc";
          }
        }
        sort = [
          {
            key: sortHeader,
            direction: direction || DEFAULT_SORT_DIRECTION,
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

      if (!isEqual(page, pageIndex)) {
        setPage(pageIndex);
      }

      // Rebuild queryParams to dispatch new browser location to react-router
      const newQueryParams: { [key: string]: string | number | undefined } = {};
      if (!isEmpty(searchText)) {
        newQueryParams.query = searchText;
      }
      newQueryParams.page = pageIndex;
      newQueryParams.order_key = sort[0].key || DEFAULT_SORT_HEADER;
      newQueryParams.order_direction =
        sort[0].direction || DEFAULT_SORT_DIRECTION;

      newQueryParams.team_id = teamIdForApi;

      if (status) {
        newQueryParams.status = status;
      }
      if (policyId && policyResponse) {
        newQueryParams.policy_id = policyId;
        newQueryParams.policy_response = policyResponse;
      } else if (macSettingsStatus) {
        newQueryParams.macos_settings = macSettingsStatus;
      } else if (softwareId) {
        newQueryParams.software_id = softwareId;
      } else if (softwareVersionId) {
        newQueryParams.software_version_id = softwareVersionId;
      } else if (softwareTitleId) {
        newQueryParams.software_title_id = softwareTitleId;
        if (softwareStatus && teamIdForApi !== API_ALL_TEAMS_ID) {
          // software_status is only valid when software_title_id is present and a subset of hosts ('No team' or a team) is selected
          newQueryParams[HOSTS_QUERY_PARAMS.SOFTWARE_STATUS] = softwareStatus;
        }
      } else if (mdmId) {
        newQueryParams.mdm_id = mdmId;
      } else if (mdmEnrollmentStatus) {
        newQueryParams.mdm_enrollment_status = mdmEnrollmentStatus;
      } else if (munkiIssueId) {
        newQueryParams.munki_issue_id = munkiIssueId;
      } else if (missingHosts) {
        // Premium feature only
        newQueryParams.status = "missing";
      } else if (lowDiskSpaceHosts && isPremiumTier) {
        // Premium feature only
        newQueryParams.low_disk_space = lowDiskSpaceHosts;
      } else if (osVersionId || (osName && osVersion)) {
        newQueryParams.os_version_id = osVersionId;
        newQueryParams.os_name = osName;
        newQueryParams.os_version = osVersion;
      } else if (vulnerability) {
        newQueryParams.vulnerability = vulnerability;
      } else if (osSettingsStatus) {
        newQueryParams[PARAMS.OS_SETTINGS] = osSettingsStatus;
      } else if (diskEncryptionStatus && isPremiumTier) {
        // Premium feature only
        newQueryParams[PARAMS.DISK_ENCRYPTION] = diskEncryptionStatus;
      } else if (bootstrapPackageStatus && isPremiumTier) {
        newQueryParams.bootstrap_package = bootstrapPackageStatus;
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
      isRouteOk,
      tableQueryData,
      sortBy,
      searchQuery,
      teamIdForApi,
      status,
      policyId,
      policyResponse,
      macSettingsStatus,
      softwareId,
      softwareVersionId,
      softwareTitleId,
      softwareStatus,
      mdmId,
      mdmEnrollmentStatus,
      munkiIssueId,
      missingHosts,
      lowDiskSpaceHosts,
      isPremiumTier,
      osVersionId,
      osName,
      osVersion,
      page,
      router,
      routeTemplate,
      routeParams,
      osSettingsStatus,
      diskEncryptionStatus,
      bootstrapPackageStatus,
      vulnerability,
    ]
  );

  const onTeamChange = useCallback(
    (teamId: number) => {
      // TODO(sarah): refactor so that this doesn't trigger two api calls (reset page index updates
      // tableQueryData)
      handleTeamChange(teamId);
      handleResetPageIndex();
      // Must clear other page paths or the team might accidentally switch
      // When navigating from host details
      setFilteredSoftwarePath("");
      setFilteredQueriesPath("");
      setFilteredPoliciesPath("");
    },
    [handleTeamChange]
  );

  const onSaveSecret = async (enrollSecretString: string) => {
    const { MANAGE_HOSTS } = PATHS;

    // Creates new list of secrets removing selected secret and adding new secret
    const currentSecrets = isAnyTeamSelected
      ? teamSecrets || []
      : globalSecrets || [];

    const newSecrets = currentSecrets.filter(
      (s) => s.secret !== selectedSecret?.secret
    );

    if (enrollSecretString) {
      newSecrets.push({ secret: enrollSecretString });
    }

    setIsUpdatingSecret(true);

    try {
      if (isAnyTeamSelected) {
        await enrollSecretsAPI.modifyTeamEnrollSecrets(
          currentTeamId,
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
    } finally {
      setIsUpdatingSecret(false);
    }
  };

  const onDeleteSecret = async () => {
    const { MANAGE_HOSTS } = PATHS;

    // create new list of secrets removing selected secret
    const currentSecrets = isAnyTeamSelected
      ? teamSecrets || []
      : globalSecrets || [];

    const newSecrets = currentSecrets.filter(
      (s) => s.secret !== selectedSecret?.secret
    );

    setIsUpdatingSecret(true);

    try {
      if (isAnyTeamSelected) {
        await enrollSecretsAPI.modifyTeamEnrollSecrets(
          currentTeamId,
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
    } finally {
      setIsUpdatingSecret(false);
    }
  };

  const onDeleteLabel = async () => {
    if (!selectedLabel) {
      console.error("Label isn't available. This should not happen.");
      return false;
    }
    setIsUpdatingLabel(true);

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
      renderFlash("success", "Successfully deleted label.");
    } catch (error) {
      renderFlash("error", getDeleteLabelErrorMessages(error));
    } finally {
      setIsUpdatingLabel(false);
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

  // Bulk transfer is hidden for defined unsupportedFilters
  const onTransferHostSubmit = async (transferTeam: ITeam) => {
    setIsUpdatingHosts(true);

    const teamId = typeof transferTeam.id === "number" ? transferTeam.id : null;

    const action = isAllMatchingHostsSelected
      ? hostsAPI.transferToTeamByFilter({
          teamId,
          query: searchQuery,
          status,
          labelId: selectedLabel?.id,
          currentTeam: teamIdForApi,
          policyId,
          policyResponse,
          softwareId,
          softwareTitleId,
          softwareVersionId,
          softwareStatus,
          osName,
          osVersionId,
          osVersion,
          macSettingsStatus,
          bootstrapPackageStatus,
          mdmId,
          mdmEnrollmentStatus,
          munkiIssueId,
          lowDiskSpaceHosts,
          osSettings: osSettingsStatus,
          diskEncryptionStatus,
          vulnerability,
        })
      : hostsAPI.transferToTeam(teamId, selectedHostIds);

    try {
      await action;

      const successMessage =
        teamId === null
          ? `Hosts successfully removed from teams.`
          : `Hosts successfully transferred to  ${transferTeam.name}.`;

      renderFlash("success", successMessage);
      setResetSelectedRows(true);
      refetchHosts();
      toggleTransferHostModal();
      setSelectedHostIds([]);
      setIsAllMatchingHostsSelected(false);
    } catch (error) {
      renderFlash("error", "Could not transfer hosts. Please try again.");
    } finally {
      setIsUpdatingHosts(false);
    }
  };

  // Bulk delete is hidden for defined unsupportedFilters
  const onDeleteHostSubmit = async () => {
    setIsUpdatingHosts(true);

    try {
      await (isAllMatchingHostsSelected
        ? hostsAPI.destroyByFilter({
            teamId: teamIdForApi,
            query: searchQuery,
            status,
            labelId: selectedLabel?.id,
            policyId,
            policyResponse,
            softwareId,
            softwareTitleId,
            softwareVersionId,
            softwareStatus,
            osName,
            osVersionId,
            osVersion,
            macSettingsStatus,
            bootstrapPackageStatus,
            mdmId,
            mdmEnrollmentStatus,
            munkiIssueId,
            lowDiskSpaceHosts,
            osSettings: osSettingsStatus,
            diskEncryptionStatus,
            vulnerability,
          })
        : hostsAPI.destroyBulk(selectedHostIds));

      const successMessage = `${
        selectedHostIds.length === 1 ? "Host" : "Hosts"
      } successfully deleted.`;

      renderFlash("success", successMessage);
      setResetSelectedRows(true);
      refetchHosts();
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
    } finally {
      setIsUpdatingHosts(false);
    }
  };

  const renderTeamsFilterDropdown = () => (
    <TeamsDropdown
      currentUserTeams={userTeams || []}
      selectedTeamId={currentTeamId}
      isDisabled={isLoadingHosts || isLoadingHostsCount} // TODO: why?
      onChange={onTeamChange}
      includeNoTeams
    />
  );

  const renderEditColumnsModal = () => {
    if (!config || !currentUser) {
      return null;
    }

    return (
      <EditColumnsModal
        columns={generateAvailableTableHeaders({ isFreeTier, isOnlyObserver })}
        hiddenColumns={hiddenColumns}
        onSaveColumns={onSaveColumns}
        onCancelColumns={toggleEditColumnsModal}
      />
    );
  };

  const renderSecretEditorModal = () => (
    <SecretEditorModal
      selectedTeam={teamIdForApi || 0}
      teams={teams || []}
      onSaveSecret={onSaveSecret}
      toggleSecretEditorModal={toggleSecretEditorModal}
      selectedSecret={selectedSecret}
      isUpdatingSecret={isUpdatingSecret}
    />
  );

  const renderDeleteSecretModal = () => (
    <DeleteSecretModal
      onDeleteSecret={onDeleteSecret}
      selectedTeam={teamIdForApi || 0}
      teams={teams || []}
      toggleDeleteSecretModal={toggleDeleteSecretModal}
      isUpdatingSecret={isUpdatingSecret}
    />
  );

  const renderEnrollSecretModal = () => (
    <EnrollSecretModal
      selectedTeam={teamIdForApi || 0}
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
      isUpdatingLabel={isUpdatingLabel}
    />
  );

  const renderAddHostsModal = () => {
    const enrollSecret = isAnyTeamSelected
      ? teamSecrets?.[0].secret
      : globalSecrets?.[0].secret;
    return (
      <AddHostsModal
        currentTeamName={currentTeamName || "Fleet"}
        enrollSecret={enrollSecret}
        isAnyTeamSelected={isAnyTeamSelected}
        isLoading={isLoadingTeams || isGlobalSecretsLoading}
        onCancel={toggleAddHostsModal}
        openEnrollSecretModal={() => setShowEnrollSecretModal(true)}
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
        isUpdating={isUpdatingHosts}
        multipleHosts={selectedHostIds.length > 1}
      />
    );
  };

  const renderDeleteHostModal = () => (
    <DeleteHostModal
      selectedHostIds={selectedHostIds}
      onSubmit={onDeleteHostSubmit}
      onCancel={toggleDeleteHostModal}
      isAllMatchingHostsSelected={isAllMatchingHostsSelected}
      hostsCount={hostsCount}
      isUpdating={isUpdatingHosts}
    />
  );

  const renderHeader = () => (
    <div className={`${baseClass}__header`}>
      <div className={`${baseClass}__text`}>
        <div className={`${baseClass}__title`}>
          {isFreeTier && <h1>Hosts</h1>}
          {isPremiumTier &&
            userTeams &&
            (userTeams.length > 1 || isOnGlobalTeam) &&
            renderTeamsFilterDropdown()}
          {isPremiumTier &&
            !isOnGlobalTeam &&
            userTeams &&
            userTeams.length === 1 && <h1>{userTeams[0].name}</h1>}
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
      const tableColumns = generateVisibleTableColumns({
        hiddenColumns: currentHiddenColumns,
        isFreeTier,
        isOnlyObserver,
      });

      const columnIds = tableColumns
        .map((column) => (column.id ? column.id : ""))
        // "selection" colum does not include any relevent data for the CSV
        // so we filter it out.
        .filter((element) => element !== "" && element !== "selection");
      visibleColumns = columnIds.join(",");
    }

    let options = {
      selectedLabels: selectedFilters,
      globalFilter: searchQuery,
      sortBy,
      teamId: teamIdForApi,
      policyId,
      policyResponse,
      macSettingsStatus,
      softwareId,
      softwareTitleId,
      softwareVersionId,
      softwareStatus,
      status,
      mdmId,
      mdmEnrollmentStatus,
      munkiIssueId,
      lowDiskSpaceHosts,
      osName,
      osVersionId,
      osVersion,
      osSettings: osSettingsStatus,
      bootstrapPackageStatus,
      vulnerability,
      visibleColumns,
    };

    options = {
      ...options,
      teamId: teamIdForApi,
    };

    if (
      queryParams.team_id !== API_ALL_TEAMS_ID &&
      queryParams.team_id !== ""
    ) {
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
    return (
      <>
        <TableCount name="hosts" count={hostsCount} />
        {!!hostsCount && (
          <Button
            className={`${baseClass}__export-btn`}
            onClick={onExportHostsResults}
            variant="text-icon"
          >
            <>
              Export hosts
              <Icon name="download" size="small" color="core-fleet-blue" />
            </>
          </Button>
        )}
      </>
    );
  }, [isLoadingHostsCount, hostsCount]);

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
          value={status || ""}
          className={`${baseClass}__status_dropdown`}
          options={getHostSelectStatuses(isSandboxMode)}
          searchable={false}
          onChange={handleStatusDropdownChange}
          iconName="filter"
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

  // TODO: try to reduce overlap between maybeEmptyHosts and includesFilterQueryParam
  const maybeEmptyHosts =
    hostsCount === 0 && searchQuery === "" && !labelID && !status;

  const includesFilterQueryParam = MANAGE_HOSTS_PAGE_FILTER_KEYS.some(
    (filter) =>
      filter !== "team_id" &&
      typeof queryParams === "object" &&
      filter in queryParams // TODO: replace this with `Object.hasOwn(queryParams, filter)` when we upgrade to es2022
  );

  const renderTable = () => {
    if (!config || !currentUser || !isRouteOk) {
      return <Spinner />;
    }

    if (hasErrors) {
      return <TableDataError />;
    }
    if (maybeEmptyHosts) {
      const emptyState = () => {
        const emptyHosts: IEmptyTableProps = {
          graphicName: "empty-hosts",
          header: "Hosts will show up here once they’re added to Fleet",
          info:
            "Expecting to see hosts? Try again in a few seconds as the system catches up.",
        };
        if (includesFilterQueryParam) {
          delete emptyHosts.graphicName;
          emptyHosts.header = "No hosts match the current criteria";
          emptyHosts.info =
            "Expecting to see new hosts? Try again in a few seconds as the system catches up.";
        } else if (canEnrollHosts) {
          emptyHosts.header = "Add your hosts to Fleet";
          emptyHosts.info =
            "Generate Fleet's agent (fleetd) to add your own hosts.";
          emptyHosts.primaryButton = (
            <Button variant="brand" onClick={toggleAddHostsModal} type="button">
              Add hosts
            </Button>
          );
        }
        return emptyHosts;
      };

      return (
        <>
          {EmptyTable({
            graphicName: emptyState().graphicName,
            header: emptyState().header,
            info: emptyState().info,
            additionalInfo: emptyState().additionalInfo,
            primaryButton: emptyState().primaryButton,
          })}
        </>
      );
    }

    const secondarySelectActions: IActionButtonProps[] = [
      {
        name: "transfer",
        onActionButtonClick: onTransferToTeamClick,
        buttonText: "Transfer",
        variant: "text-icon",
        iconSvg: "transfer",
        hideButton: !isPremiumTier || (!isGlobalAdmin && !isGlobalMaintainer),
        indicatePremiumFeature: isPremiumTier && isSandboxMode,
      },
    ];

    const tableColumns = generateVisibleTableColumns({
      hiddenColumns,
      isFreeTier,
      isOnlyObserver:
        isOnlyObserver || (!isOnGlobalTeam && !isTeamMaintainerOrTeamAdmin),
    });

    const emptyState = () => {
      const emptyHosts: IEmptyTableProps = {
        header: "No hosts match the current criteria",
        info:
          "Expecting to see new hosts? Try again in a few seconds as the system catches up.",
      };
      if (isLastPage) {
        emptyHosts.header = "No more hosts to display";
        emptyHosts.info =
          "Expecting to see more hosts? Try again in a few seconds as the system catches up.";
      }

      return emptyHosts;
    };

    // Shortterm fix for #17257
    const unsupportedFilter = !!(
      policyId ||
      policyResponse ||
      softwareId ||
      softwareTitleId ||
      softwareVersionId ||
      osName ||
      osVersionId ||
      osVersion ||
      macSettingsStatus ||
      bootstrapPackageStatus ||
      mdmId ||
      mdmEnrollmentStatus ||
      munkiIssueId ||
      lowDiskSpaceHosts ||
      osSettingsStatus ||
      diskEncryptionStatus ||
      vulnerability
    );

    return (
      <TableContainer
        resultsTitle="hosts"
        columnConfigs={tableColumns}
        data={hostsData?.hosts || []}
        isLoading={isLoadingHosts || isLoadingHostsCount || isLoadingPolicy}
        manualSortBy
        defaultSortHeader={(sortBy[0] && sortBy[0].key) || DEFAULT_SORT_HEADER}
        defaultSortDirection={
          (sortBy[0] && sortBy[0].direction) || DEFAULT_SORT_DIRECTION
        }
        defaultPageIndex={page || DEFAULT_PAGE_INDEX}
        defaultSearchQuery={searchQuery}
        pageSize={DEFAULT_PAGE_SIZE}
        additionalQueries={JSON.stringify(selectedFilters)}
        inputPlaceHolder={HOSTS_SEARCH_BOX_PLACEHOLDER}
        actionButton={{
          name: "edit columns",
          buttonText: "Edit columns",
          iconSvg: "columns",
          variant: "text-icon",
          onActionButtonClick: toggleEditColumnsModal,
        }}
        primarySelectAction={{
          name: "delete host",
          buttonText: "Delete",
          iconSvg: "trash",
          variant: "text-icon",
          onActionButtonClick: onDeleteHostsClick,
        }}
        secondarySelectActions={secondarySelectActions}
        showMarkAllPages={!unsupportedFilter} // Shortterm fix for #17257
        isAllPagesSelected={isAllMatchingHostsSelected}
        searchable
        renderCount={renderHostCount}
        searchToolTipText={HOSTS_SEARCH_BOX_TOOLTIP}
        emptyComponent={() =>
          EmptyTable({
            header: emptyState().header,
            info: emptyState().info,
          })
        }
        customControl={renderCustomControls}
        onQueryChange={onTableQueryChange}
        toggleAllPagesSelected={toggleAllMatchingHosts}
        resetPageIndex={resetPageIndex}
        disableNextPage={isLastPage}
      />
    );
  };

  const renderNoEnrollSecretBanner = () => {
    const noTeamEnrollSecrets =
      isAnyTeamSelected && !isTeamSecretsLoading && !teamSecrets?.length;
    const noGlobalEnrollSecrets =
      (!isPremiumTier ||
        (isPremiumTier && !isAnyTeamSelected && !isLoadingTeams)) &&
      !isGlobalSecretsLoading &&
      !globalSecrets?.length;

    return (
      ((canEnrollHosts && noTeamEnrollSecrets) ||
        (canEnrollGlobalHosts && noGlobalEnrollSecrets)) &&
      showNoEnrollSecretBanner && (
        <InfoBanner
          className={`${baseClass}__no-enroll-secret-banner`}
          pageLevel
          closable
          color="grey"
        >
          <div>
            <span>
              You have no enroll secrets. Manage enroll secrets to enroll hosts
              to <b>{isAnyTeamSelected ? currentTeamName : "Fleet"}</b>.
            </span>
          </div>
        </InfoBanner>
      )
    );
  };

  const showAddHostsButton =
    canEnrollHosts &&
    !hasErrors &&
    (!maybeEmptyHosts || includesFilterQueryParam);

  return (
    <>
      <MainContent>
        <div className={`${baseClass}`}>
          <div className="header-wrap">
            {renderHeader()}
            <div className={`${baseClass} button-wrap`}>
              {!isSandboxMode && canEnrollHosts && !hasErrors && (
                <Button
                  onClick={() => setShowEnrollSecretModal(true)}
                  className={`${baseClass}__enroll-hosts button`}
                  variant="inverse"
                >
                  <span>Manage enroll secret</span>
                </Button>
              )}
              {showAddHostsButton && (
                <Button
                  onClick={toggleAddHostsModal}
                  className={`${baseClass}__add-hosts`}
                  variant="brand"
                >
                  <span>Add hosts</span>
                </Button>
              )}
            </div>
          </div>
          {/* TODO: look at improving the props API for this component. Im thinking
          some of the props can be defined inside HostsFilterBlock */}
          <HostsFilterBlock
            params={{
              policyResponse,
              policyId,
              policy,
              macSettingsStatus,
              softwareId,
              softwareTitleId,
              softwareVersionId,
              softwareStatus,
              mdmId,
              mdmEnrollmentStatus,
              lowDiskSpaceHosts,
              osVersionId,
              osName,
              osVersion,
              osVersions,
              munkiIssueId,
              munkiIssueDetails: hostsData?.munki_issue || null,
              softwareDetails:
                hostsData?.software || hostsData?.software_title || null,
              mdmSolutionDetails:
                hostsData?.mobile_device_management_solution || null,
              osSettingsStatus,
              diskEncryptionStatus,
              bootstrapPackageStatus,
              vulnerability,
            }}
            selectedLabel={selectedLabel}
            isOnlyObserver={isOnlyObserver}
            handleClearRouteParam={handleClearRouteParam}
            handleClearFilter={handleClearFilter}
            onChangePoliciesFilter={handleChangePoliciesFilter}
            onChangeOsSettingsFilter={handleChangeOsSettingsFilter}
            onChangeDiskEncryptionStatusFilter={
              handleChangeDiskEncryptionStatusFilter
            }
            onChangeBootstrapPackageStatusFilter={
              handleChangeBootstrapPackageStatusFilter
            }
            onChangeMacSettingsFilter={handleMacSettingsStatusDropdownChange}
            onChangeSoftwareInstallStatusFilter={
              handleSoftwareInstallStatusChange
            }
            onClickEditLabel={onEditLabelClick}
            onClickDeleteLabel={toggleDeleteLabelModal}
          />
          {renderNoEnrollSecretBanner()}
          {renderTable()}
        </div>
      </MainContent>
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
