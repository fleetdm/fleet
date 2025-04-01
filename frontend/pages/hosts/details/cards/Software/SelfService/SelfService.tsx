import React, { useCallback, useState, useContext, useMemo } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";
import { AxiosError } from "axios";

import { NotificationContext } from "context/notification";

import deviceApi, {
  IDeviceSoftwareQueryKey,
  IGetDeviceSoftwareResponse,
} from "services/entities/device_user";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { pluralize } from "utilities/strings/stringUtils";
import { getPathWithQueryParams } from "utilities/url";

import Card from "components/Card";
import CardHeader from "components/CardHeader";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import EmptyTable from "components/EmptyTable";
import Spinner from "components/Spinner";
import SearchField from "components/forms/fields/SearchField";
import Pagination from "components/Pagination";

import { parseHostSoftwareQueryParams } from "../HostSoftware";
import SelfServiceItem from "./SelfServiceItem";

const baseClass = "software-self-service";

// These default params are not subject to change by the user
const DEFAULT_SELF_SERVICE_QUERY_PARAMS = {
  per_page: 24, // Divisible by 2, 3, 4 so pagination renders well on responsive widths
  order_key: "name",
  order_direction: "asc",
  self_service: true,
} as const;

export interface ISoftwareSelfServiceProps {
  contactUrl: string;
  deviceToken: string;
  isSoftwareEnabled?: boolean;
  pathname: string;
  queryParams: ReturnType<typeof parseHostSoftwareQueryParams>;
  router: InjectedRouter;
}

const SoftwareSelfService = ({
  contactUrl,
  deviceToken,
  isSoftwareEnabled,
  pathname,
  queryParams,
  router,
}: ISoftwareSelfServiceProps) => {
  const { renderFlash } = useContext(NotificationContext);

  // State for controlling the self-service polling mechanism
  const [
    selfServiceRefetchStartTime,
    setSelfServiceRefetchStartTime,
  ] = useState<number | null>(null);

  // Memoize the query key
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
  >(
    queryKey,
    (context) => deviceApi.getDeviceSoftware(context.queryKey[0]), // Changed destructuring to context.queryKey
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: isSoftwareEnabled, // if software inventory is disabled, we don't bother fetching and always show the empty state
      keepPreviousData: true,
      staleTime: 7000,
      onSuccess: (response) => {
        // Check if any software is still installing (pending_install)
        const hasPendingInstalls = response.software.some(
          (software) => software.status === "pending_install"
        );

        if (hasPendingInstalls) {
          // If our timer wasn't already started
          if (!selfServiceRefetchStartTime) {
            setSelfServiceRefetchStartTime(Date.now());

            // Poll the API again using refetchSelfServiceSoftware.
            setTimeout(() => {
              refetchSelfServiceSoftware();
            }, 5000); // Poll every 5 seconds
          } else {
            // Check elapsed time
            const totalElapsedTime =
              Date.now() - (selfServiceRefetchStartTime || Date.now());
            if (totalElapsedTime < 120000) {
              // Continue polling if within the timeout
              setTimeout(() => {
                refetchSelfServiceSoftware();
              }, 5000); // Poll every 5 seconds
            } else {
              // Timeout reached
              renderFlash(
                "error",
                "Self-service software status check timed out. Please refresh the page."
              );
            }
          }
        }
      },
      onError: () => {
        renderFlash(
          "error",
          "We're having trouble fetching self-service software statuses. Please refresh the page."
        );
      },
    }
  );

  const onSearchQueryChange = (value: string) => {
    router.push(
      getPathWithQueryParams(pathname, {
        query: value,
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

  console.log("queryParams", queryParams);
  const onPrevPage = useCallback(() => {
    router.push(
      getPathWithQueryParams(pathname, {
        query: queryParams.query,
        page: queryParams.page - 1,
      })
    );
  }, [pathname, queryParams.page, router]);

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

  const renderSelfServiceCard = () => {
    const renderHeader = () => {
      const itemCount = data?.count || 0;

      return (
        <div className={`${baseClass}__header`}>
          <div className={`${baseClass}__items-count`}>
            {`${itemCount} ${pluralize(itemCount, "item")}`}
          </div>
          <div className={`${baseClass}__search`}>
            <SearchField
              placeholder="Search by name"
              onChange={onSearchQueryChange}
            />
          </div>
        </div>
      );
    };

    if (isLoading) {
      return (
        <>
          <Spinner />
        </>
      );
    }

    if (isError) {
      return <DataError />;
    }

    if (isEmpty || !data) {
      return (
        <>
          {renderHeader()}
          <EmptyTable
            graphicName="empty-software"
            header="No self-service software available yet"
            info="Your organization didn't add any self-service software. If you need any, reach out to your IT department."
          />
        </>
      );
    }

    if (isFetching) {
      return (
        <>
          {renderHeader()}
          <Spinner />
        </>
      );
    }

    if (isEmptySearch) {
      return (
        <>
          {renderHeader()}
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
        </>
      );
    }

    return (
      <>
        {renderHeader()}
        <div className={`${baseClass}__items`}>
          {data.software.map((s) => {
            let uuid =
              s.software_package?.last_install?.install_uuid ??
              s.app_store_app?.last_install?.command_uuid;
            if (!uuid) {
              uuid = "";
            }
            const key = `${s.id}${uuid}`;
            return (
              <SelfServiceItem
                key={key}
                deviceToken={deviceToken}
                software={s}
                onInstall={refetchSelfServiceSoftware}
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
