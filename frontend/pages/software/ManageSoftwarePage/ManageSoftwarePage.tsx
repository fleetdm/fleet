import React, { useCallback, useContext, useState } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router/lib/Router";
import ReactTooltip from "react-tooltip";
import { useDebouncedCallback } from "use-debounce/lib";
import formatDistanceToNowStrict from "date-fns/formatDistanceToNowStrict";

import { AppContext } from "context/app";
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
} from "components/TeamsDropdown/TeamsDropdownHeader";

import ExternalLinkIcon from "../../../../assets/images/open-new-tab-12x12@2x.png";
import QuestionIcon from "../../../../assets/images/icon-question-16x16@2x.png";

import generateTableHeaders from "./SoftwareTableConfig";
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
  const { availableTeams, currentTeam } = useContext(AppContext);

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

  // const teamId = parseInt(location?.query?.team_id, 10) || 0;
  const teamId = currentTeam?.id;

  // TODO: Is our implementation of keepPreviousData and loading states causing bad UX and giving up
  // advantages of the react-query cache? Are we displaying data from cache for the current or prior
  // query while refetching? How does this work with debounce?
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
        teamId,
        vulnerable: filterVuln,
      },
    ],
    () => {
      setIsLoadingSoftware(true);
      return softwareAPI.load({
        page: pageIndex,
        perPage: PAGE_SIZE,
        query: searchQuery,
        orderKey: sortHeader,
        orderDir: sortDirection || DEFAULT_SORT_DIRECTION,
        vulnerable: filterVuln,
        teamId,
      });
    },
    {
      // initialData: { software: [], counts_updated_at: "" },
      // placeholderData: { software: [], counts_updated_at: "" },
      // enabled: true,
      // If keepPreviousData is enabled,
      // useQuery no longer returns isLoading when making new calls after load
      // So we manage our own load states
      keepPreviousData: true,
      staleTime: 30000, // TODO: Discuss a reasonable staleTime given that counts are only updated infrequently?
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
    ["softwareCount", { searchQuery, vulnerable: filterVuln, teamId }],
    () => {
      setIsLoadingCount(true);
      return softwareAPI.count({
        query: searchQuery,
        vulnerable: filterVuln,
        teamId,
      });
    },
    {
      keepPreviousData: true,
      staleTime: 30000, // TODO: Discuss a reasonable staleTime given that counts are only updated infrequently?
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
          onClick={() => console.log("Manage automations button click")}
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

  // TODO: Ask backend to implement different approach to returning "0001-01-01T00:00:00Z" when counts have not yet run.
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
            // TODO: Consider renaming additionalQueries. Essentially this serves as a trigger
            // for the useDeepEffect hook to fire onQueryChange for events happeing outside of
            // the TableContainer
            additionalQueries={filterVuln ? "vulnerable" : ""}
            customControl={renderVulnFilterDropdown}
            renderCount={renderSoftwareCount}
            renderFooter={renderTableFooter}
            disableActionButton
            hideActionButton
            highlightOnHover
          />
        )}
      </div>
    </div>
  );
};

export default ManageSoftwarePage;
