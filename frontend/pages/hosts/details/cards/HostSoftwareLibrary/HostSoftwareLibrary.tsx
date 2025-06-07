import React, { useCallback, useContext, useMemo, useState } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";
import { AxiosError } from "axios";

import hostAPI, {
  IGetHostSoftwareResponse,
  IHostSoftwareQueryKey,
} from "services/entities/hosts";
import { IHostSoftware, ISoftware } from "interfaces/software";
import { HostPlatform, isAndroid, isIPadOrIPhone } from "interfaces/platform";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";

import CardHeader from "components/CardHeader";
import DataError from "components/DataError";
import Spinner from "components/Spinner";

import { generateHostSWLibraryTableHeaders as generateHostInstallersTableConfig } from "./HostSoftwareLibraryTableConfig";
import HostInstallersTable from "./HostInstallersTable";
import { getInstallErrorMessage, getUninstallErrorMessage } from "./helpers";

const baseClass = "host-software-library-card";

export interface ITableSoftware extends Omit<ISoftware, "vulnerabilities"> {
  vulnerabilities: string[]; // for client-side search purposes, we only want an array of cve strings
}

interface IHostInstallersProps {
  /** This is the host id or the device token */
  id: number | string;
  platform: HostPlatform;
  softwareUpdatedAt?: string;
  hostCanWriteSoftware: boolean;
  router: InjectedRouter;
  queryParams: ReturnType<typeof parseHostSoftwareLibraryQueryParams>;
  pathname: string;
  hostTeamId: number;
  onShowSoftwareDetails: (software: IHostSoftware) => void;
  isSoftwareEnabled?: boolean;
  hostScriptsEnabled?: boolean;
  hostMDMEnrolled?: boolean;
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
  vulnerable?: string;
  exploit?: string;
  min_cvss_score?: string;
  max_cvss_score?: string;
  category_id?: string;
}) => {
  const searchQuery = queryParams?.query ?? DEFAULT_SEARCH_QUERY;
  const sortHeader = queryParams?.order_key ?? DEFAULT_SORT_HEADER;
  const sortDirection = queryParams?.order_direction ?? DEFAULT_SORT_DIRECTION;
  const page = queryParams?.page
    ? parseInt(queryParams.page, 10)
    : DEFAULT_PAGE;
  const pageSize = DEFAULT_PAGE_SIZE;
  const categoryId = queryParams?.category_id
    ? parseInt(queryParams.category_id, 10)
    : undefined;

  return {
    page,
    query: searchQuery,
    order_key: sortHeader,
    order_direction: sortDirection,
    per_page: pageSize,
    available_for_install: true, // always true for host installers
    category_id: categoryId,
  };
};

const HostSoftwareLibrary = ({
  id,
  platform,
  softwareUpdatedAt,
  hostCanWriteSoftware,
  hostScriptsEnabled,
  router,
  queryParams,
  pathname,
  hostTeamId = 0,
  onShowSoftwareDetails,
  isSoftwareEnabled = false,
  hostMDMEnrolled,
}: IHostInstallersProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const {
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamAdmin,
    isTeamMaintainer,
  } = useContext(AppContext);

  const isUnsupported = isAndroid(platform); // no Android software

  // disables install/uninstall actions after click
  const [softwareIdActionPending, setSoftwareIdActionPending] = useState<
    number | null
  >(null);

  const {
    data: hostSoftwareLibraryRes,
    isLoading: hostSoftwareLibraryLoading,
    isError: hostSoftwareLibraryError,
    isFetching: hostSoftwareLibraryFetching,
    refetch: refetchHostSoftware,
  } = useQuery<
    IGetHostSoftwareResponse,
    AxiosError,
    IGetHostSoftwareResponse,
    IHostSoftwareQueryKey[]
  >(
    [
      {
        scope: "host_software",
        id: id as number,
        softwareUpdatedAt,
        ...queryParams,
      },
    ],
    ({ queryKey }) => {
      return hostAPI.getHostSoftware(queryKey[0]);
    },
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: isSoftwareEnabled && !isUnsupported,
      keepPreviousData: true,
      staleTime: 7000,
    }
  );

  const refetchSoftware = useMemo(() => refetchHostSoftware, [
    refetchHostSoftware,
  ]);

  const userHasSWWritePermission = Boolean(
    isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer
  );

  const installHostSoftwarePackage = useCallback(
    async (softwareId: number) => {
      setSoftwareIdActionPending(softwareId);
      try {
        await hostAPI.installHostSoftwarePackage(id as number, softwareId);
        renderFlash(
          "success",
          "Software is installing or will install when the host comes online."
        );
      } catch (e) {
        renderFlash("error", getInstallErrorMessage(e));
      }
      setSoftwareIdActionPending(null);
      refetchSoftware();
    },
    [id, renderFlash, refetchSoftware]
  );

  const uninstallHostSoftwarePackage = useCallback(
    async (softwareId: number) => {
      setSoftwareIdActionPending(softwareId);
      try {
        await hostAPI.uninstallHostSoftwarePackage(id as number, softwareId);
        renderFlash(
          "success",
          <>
            Software is uninstalling or will uninstall when the host comes
            online. To see details, go to <b>Details &gt; Activity</b>.
          </>
        );
      } catch (e) {
        renderFlash("error", getUninstallErrorMessage(e));
      }
      setSoftwareIdActionPending(null);
      refetchSoftware();
    },
    [id, renderFlash, refetchSoftware]
  );

  const tableConfig = useMemo(() => {
    return generateHostInstallersTableConfig({
      userHasSWWritePermission,
      hostScriptsEnabled,
      hostCanWriteSoftware,
      hostMDMEnrolled,
      softwareIdActionPending,
      router,
      teamId: hostTeamId,
      baseClass,
    });
  }, [
    router,
    softwareIdActionPending,
    userHasSWWritePermission,
    hostScriptsEnabled,
    hostTeamId,
    hostCanWriteSoftware,
    hostMDMEnrolled,
  ]);

  const isLoading = hostSoftwareLibraryLoading;

  const isError = hostSoftwareLibraryError;

  const data = hostSoftwareLibraryRes;

  const renderHostSoftware = () => {
    if (isLoading) {
      return <Spinner />;
    }
    // will never be the case - to handle `platform` typing discrepancy with DeviceUserPage
    if (!platform) {
      return null;
    }
    return (
      <>
        {isError && <DataError verticalPaddingSize="pad-xxxlarge" />}
        {!isError && (
          <HostInstallersTable
            isLoading={hostSoftwareLibraryFetching}
            data={data}
            platform={platform}
            router={router}
            tableConfig={tableConfig}
            sortHeader={queryParams.order_key}
            sortDirection={queryParams.order_direction}
            searchQuery={queryParams.query}
            page={queryParams.page}
            pagePath={pathname}
          />
        )}
      </>
    );
  };

  return (
    <div className={baseClass}>
      <CardHeader subheader="Software available to install on this device." />
      {renderHostSoftware()}
    </div>
  );
};

export default React.memo(HostSoftwareLibrary);
