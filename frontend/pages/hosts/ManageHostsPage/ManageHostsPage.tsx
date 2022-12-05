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
import { IMdmSolution, IMunkiIssuesAggregate } from "interfaces/macadmins";
import {
  formatOperatingSystemDisplayName,
  IOperatingSystemVersion,
} from "interfaces/operating_system";
import { IPolicy, IStoredPolicyResponse } from "interfaces/policy";
import { ISoftware } from "interfaces/software";
import { ITeam } from "interfaces/team";
import sortUtils from "utilities/sort";
import {
  HOSTS_SEARCH_BOX_PLACEHOLDER,
  HOSTS_SEARCH_BOX_TOOLTIP,
  PLATFORM_LABEL_DISPLAY_NAMES,
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

import { getValidatedTeamId } from "utilities/helpers";
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

  const {
    isLoading: isLoadingLabels,
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

  useQuery<IStoredPolicyResponse, Error>(
    ["policy"],
    () => globalPoliciesAPI.load(policyId),
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
        munki_issue,
      } = await hostsAPI.loadHosts(options);
      setHosts(returnedHosts);
      software && setSoftwareDetails(software);
      mobile_device_management_solution &&
        setMDMSolutionDetails(mobile_device_management_solution);
      munki_issue && setMunkiIssueDetails(munki_issue);
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

    const validLabel = find(labels, ["slug", slugToFind]) as ILabel;

    setSelectedLabel(validLabel);

    const options: ILoadHostsOptions = {
      selectedLabels: selectedFilters,
      globalFilter: searchQuery,
      sortBy,
      teamId: selectedTeam?.id,
      policyId,
      policyResponse,
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

    setFilteredHostsPath(location.pathname + location.search);
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

  const handleClearPoliciesFilter = () => {
    handleClearFilter(["policy_id", "policy_response"]);
  };

  const handleClearOSFilter = () => {
    handleClearFilter(["os_id", "os_name", "os_version"]);
  };

  const handleClearSoftwareFilter = () => {
    handleClearFilter(["software_id"]);
  };

  const handleClearMDMSolutionFilter = () => {
    handleClearFilter(["mdm_id"]);
  };

  const handleClearMDMEnrollmentFilter = () => {
    handleClearFilter(["mdm_enrollment_status"]);
  };

  const handleClearMunkiIssueFilter = () => {
    handleClearFilter(["munki_issue_id"]);
  };

  const handleClearLowDiskSpaceFilter = () => {
    handleClearFilter(["low_disk_space"]);
  };

  const handleTeamSelect = (teamId: number) => {
    const { MANAGE_HOSTS } = PATHS;

    const teamIdParam = getValidatedTeamId(
      availableTeams || [],
      teamId,
      currentUser,
      isOnGlobalTeam ?? false
    );

    const slimmerParams = omit(queryParams, ["team_id"]);

    const newQueryParams = !teamIdParam
      ? slimmerParams
      : Object.assign(slimmerParams, { team_id: teamIdParam });

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
            onClear={handleClearRouteParam}
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
    if (!osId && !(osName && osVersion)) return null;

    let os: IOperatingSystemVersion | undefined;
    if (osId) {
      os = osVersions?.find((v) => v.os_id === osId);
    } else if (osName && osVersion) {
      const name: string = osName;
      const vers: string = osVersion;

      os = osVersions?.find(
        ({ name_only, version }) =>
          name_only.toLowerCase() === name.toLowerCase() &&
          version.toLowerCase() === vers.toLowerCase()
      );
    }
    if (!os) return null;

    const { name, name_only, version } = os;
    const label = formatOperatingSystemDisplayName(
      name_only || version
        ? `${name_only || ""} ${version || ""}`
        : `${name || ""}`
    );
    const TooltipDescription = (
      <span className={`tooltip__tooltip-text`}>
        {`Hosts with ${formatOperatingSystemDisplayName(name_only || name)}`},
        <br />
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
        label={policy?.name ?? "..."}
        onClear={handleClearPoliciesFilter}
        className={`${baseClass}__policies-filter-pill`}
      />
    </>
  );

  const renderSoftwareFilterBlock = () => {
    if (!softwareDetails) return null;

    const { name, version } = softwareDetails;
    const label = `${name || "Unknown software"} ${version || ""}`;

    const TooltipDescription = (
      <span className={`tooltip__tooltip-text`}>
        Hosts with {name || "Unknown software"},
        <br />
        {version || "version unknown"} installed
      </span>
    );

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

  const renderMunkiIssueFilterBlock = () => {
    if (munkiIssueDetails) {
      return (
        <FilterPill
          label={munkiIssueDetails.name}
          tooltipDescription={
            <span className={`tooltip__tooltip-text`}>
              Hosts that reported this Munki issue <br />
              the last time Munki ran on each host.
            </span>
          }
          onClear={handleClearMunkiIssueFilter}
        />
      );
    }
    return null;
  };

  const renderLowDiskSpaceFilterBlock = () => {
    const TooltipDescription = (
      <span className={`tooltip__tooltip-text`}>
        Hosts that have {lowDiskSpaceHosts} GB or less <br />
        disk space available.
      </span>
    );

    return (
      <FilterPill
        label="Low disk space"
        tooltipDescription={TooltipDescription}
        onClear={handleClearLowDiskSpaceFilter}
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
        isUpdatingHosts={isUpdatingHosts}
      />
    );
  };

  const renderDeleteHostModal = () => (
    <DeleteHostModal
      selectedHostIds={selectedHostIds}
      onSubmit={onDeleteHostSubmit}
      onCancel={toggleDeleteHostModal}
      isAllMatchingHostsSelected={isAllMatchingHostsSelected}
      isUpdatingHosts={isUpdatingHosts}
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

  const renderActiveFilterBlock = () => {
    const showSelectedLabel = selectedLabel && selectedLabel.type !== "all";

    if (
      showSelectedLabel ||
      policyId ||
      softwareId ||
      showSelectedLabel ||
      mdmId ||
      mdmEnrollmentStatus ||
      lowDiskSpaceHosts ||
      osId ||
      (osName && osVersion) ||
      munkiIssueId
    ) {
      const renderFilterPill = () => {
        switch (true) {
          // backend allows for pill combos label x low disk space
          case showSelectedLabel && !!lowDiskSpaceHosts:
            return (
              <>
                {renderLabelFilterPill()} {renderLowDiskSpaceFilterBlock()}
              </>
            );
          case showSelectedLabel:
            return renderLabelFilterPill();
          case !!policyId:
            return renderPoliciesFilterBlock();
          case !!softwareId:
            return renderSoftwareFilterBlock();
          case !!mdmId:
            return renderMDMSolutionFilterBlock();
          case !!mdmEnrollmentStatus:
            return renderMDMEnrollmentFilterBlock();
          case !!osId || (!!osName && !!osVersion):
            return renderOSFilterBlock();
          case !!munkiIssueId:
            return renderMunkiIssueFilterBlock();
          case !!lowDiskSpaceHosts:
            return renderLowDiskSpaceFilterBlock();
          default:
            return null;
        }
      };

      return (
        <div className={`${baseClass}__labels-active-filter-wrap`}>
          {renderFilterPill()}
        </div>
      );
    }
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
    if (!config || !currentUser || !hosts || !teamSync) {
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
        pageSize={100}
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
          {renderActiveFilterBlock()}
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
