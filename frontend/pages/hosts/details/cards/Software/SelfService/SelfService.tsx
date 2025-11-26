import React, {
  useCallback,
  useState,
  useContext,
  useMemo,
  useRef,
  useEffect,
} from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";
import { AxiosError } from "axios";

import { NotificationContext } from "context/notification";
import { INotification } from "interfaces/notification";
import {
  IDeviceSoftware,
  IHostSoftware,
  IDeviceSoftwareWithUiStatus,
  IVPPHostSoftware,
} from "interfaces/software";

import deviceApi, {
  IDeviceSoftwareQueryKey,
  IGetDeviceSoftwareResponse,
} from "services/entities/device_user";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { getExtensionFromFileName } from "utilities/file/fileUtils";

import SoftwareUninstallDetailsModal, {
  ISWUninstallDetailsParentState,
} from "components/ActivityDetails/InstallDetails/SoftwareUninstallDetailsModal/SoftwareUninstallDetailsModal";
import SoftwareInstallDetailsModal from "components/ActivityDetails/InstallDetails/SoftwareInstallDetailsModal";
import SoftwareIpaInstallDetailsModal from "components/ActivityDetails/InstallDetails/SoftwareIpaInstallDetailsModal";
import SoftwareScriptDetailsModal from "components/ActivityDetails/InstallDetails/SoftwareScriptDetailsModal";
import { VppInstallDetailsModal } from "components/ActivityDetails/InstallDetails/VppInstallDetailsModal/VppInstallDetailsModal";

import UpdatesCard from "./components/UpdatesCard/UpdatesCard";
import SelfServiceCard from "./SelfServiceCard/SelfServiceCard";
import SoftwareUpdateModal from "./components/SoftwareUpdateModal";
import UninstallSoftwareModal from "./components/UninstallSoftwareModal";
import SoftwareInstructionsModal from "./components/OpenSoftwareModal";

import { generateSoftwareTableHeaders } from "./components/SelfServiceTable/SelfServiceTableConfig";
import { getLastInstall } from "../../HostSoftwareLibrary/helpers";

import { getUiStatus } from "../helpers";

const baseClass = "software-self-service";

// Kept separately for stable, API-specific filtering (e.g., self_service: true)
// that uses client-only search/filtering after fetching.
const DEFAULT_SELF_SERVICE_CONFIG = {
  // API default params are not subject to change by user
  api: {
    per_page: 9999, // Note: There is no API pagination on this page because of time constraints (e.g. categories and install statuses are not filtered by API)
    order_key: "name",
    order_direction: "asc" as "asc" | "desc",
    self_service: true,
    category_id: undefined,
  },
  // Subject to change by user
  ui: {
    search_query: "",
    page: 0,
    sort_header: "name",
    sort_direction: "asc" as "asc" | "desc",
    page_size: 9999, // 4.77 Design decision to remove UI pagination
  },
};

export const SELF_SERVICE_SUBHEADER =
  "Install organization-approved apps provided by your IT department.";

export interface ISoftwareSelfServiceProps {
  contactUrl: string;
  deviceToken: string;
  isSoftwareEnabled?: boolean;
  pathname: string;
  queryParams: ReturnType<typeof parseSelfServiceQueryParams>;
  router: InjectedRouter;
  refetchHostDetails: () => void;
  isHostDetailsPolling: boolean;
  hostSoftwareUpdatedAt?: string | null;
  hostDisplayName: string;
  isMobileView?: boolean;
}

export const parseSelfServiceQueryParams = (queryParams: {
  page?: string;
  query?: string;
  order_key?: string;
  order_direction?: "asc" | "desc";
  category_id?: string;
}) => {
  const searchQuery =
    queryParams?.query ?? DEFAULT_SELF_SERVICE_CONFIG.ui.search_query;
  const sortHeader =
    queryParams?.order_key ?? DEFAULT_SELF_SERVICE_CONFIG.ui.sort_header;
  const sortDirection =
    queryParams?.order_direction ??
    DEFAULT_SELF_SERVICE_CONFIG.ui.sort_direction;
  const page = queryParams?.page
    ? parseInt(queryParams.page, 10)
    : DEFAULT_SELF_SERVICE_CONFIG.ui.page;
  const pageSize = DEFAULT_SELF_SERVICE_CONFIG.ui.page_size;
  const categoryId = queryParams?.category_id
    ? parseInt(queryParams.category_id, 10)
    : undefined;

  return {
    page,
    query: searchQuery,
    order_key: sortHeader,
    order_direction: sortDirection,
    per_page: pageSize,
    category_id: categoryId,
  };
};

const getInstallerName = (hostSW: IDeviceSoftwareWithUiStatus) => {
  if (hostSW.source === "apps" && hostSW.installed_versions) {
    const filePath = hostSW.installed_versions[0].installed_paths[0];
    // Match the last segment ending in .app and extract the name before .app
    const match = filePath.match(/\/([^/]+)\.app$/);
    return match ? match[1] : hostSW.name;
  }
  return hostSW.name;
};

const SoftwareSelfService = ({
  contactUrl,
  deviceToken,
  isSoftwareEnabled,
  pathname,
  queryParams,
  router,
  refetchHostDetails,
  isHostDetailsPolling,
  hostSoftwareUpdatedAt,
  hostDisplayName,
  isMobileView = false,
}: ISoftwareSelfServiceProps) => {
  const { renderFlash, renderMultiFlash } = useContext(NotificationContext);

  /** Guards against setState/side-effects after unmount */
  const isMountedRef = useRef(false);
  /** Stores software IDs for which the user has initiated an action (install/uninstall) */
  const userActionIdsRef = useRef<Set<number>>(new Set());
  /** Stores timeout handles for each “recently updated” status for proper clear/removal on unmount */
  const recentlyUpdatedTimeouts = useRef<{ [key: number]: NodeJS.Timeout }>({});
  /** Stores the set of pending install/uninstall software IDs for polling */
  const pendingSoftwareIdsRef = useRef<Set<string>>(new Set());
  /** Stores polling timeout for regularly checking API */
  const pollingTimeoutIdRef = useRef<NodeJS.Timeout | null>(null);
  /** Detects parent/host polling completion status to trigger self-update sync */
  const isAwaitingHostDetailsPolling = useRef(isHostDetailsPolling);

  const [selfServiceData, setSelfServiceData] = useState<
    IGetDeviceSoftwareResponse | undefined
  >(undefined);
  const [selectedUpdateDetails, setSelectedUpdateDetails] = useState<
    IDeviceSoftware | undefined
  >(undefined);
  const [
    selectedHostSWInstallDetails,
    setSelectedHostSWInstallDetails,
  ] = useState<IHostSoftware | undefined>(undefined);
  const [
    selectedHostSWIpaInstallDetails,
    setSelectedHostSWIpaInstallDetails,
  ] = useState<IHostSoftware | undefined>(undefined);
  const [
    selectedHostSWScriptDetails,
    setSelectedHostSWScriptDetails,
  ] = useState<IHostSoftware | undefined>(undefined);
  const [
    selectedVPPInstallDetails,
    setSelectedVPPInstallDetails,
  ] = useState<IVPPHostSoftware | null>(null);
  const [
    selectedHostSWUninstallDetails,
    setSelectedHostSWUninstallDetails,
  ] = useState<ISWUninstallDetailsParentState | undefined>(undefined);
  const [showUninstallSoftwareModal, setShowUninstallSoftwareModal] = useState(
    false
  );
  const [showOpenInstructionsModal, setShowOpenInstructionsModal] = useState(
    false
  );
  const [recentlyUpdatedSoftwareIds, setRecentlyUpdatedSoftwareIds] = useState<
    Set<number>
  >(new Set());

  // Cleanup on unmount
  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
      // Clean up timeouts for "recently updated"
      Object.values(recentlyUpdatedTimeouts.current).forEach(clearTimeout);
      recentlyUpdatedTimeouts.current = {};
      // Clean up polling timeout
      if (pollingTimeoutIdRef.current) {
        clearTimeout(pollingTimeoutIdRef.current);
        pollingTimeoutIdRef.current = null;
      }
      // Reset pending IDs
      pendingSoftwareIdsRef.current = new Set();
    };
  }, []);

  /** Registers a software ID as user-initiated action */
  const registerUserSoftwareAction = useCallback((id: number) => {
    userActionIdsRef.current.add(id);
    // Prevent double timeouts
    if (recentlyUpdatedTimeouts.current[id]) {
      clearTimeout(recentlyUpdatedTimeouts.current[id]);
      delete recentlyUpdatedTimeouts.current[id];
    }
    // Schedule removal of "recently updated" after 2 minutes
    recentlyUpdatedTimeouts.current[id] = setTimeout(() => {
      if (isMountedRef.current) {
        setRecentlyUpdatedSoftwareIds((prev) => {
          const next = new Set(prev);
          next.delete(id);
          return next;
        });
      }
      delete recentlyUpdatedTimeouts.current[id];
    }, 120000); // 2 minutes
  }, []);

  const enhancedSoftware: IDeviceSoftwareWithUiStatus[] = useMemo(() => {
    if (!selfServiceData) return [];
    return selfServiceData.software.map((software) => ({
      ...software,
      ui_status: getUiStatus(
        software,
        true,
        hostSoftwareUpdatedAt,
        recentlyUpdatedSoftwareIds
      ),
    }));
  }, [selfServiceData, recentlyUpdatedSoftwareIds, hostSoftwareUpdatedAt]);

  const selectedSoftwareForUninstall = useRef<{
    softwareId: number;
    softwareName: string;
    softwareInstallerType?: string;
    version: string;
  } | null>(null);

  const selectedSoftwareForInstructions = useRef<{
    softwareName: string;
    softwareSource: string;
  } | null>(null);

  const queryKey = useMemo<IDeviceSoftwareQueryKey[]>(() => {
    return [
      {
        scope: "device_software",
        id: deviceToken,
        page: 0, // Pagination is clientside
        query: "", // Search is now client-side to reduce API calls
        ...DEFAULT_SELF_SERVICE_CONFIG.api,
      },
    ];
  }, [deviceToken]);

  // Fetch self-service software (regular API call)
  const {
    isLoading,
    isError,
    isFetching,
    refetch: refetchSelfServiceData,
  } = useQuery<
    IGetDeviceSoftwareResponse,
    AxiosError,
    IGetDeviceSoftwareResponse,
    IDeviceSoftwareQueryKey[]
  >(queryKey, (context) => deviceApi.getDeviceSoftware(context.queryKey[0]), {
    ...DEFAULT_USE_QUERY_OPTIONS,
    enabled: isSoftwareEnabled,
    keepPreviousData: true,
    onSuccess: (response) => {
      setSelfServiceData(response);
    },
  });

  // After host details polling (in parent) finishes, refetch software data.
  // Ensures self service data reflects updates to installed_versions from the latest host details.
  useEffect(() => {
    // Detect completion of the host details polling (in parent)
    // Once host details polling completes, refetch software data to retreive updated installed_versions keyed from host details data
    if (isAwaitingHostDetailsPolling.current && !isHostDetailsPolling) {
      refetchSelfServiceData();
    }
    isAwaitingHostDetailsPolling.current = isHostDetailsPolling;
  }, [isHostDetailsPolling, refetchSelfServiceData]);

  // Poll for pending installs/uninstalls
  const { refetch: refetchForPendingInstallsOrUninstalls } = useQuery<
    IGetDeviceSoftwareResponse,
    AxiosError
  >(
    ["pending_installs", queryKey[0]],
    () => deviceApi.getDeviceSoftware(queryKey[0]),
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

        // Compare new set with the previous set
        const previouslyPending = [...pendingSoftwareIdsRef.current];
        const completedAppIds = previouslyPending.filter(
          (id) => !newPendingSet.has(id)
        );
        if (completedAppIds.length > 0) {
          // Some pending installs/uninstalls finished during the last refresh
          // Mark as recently updated if the user actually initiated the action
          // so the UI shows the "recently updated" status instead of "update available"
          // or similar for install/uninstall
          setRecentlyUpdatedSoftwareIds((prev) => {
            const next = new Set(prev);
            completedAppIds.forEach((idStr) => {
              const id = Number(idStr);
              if (userActionIdsRef.current.has(id)) {
                next.add(id);
                userActionIdsRef.current.delete(id);
                // Register a timeout for this id (for cleanup and removal)
                registerUserSoftwareAction(id);
              }
            });
            return next;
          });

          // Some pending installs finished during the last refresh
          // Trigger an additional refetch to ensure UI status is up-to-date
          // If already refetching, queue another refetch
          refetchHostDetails();
        }

        // Compare new set with the previous set
        const setsAreEqual =
          newPendingSet.size === pendingSoftwareIdsRef.current.size &&
          [...newPendingSet].every((id) =>
            pendingSoftwareIdsRef.current.has(id)
          );

        if (newPendingSet.size > 0) {
          // If the set changed, update and continue polling
          if (!setsAreEqual) {
            pendingSoftwareIdsRef.current = newPendingSet;
            setSelfServiceData(response);
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
          pendingSoftwareIdsRef.current = new Set();
          if (pollingTimeoutIdRef.current) {
            clearTimeout(pollingTimeoutIdRef.current);
            pollingTimeoutIdRef.current = null;
          }
          setSelfServiceData(response);
        }
      },
      onError: () => {
        pendingSoftwareIdsRef.current = new Set();
        renderFlash(
          "error",
          "We're having trouble checking pending installs. Please refresh the page."
        );
      },
    }
  );

  const startPollingForPendingInstallsOrUninstalls = useCallback(
    (pendingIds: string[]) => {
      const newSet = new Set(pendingIds);
      const setsAreEqual =
        newSet.size === pendingSoftwareIdsRef.current.size &&
        [...newSet].every((id) => pendingSoftwareIdsRef.current.has(id));
      if (!setsAreEqual) {
        pendingSoftwareIdsRef.current = newSet;

        // Clear any existing timeout to avoid overlap
        if (pollingTimeoutIdRef.current) {
          clearTimeout(pollingTimeoutIdRef.current);
        }
        refetchForPendingInstallsOrUninstalls(); // Starts polling for pending installs
      }
    },
    [refetchForPendingInstallsOrUninstalls]
  );

  // On initial load or data change, check for pending installs/uninstalls
  useEffect(() => {
    const pendingSoftware = selfServiceData?.software.filter(
      (software) =>
        software.status === "pending_install" ||
        software.status === "pending_uninstall"
    );
    const pendingIds = pendingSoftware?.map((s) => String(s.id)) ?? [];
    if (pendingIds.length > 0) {
      startPollingForPendingInstallsOrUninstalls(pendingIds);
    }
  }, [selfServiceData, startPollingForPendingInstallsOrUninstalls]);

  const onInstallOrUninstall = useCallback(() => {
    refetchForPendingInstallsOrUninstalls();
  }, [refetchForPendingInstallsOrUninstalls]);

  const onClickInstallAction = useCallback(
    async (softwareId: number, isScriptPackage = false) => {
      try {
        await deviceApi.installSelfServiceSoftware(deviceToken, softwareId);
        if (isMountedRef.current) {
          onInstallOrUninstall();
          registerUserSoftwareAction(softwareId);
        }
      } catch (error) {
        // We only show toast message if API returns an error
        renderFlash(
          "error",
          `Couldn't ${isScriptPackage ? "run" : "install"}. Please try again.`
        );
      }
    },
    [deviceToken, onInstallOrUninstall, registerUserSoftwareAction, renderFlash]
  );

  const onClickUninstallAction = useCallback(
    (hostSW: IDeviceSoftwareWithUiStatus) => {
      selectedSoftwareForUninstall.current = {
        softwareId: hostSW.id,
        softwareName: hostSW.name,
        softwareInstallerType: getExtensionFromFileName(
          hostSW.software_package?.name || ""
        ),
        version: hostSW.software_package?.version || "",
      };
      setShowUninstallSoftwareModal(true);
    },
    []
  );

  const onClickOpenInstructionsAction = useCallback(
    (hostSW: IDeviceSoftwareWithUiStatus) => {
      selectedSoftwareForInstructions.current = {
        softwareName: getInstallerName(hostSW),
        softwareSource: hostSW.source,
      };
      setShowOpenInstructionsModal(true);
    },
    []
  );

  const onClickUpdateAction = useCallback(
    async (id: number) => {
      try {
        await deviceApi.installSelfServiceSoftware(deviceToken, id);
        registerUserSoftwareAction(id);
        onInstallOrUninstall();
      } catch (error) {
        // Only show toast message if API returns an error
        renderFlash("error", "Couldn't update software. Please try again.");
      }
    },
    [deviceToken, registerUserSoftwareAction, onInstallOrUninstall, renderFlash]
  );

  const onClickUpdateAll = useCallback(async () => {
    const updateAvailableSoftware = enhancedSoftware.filter(
      (software) =>
        software.ui_status === "update_available" ||
        software.ui_status === "failed_install_update_available" ||
        software.ui_status === "failed_uninstall_update_available"
    );

    // This should not happen
    if (!updateAvailableSoftware.length) {
      renderFlash("success", "No updates available.");
      return;
    }

    // Trigger updates
    const promises = updateAvailableSoftware.map((software) =>
      deviceApi.installSelfServiceSoftware(deviceToken, software.id)
    );

    const results = await Promise.allSettled(promises);

    // Only show toast message for updates that API returns an error
    const failedUpdates = results
      .map((result, idx) =>
        result.status === "rejected" ? updateAvailableSoftware[idx] : null
      )
      .filter(Boolean) as typeof updateAvailableSoftware;

    if (failedUpdates.length > 0) {
      const errorNotifications: INotification[] = failedUpdates.map(
        (software) => ({
          id: `update-error-${software.id}`,
          alertType: "error",
          isVisible: true,
          message: `Couldn't update ${software.name}. Please try again.`,
          persistOnPageChange: false,
        })
      );

      renderMultiFlash({
        notifications: errorNotifications,
      });
    }

    // Only register success IDs for follow‑up “recently updated” handling
    results.forEach((result, idx) => {
      if (result.status === "fulfilled") {
        registerUserSoftwareAction(updateAvailableSoftware[idx].id);
      }
    });
    // Refresh data after update is triggered
    onInstallOrUninstall();
  }, [
    deviceToken,
    renderFlash,
    renderMultiFlash,
    enhancedSoftware,
    registerUserSoftwareAction,
    onInstallOrUninstall,
  ]);

  const onShowUpdateDetails = useCallback(
    (software?: IDeviceSoftware) => {
      setSelectedUpdateDetails(software);
    },
    [setSelectedUpdateDetails]
  );

  const onShowInstallDetails = useCallback(
    (hostSoftware?: IHostSoftware) => {
      setSelectedHostSWInstallDetails(hostSoftware);
    },
    [setSelectedHostSWInstallDetails]
  );

  const onShowIpaInstallDetails = useCallback(
    (hostSoftware?: IHostSoftware) => {
      setSelectedHostSWIpaInstallDetails(hostSoftware);
    },
    [setSelectedHostSWIpaInstallDetails]
  );

  const onShowScriptDetails = useCallback(
    (hostSoftware?: IHostSoftware) => {
      setSelectedHostSWScriptDetails(hostSoftware);
    },
    [setSelectedHostSWScriptDetails]
  );

  const onShowVPPInstallDetails = useCallback(
    (s: IVPPHostSoftware) => {
      setSelectedVPPInstallDetails(s);
    },
    [setSelectedVPPInstallDetails]
  );

  const onShowUninstallDetails = useCallback(
    (uninstallModalDetails: ISWUninstallDetailsParentState) => {
      setSelectedHostSWUninstallDetails(uninstallModalDetails);
    },
    [setSelectedHostSWUninstallDetails]
  );

  const onClickFailedUpdateStatus = (hostSoftware: IHostSoftware) => {
    const lastInstall = getLastInstall(hostSoftware);

    if (onShowInstallDetails && lastInstall) {
      if ("command_uuid" in lastInstall) {
        // vpp software
        onShowVPPInstallDetails({
          ...hostSoftware,
          commandUuid: lastInstall.command_uuid,
        });
      } else if ("install_uuid" in lastInstall) {
        // other software
        onShowInstallDetails(hostSoftware);
      } else {
        onShowInstallDetails(undefined);
      }
    }
  };

  const onExitSoftwareInstructionsModal = () => {
    selectedSoftwareForUninstall.current = null;
    setShowOpenInstructionsModal(false);
  };

  const onExitUninstallSoftwareModal = () => {
    selectedSoftwareForUninstall.current = null;
    setShowUninstallSoftwareModal(false);
  };

  const onSuccessUninstallSoftwareModal = () => {
    selectedSoftwareForUninstall.current = null;
    setShowUninstallSoftwareModal(false);
    onInstallOrUninstall();
  };

  // TODO: handle empty state better, this is just a placeholder for now
  // TODO: what should happen if query params are invalid (e.g., page is negative or exceeds the
  // available results)?
  const isEmpty =
    !selfServiceData?.software.length &&
    !selfServiceData?.meta.has_previous_results &&
    queryParams.query === "";
  const isEmptySearch =
    !selfServiceData?.software.length &&
    !selfServiceData?.meta.has_previous_results &&
    queryParams.query !== "";

  const tableConfig = useMemo(() => {
    return generateSoftwareTableHeaders({
      onShowUpdateDetails,
      onShowInstallDetails,
      onShowIpaInstallDetails,
      onShowScriptDetails,
      onShowVPPInstallDetails,
      onShowUninstallDetails,
      onClickInstallAction,
      onClickUninstallAction,
      onClickOpenInstructionsAction,
    });
  }, [
    onShowUpdateDetails,
    onShowInstallDetails,
    onShowIpaInstallDetails,
    onShowScriptDetails,
    onShowVPPInstallDetails,
    onShowUninstallDetails,
    onClickInstallAction,
    onClickUninstallAction,
    onClickOpenInstructionsAction,
  ]);

  if (isMobileView)
    return (
      <SelfServiceCard
        contactUrl={contactUrl}
        queryParams={queryParams}
        enhancedSoftware={enhancedSoftware}
        selfServiceData={selfServiceData}
        tableConfig={tableConfig}
        isLoading={isLoading}
        isError={isError}
        isFetching={isFetching}
        isEmpty={isEmpty}
        isEmptySearch={isEmptySearch}
        router={router}
        pathname={pathname}
        isMobileView={isMobileView}
        onClickInstallAction={onClickInstallAction}
      />
    );

  return (
    <div className={baseClass}>
      <UpdatesCard
        enhancedSoftware={enhancedSoftware}
        isLoading={isLoading}
        isError={isError}
        onClickUpdateAll={onClickUpdateAll}
        onClickUpdateAction={onClickUpdateAction}
        onClickFailedUpdateStatus={onClickFailedUpdateStatus}
      />
      <SelfServiceCard
        contactUrl={contactUrl}
        queryParams={queryParams}
        enhancedSoftware={enhancedSoftware}
        selfServiceData={selfServiceData}
        tableConfig={tableConfig}
        isLoading={isLoading}
        isError={isError}
        isFetching={isFetching}
        isEmpty={isEmpty}
        isEmptySearch={isEmptySearch}
        router={router}
        pathname={pathname}
      />
      {showUninstallSoftwareModal && selectedSoftwareForUninstall.current && (
        <UninstallSoftwareModal
          softwareId={selectedSoftwareForUninstall.current.softwareId}
          softwareName={selectedSoftwareForUninstall.current.softwareName}
          token={deviceToken}
          onExit={onExitUninstallSoftwareModal}
          onSuccess={onSuccessUninstallSoftwareModal}
        />
      )}
      {showOpenInstructionsModal && selectedSoftwareForInstructions.current && (
        <SoftwareInstructionsModal
          softwareName={selectedSoftwareForInstructions.current.softwareName}
          softwareSource={
            selectedSoftwareForInstructions.current.softwareSource
          }
          onExit={onExitSoftwareInstructionsModal}
        />
      )}
      {selectedHostSWInstallDetails && (
        <SoftwareInstallDetailsModal
          hostSoftware={selectedHostSWInstallDetails}
          details={{
            host_display_name: hostDisplayName,
            install_uuid:
              selectedHostSWInstallDetails.software_package?.last_install
                ?.install_uuid,
          }}
          onRetry={onClickInstallAction}
          onCancel={() => setSelectedHostSWInstallDetails(undefined)}
          deviceAuthToken={deviceToken}
          contactUrl={contactUrl}
        />
      )}
      {selectedHostSWIpaInstallDetails && (
        <SoftwareIpaInstallDetailsModal
          hostSoftware={selectedHostSWIpaInstallDetails}
          details={{
            hostDisplayName,
            fleetInstallStatus: selectedHostSWIpaInstallDetails.status,
            appName:
              selectedHostSWIpaInstallDetails.display_name ||
              selectedHostSWIpaInstallDetails.name,
            commandUuid:
              selectedHostSWIpaInstallDetails.software_package?.last_install
                ?.install_uuid, // slightly redundant, see explanation in `SoftwareInstallDetailsModal
          }}
          onRetry={onClickInstallAction}
          onCancel={() => setSelectedHostSWIpaInstallDetails(undefined)}
          deviceAuthToken={deviceToken}
        />
      )}
      {selectedHostSWScriptDetails && (
        <SoftwareScriptDetailsModal
          hostSoftware={selectedHostSWScriptDetails}
          details={{
            host_display_name: hostDisplayName,
            install_uuid:
              selectedHostSWScriptDetails.software_package?.last_install
                ?.install_uuid,
          }}
          onRerun={onClickInstallAction}
          onCancel={() => setSelectedHostSWScriptDetails(undefined)}
          deviceAuthToken={deviceToken}
          contactUrl={contactUrl}
        />
      )}
      {selectedVPPInstallDetails && (
        <VppInstallDetailsModal
          deviceAuthToken={deviceToken}
          details={{
            fleetInstallStatus: selectedVPPInstallDetails.status,
            hostDisplayName,
            appName:
              selectedVPPInstallDetails.display_name ||
              selectedVPPInstallDetails.name,
            commandUuid: selectedVPPInstallDetails.commandUuid,
          }}
          hostSoftware={selectedVPPInstallDetails}
          onCancel={() => setSelectedVPPInstallDetails(null)}
          onRetry={onClickInstallAction}
        />
      )}
      {selectedHostSWUninstallDetails && (
        <SoftwareUninstallDetailsModal
          {...selectedHostSWUninstallDetails}
          hostDisplayName={hostDisplayName}
          onCancel={() => setSelectedHostSWUninstallDetails(undefined)}
          onRetry={onClickUninstallAction}
          deviceAuthToken={deviceToken}
          contactUrl={contactUrl}
        />
      )}
      {selectedUpdateDetails && (
        <SoftwareUpdateModal
          hostDisplayName={hostDisplayName}
          software={selectedUpdateDetails}
          onUpdate={onClickInstallAction}
          onExit={() => setSelectedUpdateDetails(undefined)}
          isDeviceUser
        />
      )}
    </div>
  );
};

export default SoftwareSelfService;
