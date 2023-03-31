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
import hostsAPI, {
  ILoadHostsOptions,
  ISortOption,
  MacSettingsStatusQueryParam,
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
import { TableContext } from "context/table";
import { NotificationContext } from "context/notification";
import {
  IEnrollSecret,
  IEnrollSecretsResponse,
} from "interfaces/enroll_secret";
import { IHost } from "interfaces/host";
import { ILabel } from "interfaces/label";
import { IMunkiIssuesAggregate } from "interfaces/macadmins";
import { IMdmSolution } from "interfaces/mdm";
import { IOperatingSystemVersion } from "interfaces/operating_system";
import { IPolicy, IStoredPolicyResponse } from "interfaces/policy";
import { ISoftware } from "interfaces/software";
import { ITeam } from "interfaces/team";
import { IEmptyTableProps } from "interfaces/empty_table";

import sortUtils from "utilities/sort";
import {
  HOSTS_SEARCH_BOX_PLACEHOLDER,
  HOSTS_SEARCH_BOX_TOOLTIP,
  PolicyResponse,
} from "utilities/constants";

import Button from "components/buttons/Button";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import TableContainer from "components/TableContainer";
import TableDataError from "components/DataError";
import { IActionButtonProps } from "components/TableContainer/DataTable/ActionButton";
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
  HOST_SELECT_STATUSES,
} from "./constants";
import { isAcceptableStatus, getNextLocationPath } from "./helpers";
import DeleteSecretModal from "../../../components/EnrollSecrets/DeleteSecretModal";
import SecretEditorModal from "../../../components/EnrollSecrets/SecretEditorModal";
import AddHostsModal from "../../../components/AddHostsModal";
import EnrollSecretModal from "../../../components/EnrollSecrets/EnrollSecretModal";
// @ts-ignore
import EditColumnsModal from "./components/EditColumnsModal/EditColumnsModal";
import TransferHostModal from "../components/TransferHostModal";
import DeleteHostModal from "../components/DeleteHostModal";
import DeleteLabelModal from "./components/DeleteLabelModal";
import EditColumnsIcon from "../../../../assets/images/icon-edit-columns-16x16@2x.png";
import CloseIconBlack from "../../../../assets/images/icon-close-fleet-black-16x16@2x.png";
import DownloadIcon from "../../../../assets/images/icon-download-12x12@2x.png";
import LabelFilterSelect from "./components/LabelFilterSelect";
import HostsFilterBlock from "./components/HostsFilterBlock";

interface IManageHostsProps {
  route: RouteProps;
  router: InjectedRouter;
  params: Params;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  location: any; // no type in react-router v3
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
    setAvailableTeams,
    setCurrentTeam,
    setFilteredHostsPath,
  } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  if (queryParams.team_id) {
    const teamIdParam = parseInt(queryParams.team_id, 10);
    if (
      isNaN(teamIdParam) ||
      (teamIdParam &&
        availableTeams &&
        !availableTeams.find(
          (availableTeam) => availableTeam.id === teamIdParam
        ))
    ) {
      router.replace({
        pathname: location.pathname,
        query: omit(queryParams, "team_id"),
      });
    }
  }
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
    storedHiddenColumns || defaultHiddenColumns
  );
  const [selectedHostIds, setSelectedHostIds] = useState<number[]>([]);
  const [isAllMatchingHostsSelected, setIsAllMatchingHostsSelected] = useState(
    false
  );
  const [searchQuery, setSearchQuery] = useState(initialQuery);
  const [hosts, setHosts] = useState<IHost[]>();
  const [isHostsLoading, setIsHostsLoading] = useState(false);
  const [hasHostErrors, setHasHostErrors] = useState(false);
  const [filteredHostCount, setFilteredHostCount] = useState<number>();
  const [isHostCountLoading, setIsHostCountLoading] = useState(false);
  const [hasHostCountErrors, setHasHostCountErrors] = useState(false);
  const [sortBy, setSortBy] = useState<ISortOption[]>(initialSortBy);
  const [policy, setPolicy] = useState<IPolicy>();
  const [softwareDetails, setSoftwareDetails] = useState<ISoftware | null>(
    null
  );
  const [
    mdmSolutionDetails,
    setMDMSolutionDetails,
  ] = useState<IMdmSolution | null>(null);
  const [
    munkiIssueDetails,
    setMunkiIssueDetails,
  ] = useState<IMunkiIssuesAggregate | null>(null);
  const [tableQueryData, setTableQueryData] = useState<ITableQueryProps>();
  const [
    currentQueryOptions,
    setCurrentQueryOptions,
  ] = useState<ILoadHostsOptions>();
  const [resetPageIndex, setResetPageIndex] = useState<boolean>(false);
  const [isUpdatingLabel, setIsUpdatingLabel] = useState<boolean>(false);
  const [isUpdatingSecret, setIsUpdatingSecret] = useState<boolean>(false);
  const [isUpdatingHosts, setIsUpdatingHosts] = useState<boolean>(false);

  // ======== end states
  const routeTemplate = route?.path ?? "";
  const policyId = queryParams?.policy_id;
  const policyResponse: PolicyResponse = queryParams?.policy_response;
  const macSettingsStatus = queryParams?.macos_settings;
  const softwareId =
    queryParams?.software_id !== undefined
      ? parseInt(queryParams.software_id, 10)
      : undefined;
  const status = isAcceptableStatus(queryParams?.status)
    ? queryParams?.status
    : undefined;
  const mdmId =
    queryParams?.mdm_id !== undefined
      ? parseInt(queryParams.mdm_id, 10)
      : undefined;
  const mdmEnrollmentStatus = queryParams?.mdm_enrollment_status;
  const { os_id: osId, os_name: osName, os_version: osVersion } = queryParams;
  const munkiIssueId =
    queryParams?.munki_issue_id !== undefined
      ? parseInt(queryParams.munki_issue_id, 10)
      : undefined;
  const lowDiskSpaceHosts =
    queryParams?.low_disk_space !== undefined
      ? parseInt(queryParams.low_disk_space, 10)
      : undefined;
  const missingHosts = queryParams?.status === "missing";
  const { active_label: activeLabel, label_id: labelID } = routeParams;

  // ===== filter matching
  const selectedFilters: string[] = [];
  labelID && selectedFilters.push(`${LABEL_SLUG_PREFIX}${labelID}`);
  activeLabel && selectedFilters.push(activeLabel);
  // ===== end filter matching

  const canEnrollHosts =
    isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer;
  const canEnrollGlobalHosts = isGlobalAdmin || isGlobalMaintainer;
  const canAddNewLabels = (isGlobalAdmin || isGlobalMaintainer) ?? false;

  const { data: labels, refetch: refetchLabels } = useQuery<
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
        setAvailableTeams(responseTeams);
        if (
          responseTeams.filter(
            (responseTeam) => responseTeam.id === currentTeam?.id
          )
        ) {
          setCurrentTeam(undefined);
        }
        if (!currentTeam && !isOnGlobalTeam && responseTeams.length) {
          setCurrentTeam(responseTeams[0]);
        }
      },
    }
  );

  const { isLoading: isLoadingPolicy } = useQuery<IStoredPolicyResponse, Error>(
    ["policy"],
    () => globalPoliciesAPI.load(policyId),
    {
      enabled: !!policyId,
      onSuccess: ({ policy: policyAPIResponse }) => {
        setPolicy(policyAPIResponse);
      },
      onError: () => {
        setHasHostErrors(true);
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

  const retrieveHosts = async (options: ILoadHostsOptions = {}) => {
    setIsHostsLoading(true);
    options = {
      ...options,
      teamId: queryParams.team_id ? queryParams.team_id : currentTeam?.id,
    };

    try {
      const {
        hosts: returnedHosts,
        software,
        mobile_device_management_solution: mdmSolution,
        munki_issue: munkiIssue,
      } = await hostsAPI.loadHosts(options);
      setHosts(returnedHosts);
      software && setSoftwareDetails(software);
      mdmSolution && setMDMSolutionDetails(mdmSolution);
      munkiIssue && setMunkiIssueDetails(munkiIssue);
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
      teamId: currentTeam?.id,
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

    const validLabel = find(labels, ["slug", slugToFind]) as ILabel;

    setSelectedLabel(validLabel);

    const options: ILoadHostsOptions = {
      selectedLabels: selectedFilters,
      globalFilter: searchQuery,
      sortBy,
      teamId: selectedTeam?.id,
      policyId,
      policyResponse,
      macSettingsStatus,
      softwareId,
      status,
      mdmId,
      mdmEnrollmentStatus,
      munkiIssueId,
      lowDiskSpaceHosts,
      osId,
      osName,
      osVersion,
      page: tableQueryData ? tableQueryData.pageIndex : 0,
      perPage: tableQueryData ? tableQueryData.pageSize : 50,
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
    if (!location.search.includes("software_id")) {
      setFilteredHostsPath(location.pathname + location.search);
    }
  }, [availableTeams, currentTeam, location, labels]);

  const isLastPage =
    tableQueryData &&
    !!filteredHostCount &&
    DEFAULT_PAGE_SIZE * tableQueryData.pageIndex + (hosts?.length || 0) >=
      filteredHostCount;

  const handleLabelChange = ({ slug }: ILabel): boolean => {
    const { MANAGE_HOSTS } = PATHS;

    // Non-status labels are not compatible with policies or software filters
    // so omit policies and software params from next location
    let newQueryParams = queryParams;
    if (slug) {
      newQueryParams = omit(newQueryParams, [
        "policy_id",
        "policy_response",
        "software_id",
      ]);
    }

    router.replace(
      getNextLocationPath({
        pathPrefix: `${MANAGE_HOSTS}/${slug}`,
        queryParams: newQueryParams,
      })
    );

    return true;
  };

  // NOTE: used to reset page number to 0 when modifying filters
  const handleResetPageIndex = () => {
    setTableQueryData({
      ...tableQueryData,
      pageIndex: 0,
    } as ITableQueryProps);

    setResetPageIndex(true);
  };

  const handleChangePoliciesFilter = (response: PolicyResponse) => {
    handleResetPageIndex();

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

  const handleClearRouteParam = () => {
    handleResetPageIndex();

    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.MANAGE_HOSTS,
        routeTemplate,
        routeParams: undefined,
        queryParams,
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
        queryParams: omit(queryParams, omitParams),
      })
    );
  };

  const handleTeamSelect = (teamId: number) => {
    const { MANAGE_HOSTS } = PATHS;

    const slimmerParams = omit(queryParams, ["team_id"]);

    const newQueryParams = !teamId
      ? slimmerParams
      : Object.assign(slimmerParams, { team_id: teamId });

    const nextLocation = getNextLocationPath({
      pathPrefix: MANAGE_HOSTS,
      routeTemplate,
      routeParams,
      queryParams: newQueryParams,
    });

    handleResetPageIndex();
    router.replace(nextLocation);
    const selectedTeam = find(availableTeams, ["id", teamId]);
    setCurrentTeam(selectedTeam);
  };

  const handleStatusDropdownChange = (statusName: string) => {
    handleResetPageIndex();

    router.replace(
      getNextLocationPath({
        pathPrefix: PATHS.MANAGE_HOSTS,
        routeTemplate,
        routeParams,
        queryParams: { ...queryParams, status: statusName },
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
        queryParams: { ...queryParams, macos_settings: newMacSettingsStatus },
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

  const onSaveColumns = (newHiddenColumns: string[]) => {
    localStorage.setItem("hostHiddenColumns", JSON.stringify(newHiddenColumns));
    setHiddenColumns(newHiddenColumns);
    setShowEditColumnsModal(false);
  };

  // NOTE: used to reset page number to 0 when modifying filters
  useEffect(() => {
    setResetPageIndex(false);
    if (queryParams.add_hosts === "true") {
      setShowAddHostsModal(true);
    }
  }, [queryParams]);

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

      if (currentTeam) {
        newQueryParams.team_id = currentTeam.id;
      }

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
      } else if (osId || (osName && osVersion)) {
        newQueryParams.os_id = osId;
        newQueryParams.os_name = osName;
        newQueryParams.os_version = osVersion;
      }

      router.replace(
        getNextLocationPath({
          pathPrefix: PATHS.MANAGE_HOSTS,
          routeTemplate,
          routeParams,
          queryParams: newQueryParams,
        })
      );

      return 0;
    },
    [
      availableTeams,
      currentTeam,
      currentUser,
      policyId,
      queryParams,
      macSettingsStatus,
      softwareId,
      status,
      mdmId,
      mdmEnrollmentStatus,
      munkiIssueId,
      lowDiskSpaceHosts,
      osId,
      osName,
      osVersion,
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

    setIsUpdatingSecret(true);

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
    } finally {
      setIsUpdatingSecret(false);
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

    setIsUpdatingSecret(true);

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
      console.error(error);
      renderFlash("error", "Could not delete label. Please try again.");
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

  const onTransferHostSubmit = async (transferTeam: ITeam) => {
    setIsUpdatingHosts(true);

    const teamId = typeof transferTeam.id === "number" ? transferTeam.id : null;
    let action = hostsAPI.transferToTeam(teamId, selectedHostIds);

    if (isAllMatchingHostsSelected) {
      const labelId = selectedLabel?.id;

      action = hostsAPI.transferToTeamByFilter({
        teamId,
        query: searchQuery,
        status,
        labelId,
      });
    }

    try {
      await action;

      const successMessage =
        teamId === null
          ? `Hosts successfully removed from teams.`
          : `Hosts successfully transferred to  ${transferTeam.name}.`;

      renderFlash("success", successMessage);
      setResetSelectedRows(true);
      refetchHosts({
        selectedLabels: selectedFilters,
        globalFilter: searchQuery,
        sortBy,
        teamId: currentTeam?.id,
        policyId,
        policyResponse,
        macSettingsStatus,
        softwareId,
        status,
        mdmId,
        mdmEnrollmentStatus,
        munkiIssueId,
        lowDiskSpaceHosts,
        osId,
        osName,
        osVersion,
      });

      toggleTransferHostModal();
      setSelectedHostIds([]);
      setIsAllMatchingHostsSelected(false);
    } catch (error) {
      renderFlash("error", "Could not transfer hosts. Please try again.");
    } finally {
      setIsUpdatingHosts(false);
    }
  };

  const onDeleteHostSubmit = async () => {
    setIsUpdatingHosts(true);

    let action = hostsAPI.destroyBulk(selectedHostIds);

    if (isAllMatchingHostsSelected) {
      const teamId = currentTeam?.id || null;

      const labelId = selectedLabel?.id;

      action = hostsAPI.destroyByFilter({
        teamId,
        query: searchQuery,
        status,
        labelId,
      });
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
        macSettingsStatus,
        softwareId,
        status,
        mdmId,
        mdmEnrollmentStatus,
        munkiIssueId,
        lowDiskSpaceHosts,
        osId,
        osName,
        osVersion,
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
    } finally {
      setIsUpdatingHosts(false);
    }
  };

  const renderTeamsFilterDropdown = () => (
    <TeamsDropdown
      currentUserTeams={availableTeams || []}
      selectedTeamId={currentTeam?.id}
      isDisabled={isHostsLoading || isHostCountLoading}
      onChange={(newSelectedValue: number) =>
        handleTeamSelect(newSelectedValue)
      }
    />
  );

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
      isUpdatingSecret={isUpdatingSecret}
    />
  );

  const renderDeleteSecretModal = () => (
    <DeleteSecretModal
      onDeleteSecret={onDeleteSecret}
      selectedTeam={currentTeam?.id || 0}
      teams={teams || []}
      toggleDeleteSecretModal={toggleDeleteSecretModal}
      isUpdatingSecret={isUpdatingSecret}
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
      isUpdatingLabel={isUpdatingLabel}
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
      isUpdating={isUpdatingHosts}
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
      macSettingsStatus,
      softwareId,
      status,
      mdmId,
      mdmEnrollmentStatus,
      munkiIssueId,
      lowDiskSpaceHosts,
      os_id: osId,
      os_name: osName,
      os_version: osVersion,
      visibleColumns,
    };

    options = {
      ...options,
      teamId: currentTeam?.id,
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
        {!!count && (
          <Button
            className={`${baseClass}__export-btn`}
            onClick={onExportHostsResults}
            variant="text-link"
          >
            <>
              Export hosts <img alt="" src={DownloadIcon} />
            </>
          </Button>
        )}
      </div>
    );
  }, [isHostCountLoading, filteredHostCount]);

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
    if (!config || !currentUser || !teamSync) {
      return <Spinner />;
    }

    if (hasHostErrors || hasHostCountErrors) {
      return <TableDataError />;
    }

    // There are no hosts for this instance yet
    if (
      filteredHostCount === 0 &&
      searchQuery === "" &&
      teamSync &&
      !labelID &&
      !status
    ) {
      const {
        software_id,
        policy_id,
        mdm_id,
        mdm_enrollment_status,
        low_disk_space,
      } = queryParams || {};
      const includesNameCardFilter = !!(
        software_id ||
        policy_id ||
        mdm_id ||
        mdm_enrollment_status ||
        low_disk_space ||
        osId ||
        osName ||
        osVersion
      );

      const emptyState = () => {
        const emptyHosts: IEmptyTableProps = {
          iconName: "empty-hosts",
          header: "Devices will show up here once they’re added to Fleet.",
          info:
            "Expecting to see devices? Try again in a few seconds as the system catches up.",
        };
        if (includesNameCardFilter) {
          delete emptyHosts.iconName;
          emptyHosts.header = "No hosts match the current criteria";
          emptyHosts.info =
            "Expecting to see new hosts? Try again in a few seconds as the system catches up.";
        }
        if (canEnrollHosts) {
          emptyHosts.header = "Add your devices to Fleet";
          emptyHosts.info = "Generate an installer to add your own devices.";
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
            iconName: emptyState().iconName,
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

    return (
      <TableContainer
        columns={tableColumns}
        data={hosts || []}
        isLoading={isHostsLoading || isHostCountLoading || isLoadingPolicy}
        manualSortBy
        defaultSortHeader={(sortBy[0] && sortBy[0].key) || DEFAULT_SORT_HEADER}
        defaultSortDirection={
          (sortBy[0] && sortBy[0].direction) || DEFAULT_SORT_DIRECTION
        }
        defaultSearchQuery={searchQuery}
        pageSize={50}
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
        emptyComponent={() =>
          EmptyTable({
            header: emptyState().header,
            info: emptyState().info,
          })
        }
        customControl={renderCustomControls}
        onActionButtonClick={toggleEditColumnsModal}
        onPrimarySelectActionClick={onDeleteHostsClick}
        onQueryChange={onTableQueryChange}
        toggleAllPagesSelected={toggleAllMatchingHosts}
        resetPageIndex={resetPageIndex}
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
            <Button
              variant="unstyled"
              onClick={() =>
                setShowNoEnrollSecretBanner(!showNoEnrollSecretBanner)
              }
            >
              <img alt="Dismiss no enroll secret banner" src={CloseIconBlack} />
            </Button>
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
                  !status &&
                  filteredHostCount === 0 &&
                  searchQuery === "" &&
                  teamSync &&
                  !labelID
                ) && (
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
              mdmId,
              mdmEnrollmentStatus,
              lowDiskSpaceHosts,
              osId,
              osName,
              osVersion,
              osVersions,
              munkiIssueId,
              munkiIssueDetails,
              softwareDetails,
              mdmSolutionDetails,
            }}
            selectedLabel={selectedLabel}
            isOnlyObserver={isOnlyObserver}
            handleClearRouteParam={handleClearRouteParam}
            handleClearFilter={handleClearFilter}
            onChangePoliciesFilter={handleChangePoliciesFilter}
            onChangeMacSettingsFilter={handleMacSettingsStatusDropdownChange}
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
