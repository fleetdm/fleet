import React, { useCallback, useContext, useState } from "react";
import { useQuery } from "react-query";
import ReactTooltip from "react-tooltip";
import { useDebouncedCallback } from "use-debounce/lib";
import formatDistanceToNow from "date-fns/formatDistanceToNow";

import { AppContext } from "context/app";
import softwareAPI, {
  ISoftwareResponse,
  ISoftwareCountResponse,
} from "services/entities/software";
import { GITHUB_NEW_ISSUE_LINK } from "utilities/constants";

import Button from "components/buttons/Button";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Spinner from "components/Spinner";
import TableContainer from "components/TableContainer";
import TableDataError from "components/TableDataError";
import TeamsDropdownHeader, {
  ITeamsDropdownState,
} from "components/TeamsDropdown/TeamsDropdownHeader";

import ExternalLinkIcon from "../../../../assets/images/open-new-tab-12x12@2x.png";
import QuestionIcon from "../../../../assets/images/icon-question-16x16@2x.png";

import generateTableHeaders from "./SoftwareTableConfig";
import EmptySoftware from "../components/EmptySoftware";

interface IManageSoftwarePageProps {
  router: any;
  location: any;
  params: any;
}

interface ITableQueryProps {
  pageIndex: number;
  pageSize: number;
  searchQuery: string;
  sortHeader: string;
  sortDirection: string;
}

const baseClass = "manage-software-page";

const DEFAULT_SORT_DIRECTION = "desc";

const DEFAULT_SORT_HEADER = "hosts_count";

const PAGE_SIZE = 20;

const SORT_DIRECTIONS = ["asc", "desc"] as const;

const VULNERABLE_OPTIONS = [
  {
    disabled: false,
    label: "All software",
    value: false,
    helpText: "All sofware installed on your hosts.",
  },
  {
    disabled: false,
    label: "Vulnerable software",
    value: true,
    helpText:
      "All software installed on your hosts with detected vulnerabilities.",
  },
];

const ManageSoftwarePage = ({
  router,
  location,
}: IManageSoftwarePageProps): JSX.Element => {
  const { availableTeams, currentTeam } = useContext(AppContext);

  const [isLoadingSoftware, setIsLoadingSoftware] = useState(true);
  const [isLoadingCount, setIsLoadingCount] = useState(true);
  const [filterVuln, setFilterVuln] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [sortDirection, setSortDirection] = useState(DEFAULT_SORT_DIRECTION);
  const [sortHeader, setSortHeader] = useState(DEFAULT_SORT_HEADER);
  const [pageIndex, setPageIndex] = useState<number>(0);

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
      pageIndex,
      searchQuery,
      sortHeader,
      sortDirection,
      filterVuln,
      teamId,
    ],
    () => {
      setIsLoadingSoftware(true);
      console.log("API pageIndex: ", pageIndex);

      return softwareAPI.load({
        page: pageIndex,
        perPage: PAGE_SIZE,
        query: searchQuery,
        // TODO confirm sort is working?
        orderKey: sortHeader,
        orderDir:
          SORT_DIRECTIONS.find((d) => d === sortDirection) ||
          DEFAULT_SORT_DIRECTION,
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
      // staleTime: 500,
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
    ["softwareCount", searchQuery, filterVuln, teamId],
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
    }: ITableQueryProps) => {
      pageIndex !== newPageIndex && setPageIndex(newPageIndex);
      searchQuery !== newSearchQuery && setSearchQuery(newSearchQuery);
      sortDirection !== newSortDirection && setSortDirection(newSortDirection);
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

  // TODO: Handle BE returning "0001-01-01T00:00:00Z" when counts have not yet run.
  const renderSoftwareCount = useCallback(() => {
    const count = softwareCount;
    let lastUpdatedAt = software?.counts_updated_at;
    lastUpdatedAt = lastUpdatedAt
      ? formatDistanceToNow(new Date(lastUpdatedAt), {
          includeSeconds: true,
          addSuffix: true,
        })
      : "never";

    if (softwareCountError && !isLoadingCount) {
      return (
        <span className={`${baseClass}__count count-error`}>
          Failed to load software count
        </span>
      );
    }

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

  const renderVulnFilterDropdown = () => {
    return (
      <Dropdown
        value={filterVuln}
        className={`${baseClass}__status_dropdown`}
        options={VULNERABLE_OPTIONS}
        searchable={false}
        onChange={(value: boolean) => {
          setFilterVuln(value);
          setPageIndex(0);
        }}
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
        <TeamsDropdownHeader
          location={location}
          router={router}
          baseClass={baseClass}
          onChange={onTeamSelect}
          defaultTitle="Software"
          description={renderHeaderDescription}
          buttons={renderHeaderButtons}
        />
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
          />
        )}
      </div>
    </div>
  );
};

export default ManageSoftwarePage;
