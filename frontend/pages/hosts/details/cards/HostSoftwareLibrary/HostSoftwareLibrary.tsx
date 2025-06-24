import React, {
  useCallback,
  useContext,
  useMemo,
  useRef,
  useState,
  useEffect,
} from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";
import { AxiosError } from "axios";

import hostAPI, {
  IGetHostSoftwareResponse,
  IHostSoftwareQueryKey,
} from "services/entities/hosts";
import { IHostSoftware, ISoftware } from "interfaces/software";
import { HostPlatform, isAndroid } from "interfaces/platform";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";

import CardHeader from "components/CardHeader";
import DataError from "components/DataError";
import Spinner from "components/Spinner";

import { generateHostSWLibraryTableHeaders } from "./HostSoftwareLibraryTable/HostSoftwareLibraryTableConfig";
import HostSoftwareLibraryTable from "./HostSoftwareLibraryTable";
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
  router: InjectedRouter;
  queryParams: ReturnType<typeof parseHostSoftwareLibraryQueryParams>;
  pathname: string;
  hostTeamId: number;
  onShowSoftwareDetails: (software?: IHostSoftware) => void;
  isSoftwareEnabled?: boolean;
  hostScriptsEnabled?: boolean;
  hostMDMEnrolled?: boolean;
  isHostOnline?: boolean;
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
  self_service?: string;
}) => {
  const searchQuery = queryParams?.query ?? DEFAULT_SEARCH_QUERY;
  const sortHeader = queryParams?.order_key ?? DEFAULT_SORT_HEADER;
  const sortDirection = queryParams?.order_direction ?? DEFAULT_SORT_DIRECTION;
  const page = queryParams?.page
    ? parseInt(queryParams.page, 10)
    : DEFAULT_PAGE;
  const pageSize = DEFAULT_PAGE_SIZE;
  const selfService = queryParams?.self_service === "true";

  return {
    page,
    query: searchQuery,
    order_key: sortHeader,
    order_direction: sortDirection,
    per_page: pageSize,
    available_for_install: true, // always true for host installers
    self_service: selfService,
  };
};

const HostSoftwareLibrary = ({
  id,
  platform,
  softwareUpdatedAt,
  hostScriptsEnabled,
  router,
  queryParams,
  pathname,
  hostTeamId = 0,
  onShowSoftwareDetails,
  isSoftwareEnabled = false,
  hostMDMEnrolled,
  isHostOnline = false,
}: IHostInstallersProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const {
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamAdmin,
    isTeamMaintainer,
  } = useContext(AppContext);

  const isUnsupported = isAndroid(platform); // no Android software

  const [hostSoftwareLibraryRes, setHostSoftwareLibraryRes] = useState<
    IGetHostSoftwareResponse | undefined
  >(undefined);

  const pendingSoftwareSetRef = useRef<Set<string>>(new Set()); // Track for polling
  const pollingTimeoutIdRef = useRef<NodeJS.Timeout | null>(null);

  const queryKey = useMemo<IHostSoftwareQueryKey[]>(() => {
    return [
      {
        scope: "host_software",
        id: id as number,
        softwareUpdatedAt,
        ...queryParams,
      },
    ];
  }, [queryParams, id, softwareUpdatedAt]);

  const {
    isLoading: hostSoftwareLibraryLoading,
    isError: hostSoftwareLibraryError,
    isFetching: hostSoftwareLibraryFetching,
  } = useQuery<
    IGetHostSoftwareResponse,
    AxiosError,
    IGetHostSoftwareResponse,
    IHostSoftwareQueryKey[]
  >(queryKey, () => hostAPI.getHostSoftware(queryKey[0]), {
    ...DEFAULT_USE_QUERY_OPTIONS,
    enabled: isSoftwareEnabled && !isUnsupported,
    keepPreviousData: true,
    onSuccess: (response) => {
      setHostSoftwareLibraryRes(response);
    },
  });

  // Poll for pending installs/uninstalls
  const { refetch: refetchForPendingInstallsOrUninstalls } = useQuery<
    IGetHostSoftwareResponse,
    AxiosError
  >(
    ["pending_installs", queryKey[0]],
    () => hostAPI.getHostSoftware(queryKey[0]),
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
          [...newPendingSet].every((pendingId) =>
            pendingSoftwareSetRef.current.has(pendingId)
          );

        if (newPendingSet.size > 0) {
          // If the set changed, update and continue polling
          if (!setsAreEqual) {
            pendingSoftwareSetRef.current = newPendingSet;
            setHostSoftwareLibraryRes(response);
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
          setHostSoftwareLibraryRes(response);
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

  // Stop polling if the host goes offline
  // Polling will automatically resume when host is online with pending installs
  useEffect(() => {
    if (!isHostOnline) {
      if (pollingTimeoutIdRef.current) {
        clearTimeout(pollingTimeoutIdRef.current);
        pollingTimeoutIdRef.current = null;
      }
      pendingSoftwareSetRef.current = new Set();
    }
  }, [isHostOnline]);

  const startPollingForPendingInstallsOrUninstalls = useCallback(
    (pendingIds: string[]) => {
      if (isHostOnline) {
        const newSet = new Set(pendingIds);
        const setsAreEqual =
          newSet.size === pendingSoftwareSetRef.current.size &&
          [...newSet].every((pendingId) =>
            pendingSoftwareSetRef.current.has(pendingId)
          );
        if (!setsAreEqual) {
          pendingSoftwareSetRef.current = newSet;

          // Clear any existing timeout to avoid overlap
          if (pollingTimeoutIdRef.current) {
            clearTimeout(pollingTimeoutIdRef.current);
          }
          // Starts polling for pending installs/uninstalls
          refetchForPendingInstallsOrUninstalls();
        }
      }
    },
    [refetchForPendingInstallsOrUninstalls, isHostOnline]
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
    const pendingSoftware = hostSoftwareLibraryRes?.software.filter(
      (software) =>
        software.status === "pending_install" ||
        software.status === "pending_uninstall"
    );
    const pendingIds = pendingSoftware?.map((s) => String(s.id)) ?? [];
    if (pendingIds.length > 0) {
      startPollingForPendingInstallsOrUninstalls(pendingIds);
    }
  }, [hostSoftwareLibraryRes, startPollingForPendingInstallsOrUninstalls]);

  const onInstallOrUninstall = useCallback(() => {
    refetchForPendingInstallsOrUninstalls();
  }, [refetchForPendingInstallsOrUninstalls]);

  const userHasSWWritePermission = Boolean(
    isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer
  );

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
        await hostAPI.installHostSoftwarePackage(id as number, softwareId);
        if (isMountedRef.current) {
          onInstallOrUninstall();
        }
        renderFlash(
          "success",
          <>
            Software{" "}
            {isHostOnline
              ? "is installing"
              : "will install when the host comes online"}
            . To see details, go to <b>Details &gt; Activity</b>.
          </>
        );
      } catch (e) {
        renderFlash("error", getInstallErrorMessage(e));
      }
    },
    [id, renderFlash, onInstallOrUninstall, isHostOnline]
  );

  const onClickUninstallAction = useCallback(
    async (softwareId: number) => {
      try {
        await hostAPI.uninstallHostSoftwarePackage(id as number, softwareId);
        if (isMountedRef.current) {
          onInstallOrUninstall();
        }
        renderFlash(
          "success",
          <>
            Software{" "}
            {isHostOnline
              ? "is uninstalling"
              : "will uninstall when the host comes online"}
            . To see details, go to <b>Details &gt; Activity</b>.
          </>
        );
      } catch (e) {
        renderFlash("error", getUninstallErrorMessage(e));
      }
    },
    [id, renderFlash, onInstallOrUninstall, isHostOnline]
  );

  const tableConfig = useMemo(() => {
    return generateHostSWLibraryTableHeaders({
      userHasSWWritePermission,
      hostScriptsEnabled,
      hostMDMEnrolled,
      router,
      teamId: hostTeamId,
      baseClass,
      onShowSoftwareDetails,
      onClickInstallAction,
      onClickUninstallAction,
      isHostOnline,
    });
  }, [
    router,
    userHasSWWritePermission,
    hostScriptsEnabled,
    hostTeamId,
    hostMDMEnrolled,
    onShowSoftwareDetails,
    onClickInstallAction,
    onClickUninstallAction,
    isHostOnline,
  ]);

  const isLoading = hostSoftwareLibraryLoading;
  const isError = hostSoftwareLibraryError;
  const data = hostSoftwareLibraryRes;

  const renderHostSoftware = () => {
    if (isLoading) {
      return <Spinner />;
    }

    if (isError) {
      return <DataError verticalPaddingSize="pad-xxxlarge" />;
    }

    return (
      <HostSoftwareLibraryTable
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
        selfService={queryParams.self_service}
      />
    );
  };

  return (
    <div className={baseClass}>
      <CardHeader subheader="Software available to install on this host." />
      {renderHostSoftware()}
    </div>
  );
};

export default React.memo(HostSoftwareLibrary);
