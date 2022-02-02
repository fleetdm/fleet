import React, { useCallback, useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";
import { useDispatch } from "react-redux";
import { InjectedRouter } from "react-router/lib/Router";
import ReactTooltip from "react-tooltip";
import { useDebouncedCallback } from "use-debounce/lib";
import formatDistanceToNowStrict from "date-fns/formatDistanceToNowStrict";

import { AppContext } from "context/app";
import { IWebhookSoftwareVulnerabilities } from "interfaces/webhook";
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";
import PATHS from "router/paths";
import configAPI from "services/entities/config";
import usersAPI, { IGetMeResponse } from "services/entities/users";
import softwareAPI, {
  ISoftwareResponse,
  ISoftwareCountResponse,
} from "services/entities/software";
import {
  GITHUB_NEW_ISSUE_LINK,
  VULNERABLE_DROPDOWN_OPTIONS,
} from "utilities/constants";

import Button from "components/buttons/Button";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Spinner from "components/Spinner";
import TableContainer, { ITableQueryData } from "components/TableContainer";
import TableDataError from "components/TableDataError";
import TeamsDropdownHeader, {
  ITeamsDropdownState,
} from "components/PageHeader/TeamsDropdownHeader";

import ExternalLinkIcon from "../../../../assets/images/open-new-tab-12x12@2x.png";
import QuestionIcon from "../../../../assets/images/icon-question-16x16@2x.png";

import generateTableHeaders from "./SoftwareTableConfig";
import ManageAutomationsModal from "./components/ManageAutomationsModal";
import EmptySoftware from "../components/EmptySoftware";

interface IManageSoftwarePageProps {
  router: InjectedRouter;
  location: {
    pathname: string;
    query: { vulnerable?: boolean };
    search: string;
  };
}
const DEFAULT_SORT_DIRECTION = "desc";
const DEFAULT_SORT_HEADER = "hosts_count";
const PAGE_SIZE = 20;

const baseClass = "manage-software-page";

const ManageSoftwarePage = ({
  router,
  location,
}: IManageSoftwarePageProps): JSX.Element => {
  const dispatch = useDispatch();
  const {
    availableTeams,
    currentTeam,
    setAvailableTeams,
    setCurrentUser,
  } = useContext(AppContext);

  const [isLoadingSoftware, setIsLoadingSoftware] = useState(true);
  const [isLoadingCount, setIsLoadingCount] = useState(true);
  const [filterVuln, setFilterVuln] = useState(
    location?.query?.vulnerable || false
  );
  const [searchQuery, setSearchQuery] = useState("");
  const [sortDirection, setSortDirection] = useState<
    "asc" | "desc" | undefined
  >(DEFAULT_SORT_DIRECTION);
  const [sortHeader, setSortHeader] = useState(DEFAULT_SORT_HEADER);
  const [pageIndex, setPageIndex] = useState(0);
  const [showManageAutomationsModal, setShowManageAutomationsModal] = useState(
    false
  );
  const [showPreviewPayloadModal, setShowPreviewPayloadModal] = useState(false);

  // TODO: experiment to see if we need this state and effect or can we rely solely on the router/location for the dropdown state?
  useEffect(() => {
    setFilterVuln(!!location.query.vulnerable);
  }, [location]);

  // TODO: combine string and object so only one array element in query key; figure out typing and
  // destructuring for queryfn
  const [
    isLoadingSoftwareVulnerabilitiesWebhook,
    setIsLoadingSoftwareVulnerabilitiesWebhook,
  ] = useState(true);
  const [
    isSoftwareVulnerabilitiesWebhookError,
    setIsSoftwareVulnerabilitiesWebhookError,
  ] = useState(false);
  const [
    softwareVulnerabilitiesWebhook,
    setSoftwareVulnerabilitiesWebhook,
  ] = useState<IWebhookSoftwareVulnerabilities | undefined>();

  useQuery(["me"], () => usersAPI.me(), {
    onSuccess: ({ user, available_teams }: IGetMeResponse) => {
      setCurrentUser(user);
      setAvailableTeams(available_teams);
    },
  });

  const { data: software, error: softwareError } = useQuery<
    ISoftwareResponse,
    Error
  >(
    [
      "software",
      {
        pageIndex,
        pageSize: PAGE_SIZE,
        searchQuery,
        sortDirection,
        sortHeader,
        teamId: currentTeam?.id,
        vulnerable: !!location.query.vulnerable,
        urlPath: location.pathname,
        urlQueryString: location.search,
      },
    ],
    () => {
      setIsLoadingSoftware(true);
      const params = {
        page: pageIndex,
        perPage: PAGE_SIZE,
        query: searchQuery,
        orderKey: sortHeader,
        orderDir: sortDirection || DEFAULT_SORT_DIRECTION,
        vulnerable: !!location.query.vulnerable,
        teamId: currentTeam?.id,
      };
      return softwareAPI.load(params);
    },
    {
      // If keepPreviousData is enabled,
      // useQuery no longer returns isLoading when making new calls after load
      // So we manage our own load states
      keepPreviousData: true,
      staleTime: 30000, // stale time can be adjusted if fresher data is desired based on software inventory interval
      onSuccess: () => {
        setIsLoadingSoftware(false);
      },
      onError: () => {
        setIsLoadingSoftware(false);
      },
    }
  );

  const { data: softwareCount, error: softwareCountError } = useQuery<
    ISoftwareCountResponse,
    Error,
    number
  >(
    [
      "softwareCount",
      {
        searchQuery,
        vulnerable: !!location.query.vulnerable,
        teamId: currentTeam?.id,
      },
    ],
    () => {
      setIsLoadingCount(true);
      return softwareAPI.count({
        query: searchQuery,
        vulnerable: !!location.query.vulnerable,
        teamId: currentTeam?.id,
      });
    },
    {
      keepPreviousData: true,
      staleTime: 30000, // stale time can be adjusted if fresher data is desired based on software inventory interval
      refetchOnWindowFocus: false,
      retry: 1,
      select: (data) => data.count,
      onSuccess: () => {
        setIsLoadingCount(false);
      },
      onError: (err) => {
        console.log("useQuery error: ", err);
        setIsLoadingCount(false);
      },
    }
  );

  const onQueryChange = useDebouncedCallback(
    async ({
      pageIndex: newPageIndex,
      searchQuery: newSearchQuery,
      sortDirection: newSortDirection,
      sortHeader: newSortHeader,
    }: ITableQueryData) => {
      pageIndex !== newPageIndex && setPageIndex(newPageIndex);
      searchQuery !== newSearchQuery && setSearchQuery(newSearchQuery);
      sortDirection !== newSortDirection &&
        setSortDirection(
          newSortDirection === "asc" || newSortDirection === "desc"
            ? newSortDirection
            : DEFAULT_SORT_DIRECTION
        );
      sortHeader !== newSortHeader && setSortHeader(newSortHeader);
    },
    300
  );

  const getSoftwareVulnerabilitiesWebhook = useCallback(async () => {
    setIsLoadingSoftwareVulnerabilitiesWebhook(true);
    setIsSoftwareVulnerabilitiesWebhookError(false);
    let result;
    try {
      result = await configAPI
        .loadAll()
        .then((response) => response.webhook_settings.vulnerabilities_webhook);
      console.log("result", result);
      setSoftwareVulnerabilitiesWebhook(result);
    } catch (error) {
      console.log(error);
      setIsSoftwareVulnerabilitiesWebhookError(true);
    } finally {
      setIsLoadingSoftwareVulnerabilitiesWebhook(false);
    }
    console.log("result", result);
    return result;
  }, []);

  const toggleManageAutomationsModal = () =>
    setShowManageAutomationsModal(!showManageAutomationsModal);

  const togglePreviewPayloadModal = useCallback(() => {
    setShowPreviewPayloadModal(!showPreviewPayloadModal);
  }, [setShowPreviewPayloadModal, showPreviewPayloadModal]);

  const onManageAutomationsClick = () => {
    toggleManageAutomationsModal();
  };

  const onCreateWebhookSubmit = async ({
    destination_url,
    enable_vulnerabilities_webhook,
  }: IWebhookSoftwareVulnerabilities) => {
    try {
      const request = configAPI.update({
        webhook_settings: {
          vulnerabilities_webhook: {
            destination_url,
            enable_vulnerabilities_webhook,
          },
        },
      });
      await request.then(() => {
        dispatch(
          renderFlash(
            "success",
            "Successfully updated vulnerability automations."
          )
        );
      });
    } catch {
      dispatch(
        renderFlash(
          "error",
          "Could not update vulnerability automations. Please try again."
        )
      );
    } finally {
      toggleManageAutomationsModal();
      getSoftwareVulnerabilitiesWebhook();
    }
  };

  const onTeamSelect = () => {
    setPageIndex(0);
  };

  const renderHeaderButtons = (
    state: ITeamsDropdownState
  ): JSX.Element | null => {
    if (
      (state.isGlobalAdmin || state.isGlobalMaintainer) &&
      state.teamId === 0
    ) {
      return (
        <Button
          onClick={onManageAutomationsClick}
          className={`${baseClass}__manage-automations button`}
          variant="brand"
        >
          <span>Manage automations</span>
        </Button>
      );
    }
    return null;
  };

  const renderHeaderDescription = (state: ITeamsDropdownState) => {
    return (
      <p>
        Search for installed software and manage automations for detected
        vulnerabilities (CVEs) on{" "}
        <b>
          {state.isPremiumTier && !!state.teamId
            ? "all hosts assigned to this team"
            : "all of your hosts"}
        </b>
        .
      </p>
    );
  };

  const renderHeader = useCallback(() => {
    return (
      <TeamsDropdownHeader
        location={location}
        router={router}
        baseClass={baseClass}
        onChange={onTeamSelect}
        defaultTitle="Software"
        description={renderHeaderDescription}
        buttons={renderHeaderButtons}
      />
    );
  }, [router, location]);

  const renderSoftwareCount = useCallback(() => {
    const count = softwareCount;
    let lastUpdatedAt = software?.counts_updated_at;
    if (!lastUpdatedAt || lastUpdatedAt === "0001-01-01T00:00:00Z") {
      lastUpdatedAt = "never";
    } else {
      lastUpdatedAt = formatDistanceToNowStrict(new Date(lastUpdatedAt), {
        addSuffix: true,
      });
    }

    if (softwareCountError && !isLoadingCount) {
      return (
        <span className={`${baseClass}__count count-error`}>
          Failed to load software count
        </span>
      );
    }

    // TODO: Use setInterval to keep last updated time current?
    return count !== undefined ? (
      <span
        className={`${baseClass}__count ${
          isLoadingCount ? "count-loading" : ""
        }`}
      >
        {`${count} software item${count === 1 ? "" : "s"}`}
        <span className="count-last-updated">
          {`Last updated ${lastUpdatedAt}`}{" "}
          <span className={`tooltip`}>
            <span
              className={`tooltip__tooltip-icon`}
              data-tip
              data-for="last-updated-tooltip"
              data-tip-disable={false}
            >
              <img alt="question icon" src={QuestionIcon} />
            </span>
            <ReactTooltip
              place="top"
              type="dark"
              effect="solid"
              backgroundColor="#3e4771"
              id="last-updated-tooltip"
              data-html
            >
              <span className={`tooltip__tooltip-text`}>
                Fleet periodically
                <br />
                queries all hosts
                <br />
                to retrieve software
              </span>
            </ReactTooltip>
          </span>
        </span>
      </span>
    ) : null;
  }, [isLoadingCount, software, softwareCountError, softwareCount]);

  // TODO: retool this with react-router location descriptor objects
  const buildUrlQueryString = (queryString: string, vulnerable: boolean) => {
    queryString = queryString.startsWith("?")
      ? queryString.slice(1)
      : queryString;
    const queryParams = queryString.split("&").filter((el) => el.includes("="));
    const index = queryParams.findIndex((el) => el.includes("vulnerable"));

    if (vulnerable) {
      const vulnParam = `vulnerable=${vulnerable}`;
      if (index >= 0) {
        // replace old vuln param
        queryParams.splice(index, 1, vulnParam);
      } else {
        // add new vuln param
        queryParams.push(vulnParam);
      }
    } else {
      // remove old vuln param
      index >= 0 && queryParams.splice(index, 1);
    }
    queryString = queryParams.length ? "?".concat(queryParams.join("&")) : "";

    return queryString;
  };

  const onVulnFilterChange = useCallback(
    (vulnerable: boolean) => {
      setFilterVuln(vulnerable);
      setPageIndex(0);
      const queryString = buildUrlQueryString(location?.search, vulnerable);
      if (location?.search !== queryString) {
        const path = location?.pathname?.concat(queryString);
        !!path && router.replace(path);
      }
    },
    [location, router]
  );

  const renderVulnFilterDropdown = () => {
    return (
      <Dropdown
        value={filterVuln}
        className={`${baseClass}__status_dropdown`}
        options={VULNERABLE_DROPDOWN_OPTIONS}
        searchable={false}
        onChange={onVulnFilterChange}
      />
    );
  };

  const renderTableFooter = () => {
    return (
      <div>
        Seeing unexpected software or vulnerabilities?{" "}
        <a
          href={GITHUB_NEW_ISSUE_LINK}
          target="_blank"
          rel="noopener noreferrer"
        >
          File an issue on GitHub
          <img alt="External link" src={ExternalLinkIcon} />
        </a>
      </div>
    );
  };

  console.log("softwareVulnerabilitiesWebhook", softwareVulnerabilitiesWebhook);
  return !availableTeams ? (
    <Spinner />
  ) : (
    <div className={baseClass}>
      <div className={`${baseClass}__wrapper body-wrap`}>
        {renderHeader()}
        {softwareError && !isLoadingSoftware ? (
          <TableDataError />
        ) : (
          <TableContainer
            columns={generateTableHeaders()}
            data={software?.software || []}
            isLoading={isLoadingSoftware}
            resultsTitle={"software items"}
            emptyComponent={() =>
              EmptySoftware(
                (filterVuln && "vulnerable") ||
                  (searchQuery && "search") ||
                  "default"
              )
            }
            defaultSortHeader={"hosts_count"}
            defaultSortDirection={"desc"}
            manualSortBy
            pageSize={PAGE_SIZE}
            showMarkAllPages={false}
            isAllPagesSelected={false}
            searchable
            inputPlaceHolder="Search software by name or vulnerabilities (CVEs)"
            onQueryChange={onQueryChange}
            additionalQueries={filterVuln ? "vulnerable" : ""} // additionalQueries serves as a trigger
            // for the useDeepEffect hook to fire onQueryChange for events happeing outside of
            // the TableContainer
            customControl={renderVulnFilterDropdown}
            renderCount={renderSoftwareCount}
            renderFooter={renderTableFooter}
            disableActionButton
            hideActionButton
            highlightOnHover
          />
        )}

        {showManageAutomationsModal && (
          <ManageAutomationsModal
            onCancel={toggleManageAutomationsModal}
            onCreateWebhookSubmit={onCreateWebhookSubmit}
            togglePreviewPayloadModal={togglePreviewPayloadModal}
            showPreviewPayloadModal={showPreviewPayloadModal}
            softwareVulnerabilityWebhookEnabled={
              softwareVulnerabilitiesWebhook &&
              softwareVulnerabilitiesWebhook.enable_vulnerabilities_webhook
            }
            currentDestinationUrl={
              (softwareVulnerabilitiesWebhook &&
                softwareVulnerabilitiesWebhook.destination_url) ||
              ""
            }
          />
        )}
      </div>
    </div>
  );
};

export default ManageSoftwarePage;
