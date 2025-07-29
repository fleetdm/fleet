import React, {
  useCallback,
  useContext,
  useMemo,
  useRef,
  useState,
  useEffect,
} from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";
import { AxiosError } from "axios";

import hostAPI, {
  IGetHostSoftwareResponse,
  IHostSoftwareQueryKey,
} from "services/entities/hosts";
import PATHS from "router/paths";
import {
  IHostSoftware,
  IVPPHostSoftware,
  ISoftware,
  IHostSoftwareWithUiStatus,
} from "interfaces/software";
import { HostPlatform, isIPadOrIPhone, isAndroid } from "interfaces/platform";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { getPathWithQueryParams } from "utilities/url";

import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";

import CardHeader from "components/CardHeader";
import DataError from "components/DataError";
import Spinner from "components/Spinner";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import { ISoftwareUninstallDetails } from "components/ActivityDetails/InstallDetails/SoftwareUninstallDetailsModal/SoftwareUninstallDetailsModal";
import SoftwareInstallDetailsModal from "components/ActivityDetails/InstallDetails/SoftwareInstallDetailsModal";
import AppInstallDetailsModal from "components/ActivityDetails/InstallDetails/AppInstallDetails";

import { generateHostSWLibraryTableHeaders } from "./HostSoftwareLibraryTable/HostSoftwareLibraryTableConfig";
import HostSoftwareLibraryTable from "./HostSoftwareLibraryTable";
import { getInstallErrorMessage, getUninstallErrorMessage } from "./helpers";
import { getUiStatus } from "../Software/helpers";
import SoftwareUpdateModal from "../Software/SoftwareUpdateModal";

const baseClass = "host-software-library-card";

export interface ITableSoftware extends Omit<ISoftware, "vulnerabilities"> {
  vulnerabilities: string[]; // for client-side search purposes, we only want an array of cve strings
}

interface IHostSoftwareLibraryProps {
  /** This is the host id or the device token */
  id: number | string;
  platform: HostPlatform;
  hostDisplayName: string;
  softwareUpdatedAt?: string;
  router: InjectedRouter;
  queryParams: ReturnType<typeof parseHostSoftwareLibraryQueryParams>;
  pathname: string;
  hostTeamId: number;
  hostName: string;
  onShowInventoryVersions: (software?: IHostSoftware) => void;
  onShowUninstallDetails: (details?: ISoftwareUninstallDetails) => void;
  isSoftwareEnabled?: boolean;
  hostScriptsEnabled?: boolean;
  hostMDMEnrolled?: boolean;
  isHostOnline?: boolean;
  refetchHostDetails: () => void;
  isHostDetailsPolling: boolean;
}

const DEFAULT_SEARCH_QUERY = "";
const DEFAULT_SORT_DIRECTION = "asc";
const DEFAULT_SORT_HEADER = "name";
const DEFAULT_PAGE = 0;
const DEFAULT_PAGE_SIZE = 20;

export const parseHostSoftwareLibraryQueryParams = (queryParams: {
  page?: string;
  query?: string;
  order_key?: string;
  order_direction?: "asc" | "desc";
  self_service?: string;
}) => {
  const searchQuery = queryParams?.query ?? DEFAULT_SEARCH_QUERY;
  const sortHeader = queryParams?.order_key ?? DEFAULT_SORT_HEADER;
  const sortDirection = queryParams?.order_direction ?? DEFAULT_SORT_DIRECTION;
  const page = queryParams?.page
    ? parseInt(queryParams.page, 10)
    : DEFAULT_PAGE;
  const pageSize = DEFAULT_PAGE_SIZE;
  const selfService = queryParams?.self_service === "true";

  return {
    page,
    query: searchQuery,
    order_key: sortHeader,
    order_direction: sortDirection,
    per_page: pageSize,
    available_for_install: true, // always true for host installers
    self_service: selfService,
  };
};

const HostSoftwareLibrary = ({
  id,
  platform,
  hostDisplayName,
  softwareUpdatedAt,
  hostScriptsEnabled,
  router,
  queryParams,
  pathname,
  hostTeamId = 0,
  hostName,
  onShowInventoryVersions,
  onShowUninstallDetails,
  isSoftwareEnabled = false,
  hostMDMEnrolled,
  isHostOnline = false,
  refetchHostDetails,
  isHostDetailsPolling,
}: IHostSoftwareLibraryProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const {
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamAdmin,
    isTeamMaintainer,
  } = useContext(AppContext);

  const isUnsupported = isAndroid(platform); // no Android software
  const isWindowsHost = platform === "windows";
  const isIPadOrIPhoneHost = isIPadOrIPhone(platform);
  const isMacOSHost = platform === "darwin";

  const [hostSoftwareLibraryRes, setHostSoftwareLibraryRes] = useState<
    IGetHostSoftwareResponse | undefined
  >(undefined);
  const [
    selectedSoftwareUpdates,
    setSelectedSoftwareUpdates,
  ] = useState<IHostSoftware | null>(null);
  // these states and modal logic exist at this level intead of the page level to match the similar
  // pattern on
  // the device user page, which needs to be at this level to manipulate relevant UI states e.g.
  // "updating..." when the user clicks "Retry" in the SoftwareInstallDetailsModal
  const [
    selectedHostSWInstallDetails,
    setSelectedHostSWInstallDetails,
  ] = useState<IHostSoftware | null>(null);
  const [
    selectedVPPInstallDetails,
    setSelectedVPPInstallDetails,
  ] = useState<IVPPHostSoftware | null>(null);

  const enhancedSoftware = useMemo(() => {
    if (!hostSoftwareLibraryRes) return [];
    return hostSoftwareLibraryRes.software.map((software) => ({
      ...software,
      ui_status: getUiStatus(software, isHostOnline),
    }));
  }, [hostSoftwareLibraryRes, isHostOnline]);

  const pendingSoftwareSetRef = useRef<Set<string>>(new Set()); // Track for polling
  const pollingTimeoutIdRef = useRef<NodeJS.Timeout | null>(null);
  const isAwaitingHostDetailsPolling = useRef(isHostDetailsPolling);

  const queryKey = useMemo<IHostSoftwareQueryKey[]>(() => {
    return [
      {
        scope: "host_software",
        id: id as number,
        softwareUpdatedAt,
        ...queryParams,
      },
    ];
  }, [queryParams, id, softwareUpdatedAt]);

  const {
    isLoading: hostSoftwareLibraryLoading,
    isError: hostSoftwareLibraryError,
    isFetching: hostSoftwareLibraryFetching,
    refetch: refetchHostSoftwareLibrary,
  } = useQuery<
    IGetHostSoftwareResponse,
    AxiosError,
    IGetHostSoftwareResponse,
    IHostSoftwareQueryKey[]
  >(queryKey, () => hostAPI.getHostSoftware(queryKey[0]), {
    ...DEFAULT_USE_QUERY_OPTIONS,
    enabled: isSoftwareEnabled && !isUnsupported,
    keepPreviousData: true,
    onSuccess: (response) => {
      setHostSoftwareLibraryRes(response);
    },
  });

  // After host details polling (in parent) finishes, refetch software data.
  // Ensures self service data reflects updates to installed_versions from the latest host details.
  useEffect(() => {
    // Detect completion of the host details polling (in parent)
    // Once host details polling completes, refetch software data to retreive updated installed_versions keyed from host details data
    if (isAwaitingHostDetailsPolling.current && !isHostDetailsPolling) {
      refetchHostSoftwareLibrary();
    }
    isAwaitingHostDetailsPolling.current = isHostDetailsPolling;
  }, [isHostDetailsPolling, refetchHostSoftwareLibrary]);

  // Poll for pending installs/uninstalls
  const { refetch: refetchForPendingInstallsOrUninstalls } = useQuery<
    IGetHostSoftwareResponse,
    AxiosError
  >(
    ["pending_installs", queryKey[0]],
    () => hostAPI.getHostSoftware(queryKey[0]),
    {
      enabled: false,
      onSuccess: (response) => {
        // Get the set of pending software IDs
        const newPendingSet = new Set(
          response.software
            .filter(
              (software) =>
                software.status === "pending_install" ||
                software.status === "pending_uninstall"
            )
            .map((software) => String(software.id))
        );

        // Refresh host details if the number of pending installs or uninstalls has decreased
        // To update the software library information of the newly installed/uninstalled software
        if (newPendingSet.size < pendingSoftwareSetRef.current.size) {
          refetchHostDetails();
        }

        // Compare new set with the previous set
        const setsAreEqual =
          newPendingSet.size === pendingSoftwareSetRef.current.size &&
          [...newPendingSet].every((pendingId) =>
            pendingSoftwareSetRef.current.has(pendingId)
          );

        if (newPendingSet.size > 0) {
          // If the set changed, update and continue polling
          if (!setsAreEqual) {
            pendingSoftwareSetRef.current = newPendingSet;
            setHostSoftwareLibraryRes(response);
          }

          // Continue polling
          if (pollingTimeoutIdRef.current) {
            clearTimeout(pollingTimeoutIdRef.current);
          }
          pollingTimeoutIdRef.current = setTimeout(() => {
            refetchForPendingInstallsOrUninstalls();
          }, 5000);
        } else {
          // No pending installs nor pending uninstalls, stop polling and refresh data
          pendingSoftwareSetRef.current = new Set();
          if (pollingTimeoutIdRef.current) {
            clearTimeout(pollingTimeoutIdRef.current);
            pollingTimeoutIdRef.current = null;
          }
          setHostSoftwareLibraryRes(response);
        }
      },
      onError: () => {
        pendingSoftwareSetRef.current = new Set();
        renderFlash(
          "error",
          "We're having trouble checking pending installs. Please refresh the page."
        );
      },
    }
  );

  // Stop polling if the host goes offline
  // Polling will automatically resume when host is online with pending installs
  useEffect(() => {
    if (!isHostOnline) {
      if (pollingTimeoutIdRef.current) {
        clearTimeout(pollingTimeoutIdRef.current);
        pollingTimeoutIdRef.current = null;
      }
      pendingSoftwareSetRef.current = new Set();
    }
  }, [isHostOnline]);

  const startPollingForPendingInstallsOrUninstalls = useCallback(
    (pendingIds: string[]) => {
      if (isHostOnline) {
        const newSet = new Set(pendingIds);
        const setsAreEqual =
          newSet.size === pendingSoftwareSetRef.current.size &&
          [...newSet].every((pendingId) =>
            pendingSoftwareSetRef.current.has(pendingId)
          );
        if (!setsAreEqual) {
          pendingSoftwareSetRef.current = newSet;

          // Clear any existing timeout to avoid overlap
          if (pollingTimeoutIdRef.current) {
            clearTimeout(pollingTimeoutIdRef.current);
          }
          // Starts polling for pending installs/uninstalls
          refetchForPendingInstallsOrUninstalls();
        }
      }
    },
    [refetchForPendingInstallsOrUninstalls, isHostOnline]
  );

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      pendingSoftwareSetRef.current = new Set();
      if (pollingTimeoutIdRef.current) {
        clearTimeout(pollingTimeoutIdRef.current);
        pollingTimeoutIdRef.current = null;
      }
    };
  }, []);

  // On initial load or data change, check for pending installs/uninstalls
  useEffect(() => {
    const pendingSoftware = hostSoftwareLibraryRes?.software.filter(
      (software) =>
        software.status === "pending_install" ||
        software.status === "pending_uninstall"
    );
    const pendingIds = pendingSoftware?.map((s) => String(s.id)) ?? [];
    if (pendingIds.length > 0) {
      startPollingForPendingInstallsOrUninstalls(pendingIds);
    }
  }, [hostSoftwareLibraryRes, startPollingForPendingInstallsOrUninstalls]);

  const onAddSoftware = useCallback(() => {
    // "Add Software" path dependent on host's platform
    const addSoftwarePathForHostPlatform = () => {
      if (isIPadOrIPhoneHost) {
        return PATHS.SOFTWARE_ADD_APP_STORE;
      }
      if (isMacOSHost || isWindowsHost) {
        return PATHS.SOFTWARE_ADD_FLEET_MAINTAINED;
      }
      return PATHS.SOFTWARE_ADD_PACKAGE;
    };

    router.push(
      getPathWithQueryParams(addSoftwarePathForHostPlatform(), {
        team_id: hostTeamId,
      })
    );
  }, [hostTeamId, isIPadOrIPhoneHost, isMacOSHost, isWindowsHost, router]);

  const onShowUpdateDetails = useCallback(
    (software?: IHostSoftware) => {
      if (software) {
        setSelectedSoftwareUpdates(software);
      }
    },
    [setSelectedSoftwareUpdates]
  );

  const onSetSelectedHostSWInstallDetails = useCallback(
    (hostSW?: IHostSoftware) => {
      if (hostSW) {
        setSelectedHostSWInstallDetails(hostSW);
      }
    },
    [setSelectedHostSWInstallDetails]
  );

  const onSetSelectedVPPInstallDetails = useCallback(
    (s?: IVPPHostSoftware) => {
      if (s) {
        setSelectedVPPInstallDetails(s);
      }
    },
    [setSelectedVPPInstallDetails]
  );

  const onInstallOrUninstall = useCallback(() => {
    // For online hosts, poll for change in pending statuses
    // For offline hosts, refresh the data without polling
    isHostOnline
      ? refetchForPendingInstallsOrUninstalls()
      : refetchHostSoftwareLibrary();
  }, [
    refetchForPendingInstallsOrUninstalls,
    refetchHostSoftwareLibrary,
    isHostOnline,
  ]);

  const userHasSWWritePermission = Boolean(
    isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer
  );

  const isMountedRef = useRef(false);
  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  const onClickInstallAction = useCallback(
    async (softwareId: number) => {
      try {
        await hostAPI.installHostSoftwarePackage(id as number, softwareId);
        if (isMountedRef.current) {
          onInstallOrUninstall();
        }
        renderFlash(
          "success",
          <>
            Software{" "}
            {isHostOnline
              ? "is installing"
              : "will install when the host comes online"}
            . To see details, go to <b>Details &gt; Activity</b>.
          </>
        );
      } catch (e) {
        renderFlash("error", getInstallErrorMessage(e));
      }
    },
    [id, renderFlash, onInstallOrUninstall, isHostOnline]
  );

  const onClickUninstallAction = useCallback(
    async (softwareId: number) => {
      try {
        await hostAPI.uninstallHostSoftwarePackage(id as number, softwareId);
        if (isMountedRef.current) {
          onInstallOrUninstall();
        }
        renderFlash(
          "success",
          <>
            Software{" "}
            {isHostOnline
              ? "is uninstalling"
              : "will uninstall when the host comes online"}
            . To see details, go to <b>Details &gt; Activity</b>.
          </>
        );
      } catch (e) {
        renderFlash("error", getUninstallErrorMessage(e));
      }
    },
    [id, renderFlash, onInstallOrUninstall, isHostOnline]
  );

  const tableConfig = useMemo(() => {
    return generateHostSWLibraryTableHeaders({
      userHasSWWritePermission,
      hostScriptsEnabled,
      hostMDMEnrolled,
      router,
      teamId: hostTeamId,
      hostName,
      baseClass,
      onShowInventoryVersions,
      onShowUpdateDetails,
      onSetSelectedHostSWInstallDetails,
      onSetSelectedVPPInstallDetails,
      onShowUninstallDetails,
      onClickInstallAction,
      onClickUninstallAction,
      isHostOnline,
    });
  }, [
    userHasSWWritePermission,
    hostScriptsEnabled,
    hostMDMEnrolled,
    router,
    hostTeamId,
    hostName,
    onShowInventoryVersions,
    onShowUpdateDetails,
    onSetSelectedHostSWInstallDetails,
    onSetSelectedVPPInstallDetails,
    onShowUninstallDetails,
    onClickInstallAction,
    onClickUninstallAction,
    isHostOnline,
  ]);

  const isLoading = hostSoftwareLibraryLoading;
  const isError = hostSoftwareLibraryError;
  const data = hostSoftwareLibraryRes;
  const enhancedData = enhancedSoftware;

  const renderHostSoftware = () => {
    if (isLoading) {
      return <Spinner />;
    }

    if (isError) {
      return <DataError verticalPaddingSize="pad-xxxlarge" />;
    }

    return (
      <HostSoftwareLibraryTable
        isLoading={hostSoftwareLibraryFetching}
        data={data}
        enhancedData={enhancedData}
        platform={platform}
        router={router}
        tableConfig={tableConfig}
        sortHeader={queryParams.order_key}
        sortDirection={queryParams.order_direction}
        searchQuery={queryParams.query}
        page={queryParams.page}
        pagePath={pathname}
        selfService={queryParams.self_service}
      />
    );
  };

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__header`}>
        <CardHeader subheader="Software available to be installed on this host" />
        {userHasSWWritePermission && (
          <Button variant="text-icon" onClick={onAddSoftware}>
            <Icon name="plus" />
            <span>Add software</span>
          </Button>
        )}
      </div>
      {renderHostSoftware()}
      {selectedSoftwareUpdates && (
        <SoftwareUpdateModal
          hostDisplayName={hostDisplayName}
          software={selectedSoftwareUpdates}
          onUpdate={onClickInstallAction}
          onExit={() => setSelectedSoftwareUpdates(null)}
        />
      )}
      {selectedHostSWInstallDetails && (
        <SoftwareInstallDetailsModal
          details={{
            host_display_name: hostDisplayName,
            install_uuid:
              selectedHostSWInstallDetails.software_package?.last_install
                ?.install_uuid, // slightly redundant, see explanation in `SoftwareInstallDetailsModal
          }}
          hostSoftware={selectedHostSWInstallDetails}
          onCancel={() => setSelectedHostSWInstallDetails(null)}
        />
      )}
      {selectedVPPInstallDetails && (
        <AppInstallDetailsModal
          details={{
            fleetInstallStatus:
              selectedVPPInstallDetails.status || "pending_install", // TODO - okay default?
            hostDisplayName,
            appName: selectedVPPInstallDetails.name,
            commandUuid: selectedVPPInstallDetails.commandUuid,
          }}
          hostSoftware={selectedVPPInstallDetails}
          onCancel={() => setSelectedVPPInstallDetails(null)}
        />
      )}
    </div>
  );
};

export default React.memo(HostSoftwareLibrary);
