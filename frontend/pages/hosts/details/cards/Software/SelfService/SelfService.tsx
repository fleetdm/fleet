import React, { useCallback } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";
import { AxiosError } from "axios";

import deviceApi, {
  IDeviceSoftwareQueryKey,
  IGetDeviceSoftwareResponse,
} from "services/entities/device_user";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { pluralize } from "utilities/strings/stringUtils";

import Card from "components/Card";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import EmptyTable from "components/EmptyTable";
import Spinner from "components/Spinner";

import Pagination from "pages/ManageControlsPage/components/Pagination";

import { parseHostSoftwareQueryParams } from "../HostSoftware";
import SelfServiceItem from "./SelfServiceItem";

const baseClass = "software-self-service";

// These default params are not subject to change by the user
const DEFAULT_SELF_SERVICE_QUERY_PARAMS = {
  per_page: 9,
  order_key: "name",
  order_direction: "asc",
  query: "",
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
  const { data, isLoading, isError, refetch } = useQuery<
    IGetDeviceSoftwareResponse,
    AxiosError,
    IGetDeviceSoftwareResponse,
    IDeviceSoftwareQueryKey[]
  >(
    [
      {
        scope: "device_software",
        id: deviceToken,
        page: queryParams.page,
        ...DEFAULT_SELF_SERVICE_QUERY_PARAMS,
      },
    ],
    ({ queryKey }) => deviceApi.getDeviceSoftware(queryKey[0]),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: isSoftwareEnabled, // if software inventory is disabled, we don't bother fetching and always show the empty state
      keepPreviousData: true,
      staleTime: 7000,
    }
  );

  const onNextPage = useCallback(() => {
    router.push(pathname.concat(`?page=${queryParams.page + 1}`));
  }, [pathname, queryParams.page, router]);

  const onPrevPage = useCallback(() => {
    router.push(pathname.concat(`?page=${queryParams.page - 1}`));
  }, [pathname, queryParams.page, router]);

  // TODO: handle empty state better, this is just a placeholder for now
  // TODO: what should happen if query params are invalid (e.g., page is negative or exceeds the
  // available results)?
  const isEmpty = !data?.software.length && !data?.meta.has_previous_results;

  return (
    <Card
      borderRadiusSize="xxlarge"
      includeShadow
      paddingSize="xxlarge"
      className={baseClass}
    >
      <div className={`${baseClass}__card-header`}>Self-service</div>
      <div className={`${baseClass}__card-subheader`}>
        Install organization-approved apps provided by your IT department.{" "}
        {contactUrl && (
          <span>
            If you need help,{" "}
            <CustomLink url={contactUrl} text="reach out to IT" newTab />
          </span>
        )}
      </div>
      {isLoading ? (
        <Spinner />
      ) : (
        <>
          {isError && <DataError />}
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
    </Card>
  );
};

export default SoftwareSelfService;
