import React, { useCallback, useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";
import { useDebouncedCallback } from "use-debounce/lib";

import { AppContext } from "context/app";
import { ISoftware } from "interfaces/software";
import PATHS from "router/paths";
import softwareAPI, {
  ISoftwareCountResponse,
} from "services/entities/software";

import Button from "components/buttons/Button";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Spinner from "components/Spinner";
import TableContainer from "components/TableContainer";
import TableDataError from "components/TableDataError";
import TeamsDropdownHeader, {
  ITeamsDropdownContext,
} from "components/TeamsDropdown/TeamsDropdownHeader";

import generateTableHeaders from "./SoftwareTableConfig";

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

const PAGE_SIZE = 20;

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

  const [softwarePageIndex, setSoftwarePageIndex] = useState<number>(0);
  const [filterVuln, setFilterVuln] = useState(false);
  const [searchString, setSearchString] = useState("");
  const [isLoadingSoftware, setIsLoadingSoftware] = useState(true);
  const [isLoadingSoftwareCount, setIsLoadingSoftwareCount] = useState(true);

  // useQuery(["me"], () => usersAPI.me(), {
  //   onSuccess: ({ user, available_teams }: IGetMeResponse) => {
  //     setCurrentUser(user);
  //     setAvailableTeams(available_teams);
  //   },
  // });

  // const teamId = parseInt(location?.query?.team_id, 10) || 0;
  const teamId = currentTeam?.id;

  // TODO: Is our implementation of keepPreviousData and loading states causing bad UX and giving up
  // advantages of the react-query cache? Are we displaying data from cache for the current or prior
  // query while refetching? How does this work with debounce?
  const { data: software } = useQuery<ISoftware[], Error>(
    ["software", softwarePageIndex, searchString, filterVuln, teamId],
    () => {
      setIsLoadingSoftware(true);

      return softwareAPI.load({
        page: softwarePageIndex,
        perPage: PAGE_SIZE,
        query: searchString,
        orderKey: "id", // count,name,id
        orderDir: "desc",
        vulnerable: filterVuln,
        teamId: teamId && teamId, // TODO ask luke about this
      });
    },
    {
      // enabled: true,
      // If keepPreviousData is enabled,
      // useQuery no longer returns isLoading when making new calls after load
      // So we manage our own load states
      keepPreviousData: true,
      // staleTime: 500,
      onSuccess: () => {
        setIsLoadingSoftware(false);
      },
      // TODO: error UX?
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
    ["softwareCount", searchString, filterVuln, teamId],
    () => {
      setIsLoadingSoftwareCount(true);
      return softwareAPI.count({
        query: searchString,
        vulnerable: filterVuln,
        teamId: teamId && teamId,
      });
    },
    {
      keepPreviousData: true,
      refetchOnWindowFocus: false,
      retry: 1,
      select: (data) => data.count,
      onSuccess: () => {
        setIsLoadingSoftwareCount(false);
      },
      onError: (err) => {
        console.log("useQuery error: ", err);
        setIsLoadingSoftwareCount(false);
      },
    }
  );

  const onQueryChange = useDebouncedCallback(
    async ({ pageIndex, searchQuery }: ITableQueryProps) => {
      setSearchString(searchQuery);

      if (pageIndex !== softwarePageIndex) {
        setSoftwarePageIndex(pageIndex);
      }
    },
    300
  );

  const onTeamSelect = (ctx: ITeamsDropdownContext) => {
    setSoftwarePageIndex(0);
  };

  const renderHeaderButtons = (
    ctx: ITeamsDropdownContext
  ): JSX.Element | null => {
    if ((ctx.isGlobalAdmin || ctx.isGlobalMaintainer) && ctx.teamId === 0) {
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

  const renderHeaderDescription = (ctx: ITeamsDropdownContext) => {
    return (
      <p>
        Search for installed software and manage automations for detected
        vulnerabilities (CVEs) on{" "}
        <b>
          {ctx.isPremiumTier && !!ctx.teamId
            ? "all hosts assigned to this team"
            : "all of your hosts"}
        </b>
        .
      </p>
    );
  };

  const renderSoftwareCount = useCallback(() => {
    const count = softwareCount;

    if (softwareCountError) {
      return <span className="count-error">Failed to load software count</span>;
    }

    return count !== undefined ? (
      <span
        className={`${isLoadingSoftwareCount ? "count-loading" : ""}`}
      >{`${count} software item${count === 1 ? "" : "s"}`}</span>
    ) : (
      <></>
    );
  }, [isLoadingSoftwareCount, softwareCountError, softwareCount]);

  const renderVulnFilterDropdown = () => {
    return (
      <Dropdown
        value={filterVuln}
        className={`${baseClass}__status_dropdown`}
        options={VULNERABLE_OPTIONS}
        searchable={false}
        onChange={(value: boolean) => {
          setFilterVuln(value);
          setSoftwarePageIndex(0); // TODO: why doesn't previous page link disable? how to get table container to follow this page index
        }}
      />
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
        <TableContainer
          columns={generateTableHeaders()}
          data={software || []}
          isLoading={isLoadingSoftware}
          defaultSortHeader={"hosts"}
          defaultSortDirection={"desc"}
          hideActionButton
          resultsTitle={"software items"}
          emptyComponent={() =>
            // EmptySoftware(modalSoftwareSearchText === "" ? "modal" : "search")
            null
          }
          showMarkAllPages={false}
          isAllPagesSelected={false}
          searchable
          disableActionButton
          pageSize={PAGE_SIZE}
          onQueryChange={onQueryChange}
          customControl={renderVulnFilterDropdown}
          renderCount={renderSoftwareCount}
        />
      </div>
    </div>
  );
};

export default ManageSoftwarePage;
