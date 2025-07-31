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
import { IDeviceSoftware, IHostSoftware } from "interfaces/software";

import deviceApi, {
  IDeviceSoftwareQueryKey,
  IGetDeviceSoftwareResponse,
} from "services/entities/device_user";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { getExtensionFromFileName } from "utilities/file/fileUtils";

import { ISoftwareUninstallDetails } from "components/ActivityDetails/InstallDetails/SoftwareUninstallDetailsModal/SoftwareUninstallDetailsModal";

import UpdatesCard from "./UpdatesCard";
import SelfServiceCard from "./SelfServiceCard";
import SoftwareUpdateModal from "../SoftwareUpdateModal";
import UninstallSoftwareModal from "./UninstallSoftwareModal";
import { generateSoftwareTableHeaders } from "./SelfServiceTableConfig";
import { parseHostSoftwareQueryParams } from "../HostSoftware";
import { InstallOrCommandUuid } from "../InstallStatusCell/InstallStatusCell";
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

export interface ISoftwareSelfServiceProps {
  contactUrl: string;
  deviceToken: string;
  isSoftwareEnabled?: boolean;
  pathname: string;
  queryParams: ReturnType<typeof parseHostSoftwareQueryParams>;
  router: InjectedRouter;
  onShowInstallDetails: (uuid?: InstallOrCommandUuid) => void;
  onShowUninstallDetails: (details?: ISoftwareUninstallDetails) => void;
  refetchHostDetails: () => void;
  isHostDetailsPolling: boolean;
  hostDisplayName: string;
}

const SoftwareSelfService = ({
  contactUrl,
  deviceToken,
  isSoftwareEnabled,
  pathname,
  queryParams,
  router,
  onShowInstallDetails,
  onShowUninstallDetails,
  refetchHostDetails,
  isHostDetailsPolling,
  hostDisplayName,
}: ISoftwareSelfServiceProps) => {
  const { renderFlash, renderMultiFlash } = useContext(NotificationContext);

  const [selfServiceData, setSelfServiceData] = useState<
    IGetDeviceSoftwareResponse | undefined
  >(undefined);
  const [selectedUpdateDetails, setSelectedUpdateDetails] = useState<
    IDeviceSoftware | undefined
  >(undefined);
  const [showUninstallSoftwareModal, setShowUninstallSoftwareModal] = useState(
    false
  );

  // Enhance with `ui_status`. See helpers.
  const enhancedSoftware = useMemo(() => {
    if (!selfServiceData) return [];
    return selfServiceData.software.map((software) => ({
      ...software,
      ui_status: getUiStatus(software, true),
    }));
  }, [selfServiceData]);

  const selectedSoftware = useRef<{
    softwareId: number;
    softwareName: string;
    softwareInstallerType?: string;
    version: string;
  } | null>(null);

  const pendingSoftwareSetRef = useRef<Set<string>>(new Set()); // Track for polling
  const pollingTimeoutIdRef = useRef<NodeJS.Timeout | null>(null);
  const isAwaitingHostDetailsPolling = useRef(isHostDetailsPolling);

  const queryKey = useMemo<IDeviceSoftwareQueryKey[]>(() => {
    return [
      {
        scope: "device_software",
        id: deviceToken,
        page: queryParams.page,
        query: "", // Search is now client-side to reduce API calls
        ...DEFAULT_SELF_SERVICE_QUERY_PARAMS,
      },
    ];
  }, [deviceToken, queryParams.page]);

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

  const onClickFailedUpdateStatus = (s: IHostSoftware) => {
    const lastInstall = getLastInstall(s);

    if (onShowInstallDetails && lastInstall) {
      if ("command_uuid" in lastInstall) {
        onShowInstallDetails({ command_uuid: lastInstall.command_uuid });
      } else if ("install_uuid" in lastInstall) {
        onShowInstallDetails({ install_uuid: lastInstall.install_uuid });
      } else {
        onShowInstallDetails(undefined);
      }
    }
  };

  const onExitUninstallSoftwareModal = () => {
    selectedSoftware.current = null;
    setShowUninstallSoftwareModal(false);
  };

  const onSuccessUninstallSoftwareModal = () => {
    selectedSoftware.current = null;
    setShowUninstallSoftwareModal(false);
    onInstallOrUninstall();
  };

  const tableConfig = useMemo(() => {
    return generateSoftwareTableHeaders({
      deviceToken,
      onInstallOrUninstall,
      onShowUpdateDetails,
      onShowInstallDetails,
      onShowUninstallDetails,
      onClickInstallAction,
      onClickUninstallAction: (software) => {
        selectedSoftware.current = {
          softwareId: software.id,
          softwareName: software.name,
          softwareInstallerType: getExtensionFromFileName(
            software.software_package?.name || ""
          ),
          version: software.software_package?.version || "",
        };
        setShowUninstallSoftwareModal(true);
      },
    });
  }, [
    deviceToken,
    onInstallOrUninstall,
    onClickInstallAction,
    onShowUpdateDetails,
    onShowInstallDetails,
    onShowUninstallDetails,
  ]);

  return (
    <div className={baseClass}>
      <UpdatesCard
        contactUrl={contactUrl}
        enhancedSoftware={enhancedSoftware}
        onClickUpdateAction={onClickUpdateAction}
        onClickUpdateAll={onClickUpdateAll}
        onClickFailedUpdateStatus={onClickFailedUpdateStatus}
        isLoading={isLoading}
        isError={isError}
      />
      <SelfServiceCard
        contactUrl={contactUrl}
        enhancedSoftware={enhancedSoftware}
        tableConfig={tableConfig}
        selfServiceData={selfServiceData}
        isLoading={isLoading}
        isError={isError}
        isFetching={isFetching}
        queryParams={queryParams}
        router={router}
        pathname={pathname}
      />
      {showUninstallSoftwareModal && selectedSoftware.current && (
        <UninstallSoftwareModal
          softwareId={selectedSoftware.current.softwareId}
          softwareName={selectedSoftware.current.softwareName}
          softwareInstallerType={selectedSoftware.current.softwareInstallerType}
          version={selectedSoftware.current.version}
          token={deviceToken}
          onExit={onExitUninstallSoftwareModal}
          onSuccess={onSuccessUninstallSoftwareModal}
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
