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
import { getPathWithQueryParams } from "utilities/url";
import { getExtensionFromFileName } from "utilities/file/fileUtils";

import { SingleValue } from "react-select-5";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import TableContainer from "components/TableContainer";
import EmptySoftwareTable from "pages/SoftwarePage/components/tables/EmptySoftwareTable";
import Button from "components/buttons/Button";
import Card from "components/Card";
import CardHeader from "components/CardHeader";
import CustomLink from "components/CustomLink";
import DeviceUserError from "components/DeviceUserError";
import EmptyTable from "components/EmptyTable";
import Spinner from "components/Spinner";
import SearchField from "components/forms/fields/SearchField";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import Pagination from "components/Pagination";

import SoftwareUninstallDetailsModal, {
  ISWUninstallDetailsParentState,
} from "components/ActivityDetails/InstallDetails/SoftwareUninstallDetailsModal/SoftwareUninstallDetailsModal";
import SoftwareInstallDetailsModal from "components/ActivityDetails/InstallDetails/SoftwareInstallDetailsModal";
import { VppInstallDetailsModal } from "components/ActivityDetails/InstallDetails/VppInstallDetailsModal/VppInstallDetailsModal";

import SoftwareUpdateModal from "../SoftwareUpdateModal";
import UninstallSoftwareModal from "./UninstallSoftwareModal";

import { generateSoftwareTableHeaders } from "./SelfServiceTableConfig";
import { parseHostSoftwareQueryParams } from "../HostSoftware";
import { getLastInstall } from "../../HostSoftwareLibrary/helpers";

import {
  CATEGORIES_NAV_ITEMS,
  filterSoftwareByCategory,
  ICategory,
} from "./helpers";
import CategoriesMenu from "./CategoriesMenu";
import { getUiStatus } from "../helpers";
import UpdateSoftwareItem from "./UpdateSoftwareItem";

const baseClass = "software-self-service";

// These default params are not subject to change by the user
const DEFAULT_SELF_SERVICE_QUERY_PARAMS = {
  per_page: 9999, // Note: There is no API pagination on this page because of time constraints (e.g. categories and install statuses are not filtered by API)
  order_key: "name",
  order_direction: "asc",
  self_service: true,
  category_id: undefined,
} as const;

const DEFAULT_CLIENT_SIDE_PAGINATION = 20;

export interface ISoftwareSelfServiceProps {
  contactUrl: string;
  deviceToken: string;
  isSoftwareEnabled?: boolean;
  pathname: string;
  queryParams: ReturnType<typeof parseHostSoftwareQueryParams>;
  router: InjectedRouter;
  refetchHostDetails: () => void;
  isHostDetailsPolling: boolean;
  hostDisplayName: string;
}

const getUpdatesPageSize = (width: number): number => {
  if (width >= 1400) return 4;
  if (width >= 880) return 3;
  return 2;
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
  const [updatesPage, setUpdatesPage] = useState(0);
  const [updatesPageSize, setUpdatesPageSize] = useState(() =>
    getUpdatesPageSize(window.innerWidth)
  );

  const enhancedSoftware = useMemo(() => {
    if (!selfServiceData) return [];
    return selfServiceData.software.map((software) => ({
      ...software,
      ui_status: getUiStatus(software, true),
    }));
  }, [selfServiceData]);

  const updateSoftware = enhancedSoftware.filter(
    (software) =>
      software.ui_status === "updating" ||
      software.ui_status === "pending_update" || // Should never show as self-service = host online
      software.ui_status === "update_available" ||
      software.ui_status === "failed_install_update_available" ||
      software.ui_status === "failed_uninstall_update_available"
  );

  // The page size only changes at 2 breakpoints and state update is very lightweight.
  // Page size controls the number of shown cards and current page; no API calls or
  // expensive UI updates occur so debouncing the resize handler isn’t necessary.
  useEffect(() => {
    const handleResize = () => {
      const newPageSize = getUpdatesPageSize(window.innerWidth);
      setUpdatesPageSize(() => {
        const newTotalPages = Math.ceil(updateSoftware.length / newPageSize);
        setUpdatesPage((prevPage) => {
          // If the current page is now out of range, go to the last valid page
          return Math.min(prevPage, Math.max(0, newTotalPages - 1));
        });
        return newPageSize;
      });
    };

    window.addEventListener("resize", handleResize);
    return () => window.removeEventListener("resize", handleResize);
  }, [updateSoftware.length]);

  const paginatedUpdates = useMemo(() => {
    const start = updatesPage * updatesPageSize;
    return updateSoftware.slice(start, start + updatesPageSize);
  }, [updateSoftware, updatesPage, updatesPageSize]);

  const totalUpdatesPages = Math.ceil(updateSoftware.length / updatesPageSize);

  const onNextUpdatesPage = () => {
    setUpdatesPage((prev) => Math.min(prev + 1, totalUpdatesPages - 1));
  };

  const onPreviousUpdatesPage = () => {
    setUpdatesPage((prev) => Math.max(prev - 1, 0));
  };

  const disableUpdateAllButton = useMemo(() => {
    // Disable if all statuses are "updating"
    return (
      updateSoftware.length > 0 &&
      updateSoftware.every((software) => software.ui_status === "updating")
    );
  }, [updateSoftware]);

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

  const onClickUninstallAction = useCallback(
    (hostSW: IHostSoftwareWithUiStatus) => {
      selectedSoftware.current = {
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

  const onSearchQueryChange = (value: string) => {
    router.push(
      getPathWithQueryParams(pathname, {
        query: value,
        category_id: queryParams.category_id,
        page: 0, // Always reset to page 0 when searching
      })
    );
  };

  const onCategoriesDropdownChange = (
    option: SingleValue<CustomOptionType>
  ) => {
    router.push(
      getPathWithQueryParams(pathname, {
        category_id: option?.value !== "undefined" ? option?.value : undefined,
        query: queryParams.query,
        page: 0, // Always reset to page 0 when searching
      })
    );
  };

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

  const onExitUninstallSoftwareModal = () => {
    selectedSoftware.current = null;
    setShowUninstallSoftwareModal(false);
  };

  const onSuccessUninstallSoftwareModal = () => {
    selectedSoftware.current = null;
    setShowUninstallSoftwareModal(false);
    onInstallOrUninstall();
  };

  const onNextPage = useCallback(() => {
    router.push(
      getPathWithQueryParams(pathname, {
        query: queryParams.query,
        category_id: queryParams.category_id,
        page: queryParams.page + 1,
      })
    );
  }, [pathname, queryParams, router]);

  const onPrevPage = useCallback(() => {
    router.push(
      getPathWithQueryParams(pathname, {
        query: queryParams.query,
        category_id: queryParams.category_id,
        page: queryParams.page - 1,
      })
    );
  }, [pathname, queryParams, router]);

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
    });
  }, [
    onShowUpdateDetails,
    onShowInstallDetails,
    onShowVPPInstallDetails,
    onShowUninstallDetails,
    onClickInstallAction,
    onClickUninstallAction,
  ]);

  const renderUpdatesCard = () => {
    if (isLoading) {
      return <Spinner />;
    }

    if (isError) {
      return <DeviceUserError />;
    }

    return (
      <>
        <div className={`${baseClass}__items`}>
          {paginatedUpdates.map((s) => {
            return (
              <UpdateSoftwareItem
                key={s.id}
                software={s}
                onClickUpdateAction={onClickUpdateAction}
                onShowInstallerDetails={() => {
                  onClickFailedUpdateStatus(s);
                }}
              />
            );
          })}
        </div>
        <Pagination
          disableNext={updatesPage >= totalUpdatesPages - 1}
          disablePrev={updatesPage === 0}
          hidePagination={
            updatesPage >= totalUpdatesPages - 1 && updatesPage === 0
          }
          onNextPage={onNextUpdatesPage}
          onPrevPage={onPreviousUpdatesPage}
          className={`${baseClass}__pagination`}
        />
      </>
    );
  };

  const renderSelfServiceCard = () => {
    const renderHeaderFilters = () => (
      <div className={`${baseClass}__header-filters`}>
        <SearchField
          placeholder="Search by name"
          onChange={onSearchQueryChange}
          defaultValue={queryParams.query}
        />
        <DropdownWrapper
          options={CATEGORIES_NAV_ITEMS.map((category: ICategory) => ({
            ...category,
            value: String(category.id), // DropdownWrapper only accepts string
          }))}
          value={String(queryParams.category_id || 0)}
          onChange={onCategoriesDropdownChange}
          name="categories-dropdown"
          className={`${baseClass}__categories-dropdown`}
        />
      </div>
    );

    const renderCategoriesMenu = () => (
      <CategoriesMenu
        queryParams={queryParams}
        categories={CATEGORIES_NAV_ITEMS}
      />
    );

    if (isLoading) {
      return <Spinner />;
    }

    if (isError) {
      return <DeviceUserError />; // Only shown on DeviceUserPage not HostDetailsPage
    }

    // No self-service software available hides categories menu and header filters
    if ((isEmpty || !selfServiceData) && !isFetching) {
      return (
        <>
          <EmptyTable
            graphicName="empty-software"
            header="No self-service software available yet"
            info="Your organization didn't add any self-service software. If you need any, reach out to your IT department."
          />
        </>
      );
    }

    return (
      <>
        {renderHeaderFilters()}
        <div className={`${baseClass}__table`}>
          {renderCategoriesMenu()}
          <TableContainer
            columnConfigs={tableConfig}
            data={filterSoftwareByCategory(
              enhancedSoftware || [],
              queryParams.category_id
            )}
            isLoading={isFetching}
            defaultSortHeader={DEFAULT_SELF_SERVICE_QUERY_PARAMS.order_key}
            defaultSortDirection={
              DEFAULT_SELF_SERVICE_QUERY_PARAMS.order_direction
            }
            pageIndex={0}
            disableNextPage={selfServiceData?.meta.has_next_results === false}
            pageSize={DEFAULT_CLIENT_SIDE_PAGINATION}
            searchQuery={queryParams.query} // Search is now client-side to reduce API calls
            searchQueryColumn="name"
            isClientSideFilter
            isClientSidePagination
            emptyComponent={() => {
              return isEmptySearch ? (
                <EmptyTable
                  graphicName="empty-search-question"
                  header="No items match the current search criteria"
                  info={
                    <>
                      Not finding what you&apos;re looking for?{" "}
                      <CustomLink
                        url={contactUrl}
                        text="Reach out to IT"
                        newTab
                      />
                    </>
                  }
                />
              ) : (
                <EmptySoftwareTable />
              );
            }}
            showMarkAllPages={false}
            isAllPagesSelected={false}
            disableTableHeader
            disableCount
          />
        </div>

        <Pagination
          disableNext={selfServiceData?.meta.has_next_results === false}
          disablePrev={selfServiceData?.meta.has_previous_results === false}
          hidePagination={
            selfServiceData?.meta.has_next_results === false &&
            selfServiceData?.meta.has_previous_results === false
          }
          onNextPage={onNextPage}
          onPrevPage={onPrevPage}
          className={`${baseClass}__pagination`}
        />
      </>
    );
  };

  return (
    <div className={baseClass}>
      {paginatedUpdates.length > 0 && (
        <Card
          className={`${baseClass}__updates-card`}
          borderRadiusSize="xxlarge"
          paddingSize="xlarge"
          includeShadow
        >
          <div className={`${baseClass}__header`}>
            <CardHeader
              header="Updates"
              subheader={
                <>
                  Your device has outdated software. Update to address potential
                  security vulnerabilities or compatibility issues.
                </>
              }
            />
            <Button
              disabled={disableUpdateAllButton}
              onClick={onClickUpdateAll}
            >
              Update all
            </Button>
          </div>
          {renderUpdatesCard()}
        </Card>
      )}
      <Card
        className={`${baseClass}__self-service-card`}
        borderRadiusSize="xxlarge"
        paddingSize="xlarge"
        includeShadow
      >
        <CardHeader
          header="Self-service"
          subheader={
            <>
              Install organization-approved apps provided by your IT department.{" "}
              {contactUrl && (
                <span>
                  If you need help,{" "}
                  <CustomLink url={contactUrl} text="reach out to IT" newTab />
                </span>
              )}
            </>
          }
        />
        {renderSelfServiceCard()}
      </Card>
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
            fleetInstallStatus:
              selectedVPPInstallDetails.status || "pending_install",
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
