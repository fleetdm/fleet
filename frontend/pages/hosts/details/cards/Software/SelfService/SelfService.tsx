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
  IHostSoftwareWithUiStatus,
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
import { VppInstallDetailsModal } from "components/ActivityDetails/InstallDetails/VppInstallDetailsModal/VppInstallDetailsModal";

import UpdatesCard from "./UpdatesCard/UpdatesCard";
import SelfServiceCard from "./SelfServiceCard/SelfServiceCard";
import SoftwareUpdateModal from "../SoftwareUpdateModal";
import UninstallSoftwareModal from "./UninstallSoftwareModal";
import SoftwareInstructionsModal from "./OpenSoftwareModal";

import { generateSoftwareTableHeaders } from "./SelfServiceTableConfig";
import { getLastInstall } from "../../HostSoftwareLibrary/helpers";

import { getUiStatus } from "../helpers";

const baseClass = "software-self-service";

// These default params are not subject to change by the user
const DEFAULT_SELF_SERVICE_QUERY_PARAMS = {
  per_page: 9999, // Note: There is no API pagination on this page because of time constraints (e.g. categories and install statuses are not filtered by API)
  order_key: "name",
  order_direction: "asc",
  self_service: true,
  category_id: undefined,
} as const;

const DEFAULT_SEARCH_QUERY = "";
const DEFAULT_SORT_DIRECTION = "asc";
const DEFAULT_SORT_HEADER = "name";
const DEFAULT_PAGE = 0;
const DEFAULT_CLIENT_SIDE_PAGINATION = 20;

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
}

export const parseSelfServiceQueryParams = (queryParams: {
  page?: string;
  query?: string;
  order_key?: string;
  order_direction?: "asc" | "desc";
  category_id?: string;
}) => {
  const searchQuery = queryParams?.query ?? DEFAULT_SEARCH_QUERY;
  const sortHeader = queryParams?.order_key ?? DEFAULT_SORT_HEADER;
  const sortDirection = queryParams?.order_direction ?? DEFAULT_SORT_DIRECTION;
  const page = queryParams?.page
    ? parseInt(queryParams.page, 10)
    : DEFAULT_PAGE;
  const pageSize = DEFAULT_CLIENT_SIDE_PAGINATION;
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

const getInstallerName = (hostSW: IHostSoftwareWithUiStatus) => {
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
}: ISoftwareSelfServiceProps) => {
  const { renderFlash, renderMultiFlash } = useContext(NotificationContext);

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

  const enhancedSoftware = useMemo(() => {
    if (!selfServiceData) return [];
    return selfServiceData.software.map((software) => ({
      ...software,
      ui_status: getUiStatus(software, true, hostSoftwareUpdatedAt),
    }));
  }, [selfServiceData, hostSoftwareUpdatedAt]);

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

  const pendingSoftwareSetRef = useRef<Set<string>>(new Set()); // Track for polling
  const pollingTimeoutIdRef = useRef<NodeJS.Timeout | null>(null);
  const isAwaitingHostDetailsPolling = useRef(isHostDetailsPolling);

  const queryKey = useMemo<IDeviceSoftwareQueryKey[]>(() => {
    return [
      {
        scope: "device_software",
        id: deviceToken,
        page: 0, // Pagination is clientside
        query: "", // Search is now client-side to reduce API calls
        ...DEFAULT_SELF_SERVICE_QUERY_PARAMS,
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

        // Refresh host details if the number of pending installs or uninstalls has decreased
        // To update the software library information
        if (newPendingSet.size < pendingSoftwareSetRef.current.size) {
          refetchHostDetails();
        }

        // Compare new set with the previous set
        const setsAreEqual =
          newPendingSet.size === pendingSoftwareSetRef.current.size &&
          [...newPendingSet].every((id) =>
            pendingSoftwareSetRef.current.has(id)
          );

        if (newPendingSet.size > 0) {
          // If the set changed, update and continue polling
          if (!setsAreEqual) {
            pendingSoftwareSetRef.current = newPendingSet;
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
          pendingSoftwareSetRef.current = new Set();
          if (pollingTimeoutIdRef.current) {
            clearTimeout(pollingTimeoutIdRef.current);
            pollingTimeoutIdRef.current = null;
          }
          setSelfServiceData(response);
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

  const startPollingForPendingInstallsOrUninstalls = useCallback(
    (pendingIds: string[]) => {
      const newSet = new Set(pendingIds);
      const setsAreEqual =
        newSet.size === pendingSoftwareSetRef.current.size &&
        [...newSet].every((id) => pendingSoftwareSetRef.current.has(id));
      if (!setsAreEqual) {
        pendingSoftwareSetRef.current = newSet;

        // Clear any existing timeout to avoid overlap
        if (pollingTimeoutIdRef.current) {
          clearTimeout(pollingTimeoutIdRef.current);
        }
        refetchForPendingInstallsOrUninstalls(); // Starts polling for pending installs
      }
    },
    [refetchForPendingInstallsOrUninstalls]
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
        await deviceApi.installSelfServiceSoftware(deviceToken, softwareId);
        if (isMountedRef.current) {
          onInstallOrUninstall();
        }
      } catch (error) {
        // We only show toast message if API returns an error
        renderFlash("error", "Couldn't install. Please try again.");
      }
    },
    [deviceToken, onInstallOrUninstall, renderFlash]
  );

  const onClickUninstallAction = useCallback(
    (hostSW: IHostSoftwareWithUiStatus) => {
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
    (hostSW: IHostSoftwareWithUiStatus) => {
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
        onInstallOrUninstall();
      } catch (error) {
        // Only show toast message if API returns an error
        renderFlash("error", "Couldn't update software. Please try again.");
      }
    },
    [deviceToken, onInstallOrUninstall, renderFlash]
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

    // Refresh the data after updates triggered
    onInstallOrUninstall();
  }, [
    deviceToken,
    renderFlash,
    renderMultiFlash,
    enhancedSoftware,
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
      onShowVPPInstallDetails,
      onShowUninstallDetails,
      onClickInstallAction,
      onClickUninstallAction,
      onClickOpenInstructionsAction,
    });
  }, [
    onShowUpdateDetails,
    onShowInstallDetails,
    onShowVPPInstallDetails,
    onShowUninstallDetails,
    onClickInstallAction,
    onClickUninstallAction,
    onClickOpenInstructionsAction,
  ]);

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
      {selectedVPPInstallDetails && (
        <VppInstallDetailsModal
          deviceAuthToken={deviceToken}
          details={{
            fleetInstallStatus: selectedVPPInstallDetails.status,
            hostDisplayName,
            appName: selectedVPPInstallDetails.name,
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
