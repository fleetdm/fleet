import React, { useCallback, useEffect, useRef } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";
import ReactTooltip from "react-tooltip";
import { AxiosError } from "axios";
import { uniqueId } from "lodash";

import { IHostSoftware, SoftwareInstallStatus } from "interfaces/software";
import deviceApi, {
  IDeviceSoftwareQueryKey,
  IGetDeviceSoftwareResponse,
} from "services/entities/device_user";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import { dateAgo } from "utilities/date_format";

import Button from "components/buttons/Button";
import Card from "components/Card";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import EmptyTable from "components/EmptyTable";
import Icon from "components/Icon";
import Spinner from "components/Spinner";

import Pagination from "pages/ManageControlsPage/components/Pagination";
import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import { IStatusDisplayConfig } from "../InstallStatusCell/InstallStatusCell";
import { parseHostSoftwareQueryParams } from "../Software";

const baseClass = "software-self-service";

// These default params are not subject to change by the user
const DEFAULT_SELF_SERVICE_QUERY_PARAMS = {
  per_page: 9, // TODO: confirm page size; dev note says 9 but design depicts 6
  order_key: "name",
  order_direction: "asc",
  query: "",
  self_service: true,
} as const;

const STATUS_CONFIG: Record<SoftwareInstallStatus, IStatusDisplayConfig> = {
  installed: {
    iconName: "success",
    displayText: "Installed",
    tooltip: ({ lastInstalledAt }) => (
      <>
        Software installed successfully ({dateAgo(lastInstalledAt as string)}).
      </>
    ),
  },
  pending: {
    iconName: "pending-outline",
    displayText: "Install in progress...",
    tooltip: () => "Software installation in progress...",
  },
  failed: {
    iconName: "error",
    displayText: "Failed",
    tooltip: ({ lastInstalledAt = "" }) => (
      <>
        Software failed to install
        {lastInstalledAt ? `(${dateAgo(lastInstalledAt)})` : ""}. Select{" "}
        <b>Retry</b> to install again, or contact your IT department.
      </>
    ),
  },
};

const InstallerStatus = ({
  id,
  status,
  last_install,
}: Pick<IHostSoftware, "id" | "status" | "last_install">) => {
  const displayConfig = STATUS_CONFIG[status as keyof typeof STATUS_CONFIG];
  if (!displayConfig) {
    // API should ensure this never happens, but just in case
    return null;
  }

  return (
    <div className={`${baseClass}__status-content`}>
      <div
        className={`${baseClass}__status-with-tooltip`}
        data-tip
        data-for={`install-tooltip__${id}`}
      >
        <Icon name={displayConfig.iconName} />
        <span>{displayConfig.displayText}</span>
      </div>
      <ReactTooltip
        className={`${baseClass}__status-tooltip`}
        effect="solid"
        backgroundColor="#3e4771"
        id={`install-tooltip__${id}`}
        data-html
      >
        <span className={`${baseClass}__status-tooltip-text`}>
          {displayConfig.tooltip({
            lastInstalledAt: last_install?.installed_at,
          })}
        </span>
      </ReactTooltip>
    </div>
  );
};

const InstallerStatusAction = ({
  deviceToken,
  software: { id, status, last_install },
  onInstall,
}: {
  deviceToken: string;
  software: IHostSoftware;
  onInstall: () => void;
}) => {
  // localStatus is used to track the status of the any user-initiated install action
  const [localStatus, setLocalStatus] = React.useState<
    SoftwareInstallStatus | undefined
  >(undefined);

  // displayStatus allows us to display the localStatus (if any) or the status from the list
  // software reponse
  const displayStatus = localStatus || status;

  // if the localStatus is "failed", we don't our tooltip to include the old installed_at date so we
  // set this to null, which tells the tooltip to omit the parenthetical date
  const lastInstall = localStatus === "failed" ? null : last_install;

  const isMountedRef = useRef(false);
  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  const onClick = useCallback(async () => {
    setLocalStatus("pending");
    try {
      // TODO: confirm specs for response handling
      const resp = await deviceApi.installSelfServiceSoftware(deviceToken, id);
      console.log("resp", resp);
      if (isMountedRef.current) {
        console.log("Component is mounted, refetching data...");
        onInstall();
      } else {
        console.log("Component is unmounted, skipping refetch...");
      }
    } catch (error) {
      // TODO: confirm specs for error handling
      console.log("error", error);
      if (isMountedRef.current) {
        setLocalStatus("failed");
      }
    } finally {
      // TODO: anything else to do here? maybe something subject to isMountedRef.current check?
      console.log("finally");
    }
  }, [deviceToken, id, onInstall]);

  return (
    <div className={`${baseClass}__item-status-action`}>
      <div className={`${baseClass}__item-status`}>
        <InstallerStatus
          id={id}
          status={displayStatus}
          last_install={lastInstall}
        />
      </div>
      <div className={`${baseClass}__item-action`}>
        {(displayStatus === "failed" || displayStatus === null) && (
          <Button
            variant="text-icon"
            type="button"
            className={`${baseClass}__item-action-button${
              localStatus === "pending" ? "--installing" : ""
            }`}
            onClick={onClick}
          >
            {displayStatus === "failed" ? "Retry" : "Install"}
          </Button>
        )}
      </div>
    </div>
  );
};

const InstallerInfo = ({ software }: { software: IHostSoftware }) => {
  const { name, source, package_available_for_install } = software;
  // TODO: version is missing from the API response
  const version = package_available_for_install || "Something is missing :(";
  return (
    <div className={`${baseClass}__item-topline`}>
      <div className={`${baseClass}__item-icon`}>
        <SoftwareIcon name={name} source={source} size="medium_large" />
      </div>
      <div className={`${baseClass}__item-name-version`}>
        <div className={`${baseClass}__item-name`}>{name}</div>
        <div className={`${baseClass}__item-version`}>{version}</div>
      </div>
    </div>
  );
};

const SoftwareSelfService = ({
  contactUrl,
  deviceToken,
  isSoftwareEnabled,
  pathname,
  queryParams,
  router,
}: {
  contactUrl: string; // TODO: confirm this has been added to the device API response
  deviceToken: string;
  isSoftwareEnabled?: boolean;
  pathname: string;
  queryParams: ReturnType<typeof parseHostSoftwareQueryParams>;
  router: InjectedRouter;
}) => {
  const { data, isLoading, isError, isFetching, refetch } = useQuery<
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
    ({ queryKey }) =>
      deviceApi.getDeviceSoftware(queryKey[0]).then((res) => {
        // TODO: remove `.then`, just using it to simulate that the data is changing when we refetch
        const newSoftware = res.software.map((s) => {
          if (s.last_install) {
            const newInstall = {
              ...s.last_install,
              install_uuid: uniqueId(),
            };
            return { ...s, last_install: newInstall };
          }
          return s;
        });
        return { ...res, software: newSoftware };
      }),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: isSoftwareEnabled,
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

  // TODO: truncate name and version with tooltip
  return (
    <Card
      borderRadiusSize="large"
      includeShadow
      largePadding
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
                    <b>{data.count} items</b>
                  </div>
                  <div className={`${baseClass}__items`}>
                    {data.software.map((s) => {
                      const key = `${s.id}${s.last_install?.install_uuid}`; // concatenating install_uuid so item updates with fresh data on refetch
                      return (
                        <div key={key} className={`${baseClass}__item`}>
                          <div className={`${baseClass}__item-content`}>
                            <InstallerInfo software={s} />
                            <InstallerStatusAction
                              deviceToken={deviceToken}
                              software={s}
                              onInstall={refetch}
                            />
                          </div>
                        </div>
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
