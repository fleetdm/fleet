import React, {
  useCallback,
  useState,
  useContext,
  useMemo,
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

import { generateSoftwareTableHeaders as generateDeviceSoftwareTableConfig } from "./SelfServiceTableConfig";
import { parseHostSoftwareQueryParams } from "../HostSoftware";
import { CATEGORIES_NAV_ITEMS, ICategory } from "./helpers";
import CategoriesMenu from "./CategoriesMenu";

const baseClass = "software-self-service";

// These default params are not subject to change by the user
const DEFAULT_SELF_SERVICE_QUERY_PARAMS = {
  per_page: 24, // Divisible by 2, 3, 4 so pagination renders well on responsive widths
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
  onShowInstallerDetails: (installUuid: string) => void;
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

  const [isPolling, setIsPolling] = useState(false); // Track polling state
  const [
    pollingTimeoutId,
    setPollingTimeoutId,
  ] = useState<NodeJS.Timeout | null>(null);

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
  const {
    data,
    isLoading,
    isError,
    isFetching,
    refetch: refetchSelfServiceSoftware,
  } = useQuery<
    IGetDeviceSoftwareResponse,
    AxiosError,
    IGetDeviceSoftwareResponse,
    IDeviceSoftwareQueryKey[]
  >(queryKey, (context) => deviceApi.getDeviceSoftware(context.queryKey[0]), {
    ...DEFAULT_USE_QUERY_OPTIONS,
    enabled: isSoftwareEnabled,
    keepPreviousData: true,
    staleTime: 7000,
  });

  // Poll for pending installs
  const { refetch: refetchForPendingInstalls } = useQuery<
    IGetDeviceSoftwareResponse,
    AxiosError
  >(
    ["pending_installs", queryKey[0]], // Include a unique key AND spread the original query key
    () => deviceApi.getDeviceSoftware(queryKey[0]), // Access the query key correctly
    {
      enabled: false,
      onSuccess: (response) => {
        const hasPendingInstalls = response.software.some(
          (software) => software.status === "pending_install"
        );

        if (hasPendingInstalls) {
          // Continue polling if pending installs exist
          const timeoutId = setTimeout(() => {
            refetchForPendingInstalls();
          }, 5000); // Poll every 5 seconds
          setPollingTimeoutId(timeoutId);
        } else {
          // Stop polling and refresh full data
          setIsPolling(false);
          refetchSelfServiceSoftware();
        }
      },
      onError: () => {
        setIsPolling(false);
        renderFlash(
          "error",
          "We're having trouble checking pending installs. Please refresh the page."
        );
      },
    }
  );

  const startPollingForPendingInstalls = useCallback(() => {
    if (!isPolling) {
      setIsPolling(true);
      refetchSelfServiceSoftware(); // Updates UI to show pending installs
      refetchForPendingInstalls(); // Starts polling for pending installs
    }
  }, [isPolling, refetchSelfServiceSoftware, refetchForPendingInstalls]);

  const stopPolling = () => {
    setIsPolling(false);
    if (pollingTimeoutId) {
      clearTimeout(pollingTimeoutId);
      setPollingTimeoutId(null);
    }
  };

  // Check if initially has pending installs, then start polling
  useEffect(() => {
    if (
      data?.software.some((software) => software.status === "pending_install")
    ) {
      startPollingForPendingInstalls();
    }
  }, [data, startPollingForPendingInstalls]);

  useEffect(() => {
    return () => stopPolling(); // Cleanup polling on unmount
  }, []);

  const onSearchQueryChange = (value: string) => {
    router.push(
      getPathWithQueryParams(pathname, {
        query: value,
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
        page: 0, // Always reset to page 0 when searching
      })
    );
  };

  const onNextPage = useCallback(() => {
    router.push(
      getPathWithQueryParams(pathname, {
        query: queryParams.query,
        page: queryParams.page + 1,
      })
    );
  }, [pathname, queryParams.page, queryParams.query, router]);

  const onPrevPage = useCallback(() => {
    router.push(
      getPathWithQueryParams(pathname, {
        query: queryParams.query,
        page: queryParams.page - 1,
      })
    );
  }, [pathname, queryParams.query, queryParams.page, router]);

  // TODO: handle empty state better, this is just a placeholder for now
  // TODO: what should happen if query params are invalid (e.g., page is negative or exceeds the
  // available results)?
  const isEmpty =
    !data?.software.length &&
    !data?.meta.has_previous_results &&
    queryParams.query === "";
  const isEmptySearch =
    !data?.software.length &&
    !data?.meta.has_previous_results &&
    queryParams.query !== "";

  const tableConfig = useMemo(() => {
    return generateDeviceSoftwareTableConfig({
      deviceToken,
      onInstall: startPollingForPendingInstalls,
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

    if (isEmpty || !data) {
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
            data={data?.software || []}
            isLoading={isLoading}
            defaultSortHeader={DEFAULT_SELF_SERVICE_QUERY_PARAMS.order_key}
            defaultSortDirection={
              DEFAULT_SELF_SERVICE_QUERY_PARAMS.order_direction
            }
            pageIndex={0}
            disableNextPage={data?.meta.has_next_results === false}
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
      {renderSelfServiceCard()}
    </Card>
  );
};

export default SoftwareSelfService;
