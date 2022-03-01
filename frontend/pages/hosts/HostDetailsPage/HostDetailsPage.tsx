import React, { useContext, useState, useCallback, useEffect } from "react";
import { useDispatch } from "react-redux";
import { Link } from "react-router";
import { Params, InjectedRouter } from "react-router/lib/Router";
import { useQuery } from "react-query";
import { useErrorHandler } from "react-error-boundary";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";

import classnames from "classnames";
import { isEmpty, pick, reduce } from "lodash";
// @ts-ignore
import { stringToClipboard } from "utilities/copy_text";

import PATHS from "router/paths";
import hostAPI from "services/entities/hosts";
import queryAPI from "services/entities/queries";
import teamAPI from "services/entities/teams";
import { AppContext } from "context/app";
import { PolicyContext } from "context/policy";
import {
  IHost,
  IDeviceMappingResponse,
  IMacadminsResponse,
  IPackStats,
} from "interfaces/host";
import { IQueryStats } from "interfaces/query_stats";
import { ISoftware } from "interfaces/software";
import { IHostPolicy } from "interfaces/policy";
import { ILabel } from "interfaces/label";
import { ITeam } from "interfaces/team";
import { IQuery } from "interfaces/query";
import { IUser } from "interfaces/user";
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";
import permissionUtils from "utilities/permissions";

import ReactTooltip from "react-tooltip";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Spinner from "components/Spinner";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import TableContainer from "components/TableContainer";
import TabsWrapper from "components/TabsWrapper";
import InfoBanner from "components/InfoBanner";
import TooltipWrapper from "components/TooltipWrapper";
import {
  Accordion,
  AccordionItem,
  AccordionItemHeading,
  AccordionItemButton,
  AccordionItemPanel,
} from "react-accessible-accordion";
import {
  humanHostUptime,
  humanHostLastSeen,
  humanHostEnrolled,
  humanHostMemory,
  humanHostDetailUpdated,
  secondsToHms,
} from "fleet/helpers";

import SoftwareTab from "./SoftwareTab/SoftwareTab";
// @ts-ignore
import SelectQueryModal from "./SelectQueryModal";
import TransferHostModal from "./TransferHostModal";
import PolicyDetailsModal from "./HostPoliciesTable/PolicyDetailsModal";
import {
  generatePolicyTableHeaders,
  generatePolicyDataSet,
} from "./HostPoliciesTable/HostPoliciesTableConfig";
import generateUsersTableHeaders from "./UsersTable/UsersTableConfig";
import {
  generatePackTableHeaders,
  generatePackDataSet,
} from "./PackTable/PackTableConfig";
import EmptyUsers from "./EmptyUsers";
import PolicyFailingCount from "./HostPoliciesTable/PolicyFailingCount";
import { isValidPolicyResponse } from "../ManageHostsPage/helpers";

import BackChevron from "../../../../assets/images/icon-chevron-down-9x6@2x.png";
import CopyIcon from "../../../../assets/images/icon-copy-clipboard-fleet-blue-20x20@2x.png";
import DeleteIcon from "../../../../assets/images/icon-action-delete-14x14@2x.png";
import IssueIcon from "../../../../assets/images/icon-issue-fleet-black-50-16x16@2x.png";
import QueryIcon from "../../../../assets/images/icon-action-query-16x16@2x.png";
import TransferIcon from "../../../../assets/images/icon-action-transfer-16x16@2x.png";

const baseClass = "host-details";

interface IHostDetailsProps {
  router: InjectedRouter; // v3
  params: Params;
}

interface IFleetQueriesResponse {
  queries: IQuery[];
}

interface ITeamsResponse {
  teams: ITeam[];
}

interface IHostResponse {
  host: IHost;
}
interface ISearchQueryData {
  searchQuery: string;
  sortHeader: string;
  sortDirection: string;
  pageSize: number;
  pageIndex: number;
}

const TAGGED_TEMPLATES = {
  queryByHostRoute: (hostId: number | undefined | null) => {
    return `${hostId ? `?host_ids=${hostId}` : ""}`;
  },
};

const HostDetailsPage = ({
  router,
  params: { host_id },
}: IHostDetailsProps): JSX.Element => {
  const hostIdFromURL = parseInt(host_id, 10);
  const dispatch = useDispatch();
  const {
    isGlobalAdmin,
    isPremiumTier,
    isOnlyObserver,
    isGlobalMaintainer,
    currentUser,
  } = useContext(AppContext);
  const {
    setLastEditedQueryName,
    setLastEditedQueryDescription,
    setLastEditedQueryBody,
    setLastEditedQueryResolution,
    setPolicyTeamId,
  } = useContext(PolicyContext);
  const handlePageError = useErrorHandler();
  const canTransferTeam =
    isPremiumTier && (isGlobalAdmin || isGlobalMaintainer);

  const canDeleteHost = (user: IUser, host: IHost) => {
    if (
      isGlobalAdmin ||
      isGlobalMaintainer ||
      permissionUtils.isTeamAdmin(user, host.team_id) ||
      permissionUtils.isTeamMaintainer(user, host.team_id)
    ) {
      return true;
    }
    return false;
  };

  const [showDeleteHostModal, setShowDeleteHostModal] = useState<boolean>(
    false
  );
  const [showTransferHostModal, setShowTransferHostModal] = useState<boolean>(
    false
  );
  const [showQueryHostModal, setShowQueryHostModal] = useState<boolean>(false);
  const [showPolicyDetailsModal, setPolicyDetailsModal] = useState<boolean>(
    false
  );
  const [showOSPolicyModal, setShowOSPolicyModal] = useState<boolean>(false);
  const [selectedPolicy, setSelectedPolicy] = useState<IHostPolicy | null>(
    null
  );

  const [refetchStartTime, setRefetchStartTime] = useState<number | null>(null);
  const [showRefetchSpinner, setShowRefetchSpinner] = useState<boolean>(false);
  const [packsState, setPacksState] = useState<IPackStats[]>();
  const [scheduleState, setScheduleState] = useState<IQueryStats[]>();
  const [hostSoftware, setHostSoftware] = useState<ISoftware[]>([]);
  const [usersState, setUsersState] = useState<{ username: string }[]>([]);
  const [usersSearchString, setUsersSearchString] = useState<string>("");
  const [copyMessage, setCopyMessage] = useState<string>("");

  const { data: fleetQueries, error: fleetQueriesError } = useQuery<
    IFleetQueriesResponse,
    Error,
    IQuery[]
  >("fleet queries", () => queryAPI.loadAll(), {
    enabled: !!hostIdFromURL,
    refetchOnMount: false,
    refetchOnReconnect: false,
    refetchOnWindowFocus: false,
    retry: false,
    select: (data: IFleetQueriesResponse) => data.queries,
  });

  const { data: teams } = useQuery<ITeamsResponse, Error, ITeam[]>(
    "teams",
    () => teamAPI.loadAll(),
    {
      enabled: !!hostIdFromURL && !!isPremiumTier,
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
      retry: false,
      select: (data: ITeamsResponse) => data.teams,
    }
  );

  const { data: deviceMapping, refetch: refetchDeviceMapping } = useQuery(
    ["deviceMapping", hostIdFromURL],
    () => hostAPI.loadHostDetailsExtension(hostIdFromURL, "device_mapping"),
    {
      enabled: !!hostIdFromURL,
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
      retry: false,
      select: (data: IDeviceMappingResponse) => data.device_mapping,
    }
  );

  const { data: macadmins, refetch: refetchMacadmins } = useQuery(
    ["macadmins", hostIdFromURL],
    () => hostAPI.loadHostDetailsExtension(hostIdFromURL, "macadmins"),
    {
      enabled: !!hostIdFromURL,
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
      retry: false,
      select: (data: IMacadminsResponse) => data.macadmins,
    }
  );

  const refetchExtensions = () => {
    deviceMapping !== null && refetchDeviceMapping();
    macadmins !== null && refetchMacadmins();
  };

  const {
    isLoading: isLoadingHost,
    data: host,
    refetch: refetchHostDetails,
  } = useQuery<IHostResponse, Error, IHost>(
    ["host", hostIdFromURL],
    () => hostAPI.loadHostDetails(hostIdFromURL),
    {
      enabled: !!hostIdFromURL,
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
      retry: false,
      select: (data: IHostResponse) => data.host,
      onSuccess: (returnedHost) => {
        setShowRefetchSpinner(returnedHost.refetch_requested);
        if (returnedHost.refetch_requested) {
          // If the API reports that a Fleet refetch request is pending, we want to check back for fresh
          // host details. Here we set a one second timeout and poll the API again using
          // fullyReloadHost. We will repeat this process with each onSuccess cycle for a total of
          // 60 seconds or until the API reports that the Fleet refetch request has been resolved
          // or that the host has gone offline.
          if (!refetchStartTime) {
            // If our 60 second timer wasn't already started (e.g., if a refetch was pending when
            // the first page loads), we start it now if the host is online. If the host is offline,
            // we skip the refetch on page load.
            if (returnedHost.status === "online") {
              setRefetchStartTime(Date.now());
              setTimeout(() => {
                refetchHostDetails();
                refetchExtensions();
              }, 1000);
            } else {
              setShowRefetchSpinner(false);
            }
          } else {
            const totalElapsedTime = Date.now() - refetchStartTime;
            if (totalElapsedTime < 60000) {
              if (returnedHost.status === "online") {
                setTimeout(() => {
                  refetchHostDetails();
                  refetchExtensions();
                }, 1000);
              } else {
                dispatch(
                  renderFlash(
                    "error",
                    `This host is offline. Please try refetching host vitals later.`
                  )
                );
                setShowRefetchSpinner(false);
              }
            } else {
              dispatch(
                renderFlash(
                  "error",
                  `We're having trouble fetching fresh vitals for this host. Please try again later.`
                )
              );
              setShowRefetchSpinner(false);
            }
          }
          return; // exit early because refectch is pending so we can avoid unecessary steps below
        }
        setHostSoftware(returnedHost.software);
        setUsersState(returnedHost.users);
        if (returnedHost.pack_stats) {
          const packStatsByType = returnedHost.pack_stats.reduce(
            (
              dictionary: {
                packs: IPackStats[];
                schedule: IQueryStats[];
              },
              pack: IPackStats
            ) => {
              if (pack.type === "pack") {
                dictionary.packs.push(pack);
              } else {
                dictionary.schedule.push(...pack.query_stats);
              }
              return dictionary;
            },
            { packs: [], schedule: [] }
          );
          setPacksState(packStatsByType.packs);
          setScheduleState(packStatsByType.schedule);
        }
      },
      onError: (error) => handlePageError(error),
    }
  );

  useEffect(() => {
    setUsersState(() => {
      return (
        host?.users.filter((user) => {
          return user.username
            .toLowerCase()
            .includes(usersSearchString.toLowerCase());
        }) || []
      );
    });
  }, [usersSearchString]);

  // returns a mixture of props from host
  const normalizeEmptyValues = (hostData: any): { [key: string]: any } => {
    return reduce(
      hostData,
      (result, value, key) => {
        if ((Number.isFinite(value) && value !== 0) || !isEmpty(value)) {
          Object.assign(result, { [key]: value });
        } else {
          Object.assign(result, { [key]: "---" });
        }
        return result;
      },
      {}
    );
  };

  const wrapFleetHelper = (
    helperFn: (value: any) => string,
    value: string
  ): any => {
    return value === "---" ? value : helperFn(value);
  };

  const titleData = normalizeEmptyValues(
    pick(host, [
      "status",
      "issues",
      "memory",
      "cpu_type",
      "os_version",
      "osquery_version",
      "enroll_secret_name",
      "detail_updated_at",
      "percent_disk_space_available",
      "gigs_disk_space_available",
    ])
  );

  const operatingSystem = host?.os_version.slice(
    0,
    host?.os_version.lastIndexOf(" ")
  );
  const operatingSystemVersion = host?.os_version.slice(
    host?.os_version.lastIndexOf(" ") + 1
  );
  const osPolicyLabel = `Is ${operatingSystem}, version ${operatingSystemVersion} or later, installed?`;
  const osPolicy = `SELECT 1 from os_version WHERE name = '${operatingSystem}' AND major || '.' || minor || '.' || patch >= '${operatingSystemVersion}';`;

  const aboutData = normalizeEmptyValues(
    pick(host, [
      "seen_time",
      "uptime",
      "last_enrolled_at",
      "hardware_model",
      "hardware_serial",
      "primary_ip",
    ])
  );

  const osqueryData = normalizeEmptyValues(
    pick(host, [
      "config_tls_refresh",
      "logger_tls_period",
      "distributed_interval",
    ])
  );

  const togglePolicyDetailsModal = useCallback(
    (policy: IHostPolicy) => {
      setPolicyDetailsModal(!showPolicyDetailsModal);
      setSelectedPolicy(policy);
    },
    [showPolicyDetailsModal, setPolicyDetailsModal, setSelectedPolicy]
  );

  const toggleOSPolicyModal = useCallback(() => {
    setShowOSPolicyModal(!showOSPolicyModal);
  }, [showOSPolicyModal, setShowOSPolicyModal]);

  const onCancelPolicyDetailsModal = useCallback(() => {
    setPolicyDetailsModal(!showPolicyDetailsModal);
    setSelectedPolicy(null);
  }, [showPolicyDetailsModal, setPolicyDetailsModal, setSelectedPolicy]);

  const onCreateNewPolicy = () => {
    const { NEW_POLICY } = PATHS;
    host?.team_name
      ? setLastEditedQueryName(`${osPolicyLabel} (${host.team_name})`)
      : setLastEditedQueryName(osPolicyLabel);
    setPolicyTeamId(host?.team_id ? host?.team_id : 0);
    setLastEditedQueryDescription(
      "Checks to see if the required minimum operating system version is installed."
    );
    setLastEditedQueryBody(osPolicy);
    setLastEditedQueryResolution("");
    router.replace(NEW_POLICY);
  };

  const onDestroyHost = async () => {
    if (host) {
      try {
        await hostAPI.destroy(host);
        dispatch(
          renderFlash(
            "success",
            `Host "${host.hostname}" was successfully deleted.`
          )
        );
        router.push(PATHS.MANAGE_HOSTS);
      } catch (error) {
        console.log(error);
        dispatch(
          renderFlash("error", `Host "${host.hostname}" could not be deleted.`)
        );
      } finally {
        setShowDeleteHostModal(false);
      }
    }
  };

  const onRefetchHost = async () => {
    if (host) {
      // Once the user clicks to refetch, the refetch loading spinner should continue spinning
      // unless there is an error. The spinner state is also controlled in the fullyReloadHost
      // method.
      setShowRefetchSpinner(true);
      try {
        await hostAPI.refetch(host).then(() => {
          setRefetchStartTime(Date.now());
          setTimeout(() => {
            refetchHostDetails();
            refetchExtensions();
          }, 1000);
        });
      } catch (error) {
        console.log(error);
        dispatch(renderFlash("error", `Host "${host.hostname}" refetch error`));
        setShowRefetchSpinner(false);
      }
    }
  };

  const onLabelClick = (label: ILabel) => {
    if (label.name === "All Hosts") {
      return router.push(PATHS.MANAGE_HOSTS);
    }

    return router.push(`${PATHS.MANAGE_HOSTS}/labels/${label.id}`);
  };

  const onQueryHostCustom = () => {
    router.push(PATHS.NEW_QUERY + TAGGED_TEMPLATES.queryByHostRoute(host?.id));
  };

  const onQueryHostSaved = (selectedQuery: IQuery) => {
    router.push(
      PATHS.EDIT_QUERY(selectedQuery) +
        TAGGED_TEMPLATES.queryByHostRoute(host?.id)
    );
  };

  const onTransferHostSubmit = async (team: ITeam) => {
    const teamId = typeof team.id === "number" ? team.id : null;

    try {
      await hostAPI.transferToTeam(teamId, [hostIdFromURL]);

      const successMessage =
        teamId === null
          ? `Host successfully removed from teams.`
          : `Host successfully transferred to  ${team.name}.`;

      dispatch(renderFlash("success", successMessage));
      refetchHostDetails(); // Note: it is not necessary to `refetchExtensions` here because only team has changed
      setShowTransferHostModal(false);
    } catch (error) {
      console.log(error);
      dispatch(
        renderFlash("error", "Could not transfer host. Please try again.")
      );
    }
  };

  const onUsersTableSearchChange = useCallback(
    (queryData: ISearchQueryData) => {
      const { searchQuery } = queryData;
      setUsersSearchString(searchQuery);
    },
    []
  );

  const renderOsPolicyLabel = () => {
    const onCopyOsPolicy = (evt: React.MouseEvent) => {
      evt.preventDefault();

      stringToClipboard(osPolicy)
        .then(() => setCopyMessage("Copied!"))
        .catch(() => setCopyMessage("Copy failed"));

      // Clear message after 1 second
      setTimeout(() => setCopyMessage(""), 1000);

      return false;
    };

    return (
      <div>
        <span className={`${baseClass}__cta`}>{osPolicyLabel}</span>{" "}
        <span className={`${baseClass}__name`}>
          <span className="buttons">
            {copyMessage && <span>{`${copyMessage} `}</span>}
            <Button
              variant="unstyled"
              className={`${baseClass}__os-policy-copy-icon`}
              onClick={onCopyOsPolicy}
            >
              <img src={CopyIcon} alt="copy" />
            </Button>
          </span>
        </span>
      </div>
    );
  };

  const renderDeleteHostModal = () => (
    <Modal
      title="Delete host"
      onExit={() => setShowDeleteHostModal(false)}
      className={`${baseClass}__modal`}
    >
      <>
        <p>
          This action will delete the host <strong>{host?.hostname}</strong>{" "}
          from your Fleet instance.
        </p>
        <p>
          The host will automatically re-enroll when it checks back into Fleet.
        </p>
        <p>
          To prevent re-enrollment, you can uninstall osquery on the host or
          revoke the host&apos;s enroll secret.
        </p>
        <div className={`${baseClass}__modal-buttons`}>
          <Button onClick={onDestroyHost} variant="alert">
            Delete
          </Button>
          <Button
            onClick={() => setShowDeleteHostModal(false)}
            variant="inverse-alert"
          >
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );

  const renderOSPolicyModal = () => (
    <Modal
      title="Operating system"
      onExit={() => setShowOSPolicyModal(false)}
      className={`${baseClass}__modal`}
    >
      <>
        <p>
          <span className={`${baseClass}__os-modal-title`}>
            {titleData.os_version}{" "}
          </span>
          <span className={`${baseClass}__os-modal-updated`}>
            Reported {humanHostDetailUpdated(titleData.detail_updated_at)}
          </span>
        </p>
        <span className={`${baseClass}__os-modal-example-title`}>
          <TooltipWrapper tipContent="A policy is a yes or no question you can ask all your devices.">
            Example policy:
          </TooltipWrapper>
        </span>
        <InputField
          disabled
          inputWrapperClass={`${baseClass}__os-policy`}
          name="os-policy"
          label={renderOsPolicyLabel()}
          type={"textarea"}
          value={osPolicy}
        />
        <div className={`${baseClass}__modal-buttons`}>
          <Button onClick={onCreateNewPolicy} variant="brand">
            Create new policy
          </Button>
          <Button onClick={() => setShowOSPolicyModal(false)} variant="inverse">
            Close
          </Button>
        </div>
      </>
    </Modal>
  );

  const renderActionButtons = () => {
    const isOnline = host?.status === "online";

    return (
      <div className={`${baseClass}__action-button-container`}>
        {canTransferTeam && (
          <Button
            onClick={() => setShowTransferHostModal(true)}
            variant="text-icon"
            className={`${baseClass}__transfer-button`}
          >
            <>
              Transfer <img src={TransferIcon} alt="Transfer host icon" />
            </>
          </Button>
        )}
        <div data-tip data-for="query" data-tip-disable={isOnline}>
          <Button
            onClick={() => setShowQueryHostModal(true)}
            variant="text-icon"
            disabled={!isOnline}
            className={`${baseClass}__query-button`}
          >
            <>
              Query <img src={QueryIcon} alt="Query host icon" />
            </>
          </Button>
        </div>
        <ReactTooltip
          place="bottom"
          type="dark"
          effect="solid"
          id="query"
          backgroundColor="#3e4771"
        >
          <span className={`${baseClass}__tooltip-text`}>
            You can’t query <br /> an offline host.
          </span>
        </ReactTooltip>
        {currentUser && host && canDeleteHost(currentUser, host) && (
          <Button
            onClick={() => setShowDeleteHostModal(true)}
            variant="text-icon"
          >
            <>
              Delete <img src={DeleteIcon} alt="Delete host icon" />
            </>
          </Button>
        )}
      </div>
    );
  };

  const renderLabels = () => {
    const { labels = [] } = host || {};

    const labelItems = labels.map((label) => {
      return (
        <li className="list__item" key={label.id}>
          <Button
            onClick={() => onLabelClick(label)}
            variant="label"
            className="list__button"
          >
            {label.name}
          </Button>
        </li>
      );
    });

    return (
      <div className="section labels col-50">
        <p className="section__header">Labels</p>
        {labels.length === 0 ? (
          <p className="info-flex__item">
            No labels are associated with this host.
          </p>
        ) : (
          <ul className="list">{labelItems}</ul>
        )}
      </div>
    );
  };

  const renderPacks = () => {
    const packs = packsState;
    const wrapperClassName = `${baseClass}__pack-table`;
    const tableHeaders = generatePackTableHeaders();

    let packsAccordion;
    if (packs) {
      packsAccordion = packs.map((pack) => {
        return (
          <AccordionItem key={pack.pack_id}>
            <AccordionItemHeading>
              <AccordionItemButton>{pack.pack_name}</AccordionItemButton>
            </AccordionItemHeading>
            <AccordionItemPanel>
              {pack.query_stats.length === 0 ? (
                <div>There are no schedule queries for this pack.</div>
              ) : (
                <>
                  {!!pack.query_stats.length && (
                    <div className={`${wrapperClassName}`}>
                      <TableContainer
                        columns={tableHeaders}
                        data={generatePackDataSet(pack.query_stats)}
                        isLoading={isLoadingHost}
                        onQueryChange={() => null}
                        resultsTitle={"queries"}
                        defaultSortHeader={"scheduled_query_name"}
                        defaultSortDirection={"asc"}
                        showMarkAllPages={false}
                        isAllPagesSelected={false}
                        emptyComponent={() => <></>}
                        disablePagination
                        disableCount
                      />
                    </div>
                  )}
                </>
              )}
            </AccordionItemPanel>
          </AccordionItem>
        );
      });
    }

    return !packs || !packs.length ? null : (
      <div className="section section--packs">
        <p className="section__header">Packs</p>
        <Accordion allowMultipleExpanded allowZeroExpanded>
          {packsAccordion}
        </Accordion>
      </div>
    );
  };

  const renderSchedule = () => {
    const schedule = scheduleState;
    const wrapperClassName = `${baseClass}__pack-table`;
    const tableHeaders = generatePackTableHeaders();

    return (
      <div className="section section--packs">
        <p className="section__header">Schedule</p>
        {!schedule || !schedule.length ? (
          <div className="results__data">
            <b>No queries are scheduled for this host.</b>
            <p>
              Expecting to see queries? Try selecting “Refetch” to ask this host
              to report new vitals.
            </p>
          </div>
        ) : (
          <div className={`${wrapperClassName}`}>
            <TableContainer
              columns={tableHeaders}
              data={generatePackDataSet(schedule)}
              isLoading={isLoadingHost}
              onQueryChange={() => null}
              resultsTitle={"queries"}
              defaultSortHeader={"scheduled_query_name"}
              defaultSortDirection={"asc"}
              showMarkAllPages={false}
              isAllPagesSelected={false}
              emptyComponent={() => <></>}
              disablePagination
              disableCount
            />
          </div>
        )}
      </div>
    );
  };

  const renderPolicies = () => {
    if (!host?.policies?.length) {
      return (
        <div className="section section--policies">
          <p className="section__header">Policies</p>
          <div className="results__data">
            <b>No policies are checked for this host.</b>
            <p>
              Expecting to see policies? Try selecting “Refetch” to ask this
              host to report new vitals.
            </p>
          </div>
        </div>
      );
    }

    const tableHeaders = generatePolicyTableHeaders(togglePolicyDetailsModal);
    const noResponses: IHostPolicy[] =
      host?.policies?.filter(
        (policy) => !isValidPolicyResponse(policy.response)
      ) || [];
    const failingResponses: IHostPolicy[] =
      host?.policies?.filter((policy) => policy.response === "fail") || [];

    return (
      <div className="section section--policies">
        <p className="section__header">Policies</p>

        {host?.policies?.length && (
          <>
            {failingResponses?.length > 0 && (
              <PolicyFailingCount policyList={host?.policies} />
            )}
            {noResponses?.length > 0 && (
              <InfoBanner>
                <p>
                  This host is not updating the response for some policies.
                  Check out the Fleet documentation on&nbsp;
                  <a
                    href="https://fleetdm.com/docs/using-fleet/faq#why-my-host-is-not-updating-a-policys-response"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    why the response might not be updating
                  </a>
                  .
                </p>
              </InfoBanner>
            )}
            <TableContainer
              columns={tableHeaders}
              data={generatePolicyDataSet(host.policies)}
              isLoading={isLoadingHost}
              defaultSortHeader={"name"}
              defaultSortDirection={"asc"}
              resultsTitle={"policy items"}
              emptyComponent={() => <></>}
              showMarkAllPages={false}
              isAllPagesSelected={false}
              disablePagination
              disableCount
              highlightOnHover
            />
          </>
        )}
      </div>
    );
  };

  const renderUsers = () => {
    const { users } = host || {};

    const tableHeaders = generateUsersTableHeaders();

    if (users) {
      return (
        <div className="section section--users">
          <p className="section__header">Users</p>
          {users.length === 0 ? (
            <p className="results__data">
              No users were detected on this host.
            </p>
          ) : (
            <TableContainer
              columns={tableHeaders}
              data={usersState}
              isLoading={isLoadingHost}
              defaultSortHeader={"username"}
              defaultSortDirection={"asc"}
              inputPlaceHolder={"Search users by username"}
              onQueryChange={onUsersTableSearchChange}
              resultsTitle={"users"}
              emptyComponent={EmptyUsers}
              showMarkAllPages={false}
              isAllPagesSelected={false}
              searchable
              wideSearch
              filteredCount={usersState.length}
              isClientSidePagination
            />
          )}
        </div>
      );
    }
  };

  const renderSoftware = () => {
    return <SoftwareTab isLoading={isLoadingHost} software={hostSoftware} />;
  };

  const renderRefetch = () => {
    const isOnline = host?.status === "online";

    return (
      <>
        <div
          className="refetch"
          data-tip
          data-for="refetch-tooltip"
          data-tip-disable={isOnline || showRefetchSpinner}
        >
          <Button
            className={`
              button
              button--unstyled
              ${!isOnline ? "refetch-offline" : ""} 
              ${showRefetchSpinner ? "refetch-spinner" : "refetch-btn"}
            `}
            disabled={!isOnline}
            onClick={onRefetchHost}
          >
            {showRefetchSpinner
              ? "Fetching fresh vitals...this may take a moment"
              : "Refetch"}
          </Button>
        </div>
        <ReactTooltip
          place="bottom"
          type="dark"
          effect="solid"
          id="refetch-tooltip"
          backgroundColor="#3e4771"
        >
          <span className={`${baseClass}__tooltip-text`}>
            You can’t fetch data from <br /> an offline host.
          </span>
        </ReactTooltip>
      </>
    );
  };

  const renderIssues = () => (
    <div className="info-flex__item info-flex__item--title">
      <span className="info-flex__header">Issues</span>
      <span className="info-flex__data">
        <span
          className="host-issue tooltip__tooltip-icon"
          data-tip
          data-for="host-issue-count"
          data-tip-disable={false}
        >
          <img alt="host issue" src={IssueIcon} />
        </span>
        <ReactTooltip
          place="bottom"
          type="dark"
          effect="solid"
          backgroundColor="#3e4771"
          id="host-issue-count"
          data-html
        >
          <span className={`tooltip__tooltip-text`}>
            Failing policies ({host?.issues.failing_policies_count})
          </span>
        </ReactTooltip>
        <span className={`total-issues-count`}>
          {host?.issues.total_issues_count}
        </span>
      </span>
    </div>
  );

  const renderHostTeam = () => (
    <div className="info-flex__item info-flex__item--title">
      <span className="info-flex__header">Team</span>
      <span className={`info-flex__data`}>
        {host?.team_name ? (
          `${host?.team_name}`
        ) : (
          <span className="info-flex__no-team">No team</span>
        )}
      </span>
    </div>
  );

  const renderDeviceUser = () => {
    const numUsers = deviceMapping?.length;
    if (numUsers) {
      return (
        <div className="info-grid__block">
          <span className="info-grid__header">Used by</span>
          <span className="info-grid__data">
            {numUsers === 1 ? (
              deviceMapping[0].email || "---"
            ) : (
              <span className={`${baseClass}__device-mapping`}>
                <span
                  className="device-user"
                  data-tip
                  data-for="device-user-tooltip"
                >
                  {`${numUsers} users`}
                </span>
                <ReactTooltip
                  place="top"
                  type="dark"
                  effect="solid"
                  id="device-user-tooltip"
                  backgroundColor="#3e4771"
                >
                  <div
                    className={`${baseClass}__tooltip-text device-user-tooltip`}
                  >
                    {deviceMapping.map((user, i, arr) => (
                      <span key={user.email}>{`${user.email}${
                        i < arr.length - 1 ? ", " : ""
                      }`}</span>
                    ))}
                  </div>
                </ReactTooltip>
              </span>
            )}
          </span>
        </div>
      );
    }
    return null;
  };

  const renderDiskSpace = () => {
    if (
      host &&
      (host.gigs_disk_space_available > 0 ||
        host.percent_disk_space_available > 0)
    ) {
      return (
        <span className="info-flex__data">
          <div className="info-flex__disk-space">
            <div
              className={
                titleData.percent_disk_space_available > 20
                  ? "info-flex__disk-space-used"
                  : "info-flex__disk-space-warning"
              }
              style={{
                width: `${100 - titleData.percent_disk_space_available}%`,
              }}
            />
          </div>
          {titleData.gigs_disk_space_available} GB available
        </span>
      );
    }
    return <span className="info-flex__data">No data available</span>;
  };

  const renderMdmData = () => {
    if (!macadmins?.mobile_device_management) {
      return null;
    }
    const mdm = macadmins.mobile_device_management;
    return mdm.enrollment_status !== "Unenrolled" ? (
      <>
        <div className="info-grid__block">
          <span className="info-grid__header">MDM enrollment</span>
          <span className="info-grid__data">
            {mdm.enrollment_status || "---"}
          </span>
        </div>
        <div className="info-grid__block">
          <span className="info-grid__header">MDM server URL</span>
          <span className="info-grid__data">{mdm.server_url || "---"}</span>
        </div>
      </>
    ) : null;
  };

  const renderMunkiData = () => {
    if (!macadmins) {
      return null;
    }
    const { munki } = macadmins;
    return munki ? (
      <>
        <div className="info-grid__block">
          <span className="info-grid__header">Munki version</span>
          <span className="info-grid__data">{munki.version || "---"}</span>
        </div>
      </>
    ) : null;
  };

  if (isLoadingHost) {
    return <Spinner />;
  }

  const statusClassName = classnames("status", `status--${host?.status}`);
  return (
    <div className={`${baseClass} body-wrap`}>
      <div>
        <Link to={PATHS.MANAGE_HOSTS} className={`${baseClass}__back-link`}>
          <img src={BackChevron} alt="back chevron" id="back-chevron" />
          <span>Back to all hosts</span>
        </Link>
      </div>
      <div className="header title">
        <div className="title__inner">
          <div className="hostname-container">
            <h1 className="hostname">{host?.hostname || "---"}</h1>
            <p className="last-fetched">
              {`Last fetched ${humanHostDetailUpdated(
                titleData.detail_updated_at
              )}`}
              &nbsp;
            </p>
            {renderRefetch()}
          </div>
        </div>
        {renderActionButtons()}
      </div>
      <div className="section title">
        <div className="title__inner">
          <div className="info-flex">
            <div className="info-flex__item info-flex__item--title">
              <span className="info-flex__header">Status</span>
              <span className={`${statusClassName} info-flex__data`}>
                {titleData.status}
              </span>
            </div>
            {titleData.issues?.total_issues_count > 0 && renderIssues()}
            {isPremiumTier && renderHostTeam()}
            <div className="info-flex__item info-flex__item--title">
              <span className="info-flex__header">Disk Space</span>
              {renderDiskSpace()}
            </div>
            <div className="info-flex__item info-flex__item--title">
              <span className="info-flex__header">RAM</span>
              <span className="info-flex__data">
                {wrapFleetHelper(humanHostMemory, titleData.memory)}
              </span>
            </div>
            <div className="info-flex__item info-flex__item--title">
              <span className="info-flex__header">CPU</span>
              <span className="info-flex__data">{titleData.cpu_type}</span>
            </div>
            <div className="info-flex__item info-flex__item--title">
              <span className="info-flex__header">OS</span>
              <span className="info-flex__data">
                {isOnlyObserver ? (
                  `${titleData.os_version}`
                ) : (
                  <Button
                    onClick={() => toggleOSPolicyModal()}
                    variant="text-link"
                    className={`${baseClass}__os-policy-button`}
                  >
                    {titleData.os_version}
                  </Button>
                )}
              </span>
            </div>
            <div className="info-flex__item info-flex__item--title">
              <span className="info-flex__header">Osquery</span>
              <span className="info-flex__data">
                {titleData.osquery_version}
              </span>
            </div>
          </div>
        </div>
      </div>
      <TabsWrapper>
        <Tabs>
          <TabList>
            <Tab>Details</Tab>
            <Tab>Software</Tab>
            <Tab>Schedule</Tab>
            <Tab>Policies</Tab>
          </TabList>
          <TabPanel>
            <div className="section about">
              <p className="section__header">About this host</p>
              <div className="info-grid">
                <div className="info-grid__block">
                  <span className="info-grid__header">First enrolled</span>
                  <span className="info-grid__data">
                    {wrapFleetHelper(
                      humanHostEnrolled,
                      aboutData.last_enrolled_at
                    )}
                  </span>
                </div>
                <div className="info-grid__block">
                  <span className="info-grid__header">Last restarted</span>
                  <span className="info-grid__data">
                    {wrapFleetHelper(humanHostUptime, aboutData.uptime)}
                  </span>
                </div>
                <div className="info-grid__block">
                  <span className="info-grid__header">Hardware model</span>
                  <span className="info-grid__data">
                    {aboutData.hardware_model}
                  </span>
                </div>
                <div className="info-grid__block">
                  <span className="info-grid__header">Serial number</span>
                  <span className="info-grid__data">
                    {aboutData.hardware_serial}
                  </span>
                </div>
                <div className="info-grid__block">
                  <span className="info-grid__header">IPv4</span>
                  <span className="info-grid__data">
                    {aboutData.primary_ip}
                  </span>
                </div>
                {renderMunkiData()}
                {renderMdmData()}
                {renderDeviceUser()}
              </div>
            </div>
            <div className="col-2">
              <div className="section osquery col-50">
                <p className="section__header">Agent options</p>
                <div className="info-grid">
                  <div className="info-grid__block">
                    <span className="info-grid__header">
                      Config TLS refresh
                    </span>
                    <span className="info-grid__data">
                      {wrapFleetHelper(
                        secondsToHms,
                        osqueryData.config_tls_refresh
                      )}
                    </span>
                  </div>
                  <div className="info-grid__block">
                    <span className="info-grid__header">Logger TLS period</span>
                    <span className="info-grid__data">
                      {wrapFleetHelper(
                        secondsToHms,
                        osqueryData.logger_tls_period
                      )}
                    </span>
                  </div>
                  <div className="info-grid__block">
                    <span className="info-grid__header">
                      Distributed interval
                    </span>
                    <span className="info-grid__data">
                      {wrapFleetHelper(
                        secondsToHms,
                        osqueryData.distributed_interval
                      )}
                    </span>
                  </div>
                </div>
              </div>
              {renderLabels()}
            </div>
            {renderUsers()}
          </TabPanel>
          <TabPanel>{renderSoftware()}</TabPanel>
          <TabPanel>
            {renderSchedule()}
            {renderPacks()}
          </TabPanel>
          <TabPanel>{renderPolicies()}</TabPanel>
        </Tabs>
      </TabsWrapper>

      {showDeleteHostModal && renderDeleteHostModal()}
      {showQueryHostModal && host && (
        <SelectQueryModal
          host={host}
          onCancel={() => setShowQueryHostModal(false)}
          queries={fleetQueries || []}
          queryErrors={fleetQueriesError}
          isOnlyObserver={isOnlyObserver}
          onQueryHostCustom={onQueryHostCustom}
          onQueryHostSaved={onQueryHostSaved}
        />
      )}
      {!!host && showTransferHostModal && (
        <TransferHostModal
          onCancel={() => setShowTransferHostModal(false)}
          onSubmit={onTransferHostSubmit}
          teams={teams || []}
          isGlobalAdmin={isGlobalAdmin as boolean}
        />
      )}
      {!!host && showPolicyDetailsModal && (
        <PolicyDetailsModal
          onCancel={onCancelPolicyDetailsModal}
          policy={selectedPolicy}
        />
      )}
      {showOSPolicyModal && renderOSPolicyModal()}
    </div>
  );
};

export default HostDetailsPage;
