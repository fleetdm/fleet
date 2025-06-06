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
import Card from "components/Card";
import CardHeader from "components/CardHeader";
import CustomLink from "components/CustomLink";
import DeviceUserError from "components/DeviceUserError";
import EmptyTable from "components/EmptyTable";
import Spinner from "components/Spinner";
import SearchField from "components/forms/fields/SearchField";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import Pagination from "components/Pagination";

import UninstallSoftwareModal from "./UninstallSoftwareModal";
import {
  InstallOrCommandUuid,
  generateSoftwareTableHeaders as generateDeviceSoftwareTableConfig,
} from "./SelfServiceTableConfig";
import { parseHostSoftwareQueryParams } from "../HostSoftware";

import {
  CATEGORIES_NAV_ITEMS,
  filterSoftwareByCategory,
  ICategory,
} from "./helpers";
import CategoriesMenu from "./CategoriesMenu";

const baseClass = "software-self-service";

// These default params are not subject to change by the user
const DEFAULT_SELF_SERVICE_QUERY_PARAMS = {
  per_page: 9999, // Note: There is no pagination on this page because of time constraints (e.g. categories and install statuses are not filtered by API)
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
  onShowInstallerDetails: (uuid?: InstallOrCommandUuid) => void;
  onShowUninstallDetails: (scriptExecutionId?: string) => void;
}

const SoftwareSelfService = ({
  contactUrl,
  deviceToken,
  isSoftwareEnabled,
  pathname,
  queryParams,
  router,
  onShowInstallerDetails,
  onShowUninstallDetails,
}: ISoftwareSelfServiceProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const [selfServiceData, setSelfServiceData] = useState<
    IGetDeviceSoftwareResponse | undefined
  >(undefined);
  const [showUninstallSoftwareModal, setShowUninstallSoftwareModal] = useState(
    false
  );

  const selectedSoftware = useRef<{
    softwareId: number;
    softwareName: string;
    softwareInstallerType?: string;
    version: string;
  } | null>(null);

  const pendingSoftwareSetRef = useRef<Set<string>>(new Set()); // Track for polling
  const pollingTimeoutIdRef = useRef<NodeJS.Timeout | null>(null);

  const queryKey = useMemo<IDeviceSoftwareQueryKey[]>(() => {
    return [
      {
        scope: "device_software",
        id: deviceToken,
        page: queryParams.page,
        query: queryParams.query,
        ...DEFAULT_SELF_SERVICE_QUERY_PARAMS,
      },
    ];
  }, [deviceToken, queryParams.page, queryParams.query]);

  // Fetch self-service software (regular API call)
  const { isLoading, isError, isFetching } = useQuery<
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
    return generateDeviceSoftwareTableConfig({
      deviceToken,
      onInstall: onInstallOrUninstall,
      onShowInstallerDetails,
      onShowUninstallDetails,
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
  }, [deviceToken, onInstallOrUninstall, onShowInstallerDetails]);

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

    if (isEmpty || !selfServiceData) {
      return (
        <>
          {renderHeaderFilters()}
          <div className={`${baseClass}__table`}>
            {renderCategoriesMenu()}
            <EmptyTable
              graphicName="empty-software"
              header="No self-service software available yet"
              info="Your organization didn't add any self-service software. If you need any, reach out to your IT department."
            />
          </div>
        </>
      );
    }

    if (isFetching) {
      return (
        <>
          {renderHeaderFilters()}
          <div className={`${baseClass}__table`}>
            {renderCategoriesMenu()}
            <Spinner />
          </div>
        </>
      );
    }

    if (isEmptySearch) {
      return (
        <>
          {renderHeaderFilters()}
          <div className={`${baseClass}__table`}>
            {renderCategoriesMenu()}
            <EmptyTable
              graphicName="empty-search-question"
              header="No items match the current search criteria"
              info={
                <>
                  Not finding what you&apos;re looking for?{" "}
                  <CustomLink url={contactUrl} text="reach out to IT" newTab />
                </>
              }
            />
          </div>
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
              selfServiceData?.software || [],
              queryParams.category_id
            )}
            isLoading={isLoading}
            defaultSortHeader={DEFAULT_SELF_SERVICE_QUERY_PARAMS.order_key}
            defaultSortDirection={
              DEFAULT_SELF_SERVICE_QUERY_PARAMS.order_direction
            }
            pageIndex={0}
            disableNextPage={selfServiceData?.meta.has_next_results === false}
            pageSize={DEFAULT_SELF_SERVICE_QUERY_PARAMS.per_page}
            emptyComponent={() => (
              <EmptySoftwareTable noSearchQuery={isEmptySearch} />
            )}
            showMarkAllPages={false}
            isAllPagesSelected={false}
            searchable={false}
            disableTableHeader
            disableCount
          />
        </div>

        <Pagination
          disableNext={selfServiceData.meta.has_next_results === false}
          disablePrev={selfServiceData.meta.has_previous_results === false}
          hidePagination={
            selfServiceData.meta.has_next_results === false &&
            selfServiceData.meta.has_previous_results === false
          }
          onNextPage={onNextPage}
          onPrevPage={onPrevPage}
          className={`${baseClass}__pagination`}
        />
      </>
    );
  };

  return (
    <>
      <Card
        className={baseClass}
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
    </>
  );
};

export default SoftwareSelfService;
