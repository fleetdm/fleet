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

import { SingleValue } from "react-select-5";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import TableContainer from "components/TableContainer";
import EmptySoftwareTable from "pages/SoftwarePage/components/tables/EmptySoftwareTable";
import Card from "components/Card";
import CardHeader from "components/CardHeader";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import EmptyTable from "components/EmptyTable";
import Spinner from "components/Spinner";
import SearchField from "components/forms/fields/SearchField";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import Pagination from "components/Pagination";

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
}

const SoftwareSelfService = ({
  contactUrl,
  deviceToken,
  isSoftwareEnabled,
  pathname,
  queryParams,
  router,
  onShowInstallerDetails,
}: ISoftwareSelfServiceProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const [selfServiceData, setSelfServiceData] = useState<
    IGetDeviceSoftwareResponse | undefined
  >(undefined);

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
    staleTime: 7000,
    onSuccess: (response) => {
      setSelfServiceData(response);
    },
  });

  // Poll for pending installs
  const { refetch: refetchForPendingInstalls } = useQuery<
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
            .filter((software) => software.status === "pending_install")
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
            refetchForPendingInstalls();
          }, 5000);
        } else {
          // No pending installs, stop polling and refresh data
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

  const startPollingForPendingInstalls = useCallback(
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
        refetchForPendingInstalls(); // Starts polling for pending installs
      }
    },
    [refetchForPendingInstalls]
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

  // On initial load or data change, check for pending installs
  useEffect(() => {
    const pendingSoftware = selfServiceData?.software.filter(
      (software) => software.status === "pending_install"
    );
    const pendingIds = pendingSoftware?.map((s) => String(s.id)) ?? [];
    if (pendingIds.length > 0) {
      startPollingForPendingInstalls(pendingIds);
    }
  }, [selfServiceData, startPollingForPendingInstalls]);

  const onInstall = useCallback(() => {
    refetchForPendingInstalls();
  }, [refetchForPendingInstalls]);

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
      onInstall,
      onShowInstallerDetails,
    });
  }, [router]);

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
          value={String(queryParams.category_id) || ""}
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
      return <DataError />;
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
<<<<<<< HEAD
      {renderSelfServiceCard()}
=======
      {isLoading ? (
        <Spinner />
      ) : (
        <>
          {isError && <DataError verticalPaddingSize="pad-xxxlarge" />}
          {!isError && (
            <div className={baseClass}>
              {isEmpty ? (
                <EmptyTable
                  graphicName="empty-software"
                  header="No self-service software available yet"
                  info="Your organization didn't add any self-service software. If you need any, reach out to your IT department."
                />
              ) : (
                <>
                  <div className={`${baseClass}__items-count`}>
                    <b>{`${data.count} ${pluralize(data.count, "item")}`}</b>
                  </div>
                  <div className={`${baseClass}__items`}>
                    {data.software.map((s) => {
                      let uuid =
                        s.software_package?.last_install?.install_uuid ??
                        s.app_store_app?.last_install?.command_uuid;
                      if (!uuid) {
                        uuid = "";
                      }
                      // concatenating uuid so item updates with fresh data on refetch
                      const key = `${s.id}${uuid}`;
                      return (
                        <SelfServiceItem
                          key={key}
                          deviceToken={deviceToken}
                          software={s}
                          onInstall={refetch}
                        />
                      );
                    })}
                  </div>
                  <Pagination
                    disableNext={data.meta.has_next_results === false}
                    disablePrev={data.meta.has_previous_results === false}
                    hidePagination={
                      data.meta.has_next_results === false &&
                      data.meta.has_previous_results === false
                    }
                    onNextPage={onNextPage}
                    onPrevPage={onPrevPage}
                    className={`${baseClass}__pagination`}
                  />
                </>
              )}
            </div>
          )}
        </>
      )}
>>>>>>> 22a5e194df (More use of variants)
    </Card>
  );
};

export default SoftwareSelfService;
