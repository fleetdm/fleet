import React, { useCallback, useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";
import { useDebouncedCallback } from "use-debounce/lib";

import { AppContext } from "context/app";
import { ISoftware } from "interfaces/software";
import PATHS from "router/paths";
import softwareAPI, {
  ISoftwareCountResponse,
} from "services/entities/software";
import usersAPI, { IGetMeResponse } from "services/entities/users";

import Button from "components/buttons/Button";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Spinner from "components/Spinner";
import TableContainer from "components/TableContainer";
import TableDataError from "components/TableDataError";
import TeamsDropdown from "components/TeamsDropdown";

import generateTableHeaders from "./SoftwareTableConfig";

interface IManageSoftwarePageProps {
  router: any;
  location: any;
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
  const {
    availableTeams,
    isGlobalAdmin,
    isGlobalMaintainer,
    isOnGlobalTeam,
    isFreeTier,
    isPremiumTier,
    currentTeam,
    setAvailableTeams,
    setCurrentUser,
    setCurrentTeam,
  } = useContext(AppContext);

  const [softwarePageIndex, setSoftwarePageIndex] = useState<number>(0);
  const [filterVuln, setFilterVuln] = useState(false);
  const [searchString, setSearchString] = useState("");
  const [isLoadingSoftware, setIsLoadingSoftware] = useState(true);
  const [isLoadingSoftwareCount, setIsLoadingSoftwareCount] = useState(true);

  useQuery(["me"], () => usersAPI.me(), {
    onSuccess: ({ user, available_teams }: IGetMeResponse) => {
      setCurrentUser(user);
      setAvailableTeams(available_teams);
    },
  });

  const teamId = parseInt(location?.query?.team_id, 10) || 0;

  const renderTeamDescription = () => {
    return (
      <p>
        Search for installed software and manage automations for detected
        vulnerabilities (CVEs) on{" "}
        <b>
          {isPremiumTier && !!teamId
            ? "all hosts assigned to this team"
            : "all of your hosts"}
        </b>
        .
      </p>
    );
  };

  const findAvailableTeam = (id: number) => {
    return availableTeams?.find((t) => t.id === id);
  };

  const handleTeamSelect = (id: number) => {
    const { MANAGE_SOFTWARE } = PATHS;

    const selectedTeam = findAvailableTeam(id);
    const path = selectedTeam?.id
      ? `${MANAGE_SOFTWARE}?team_id=${selectedTeam.id}`
      : MANAGE_SOFTWARE;

    router.replace(path);
    setCurrentTeam(selectedTeam);
    setSoftwarePageIndex(0);
  };

  // If team_id from URL query params is not valid, we instead use a default team
  // either the current team (if any) or all teams (for global users) or
  // the first available team (for non-global users)
  const getValidatedTeamId = () => {
    if (findAvailableTeam(teamId)) {
      return teamId;
    }
    if (!teamId && currentTeam) {
      return currentTeam.id;
    }
    if (!teamId && !currentTeam && !isOnGlobalTeam && availableTeams) {
      return availableTeams[0]?.id;
    }
    return 0;
  };

  // If team_id or currentTeam doesn't match validated id, switch to validated id
  useEffect(() => {
    if (availableTeams) {
      const validatedId = getValidatedTeamId();

      if (validatedId !== currentTeam?.id || validatedId !== teamId) {
        handleTeamSelect(validatedId);
      }
    }
  }, [availableTeams]);

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
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            <div className={`${baseClass}__text`}>
              <div className={`${baseClass}__title`}>
                {isFreeTier && <h1>Software</h1>}
                {isPremiumTier &&
                  (availableTeams.length > 1 || isOnGlobalTeam) && (
                    <TeamsDropdown
                      currentUserTeams={availableTeams || []}
                      selectedTeamId={teamId}
                      onChange={(newSelectedValue: number) =>
                        handleTeamSelect(newSelectedValue)
                      }
                    />
                  )}
                {isPremiumTier &&
                  !isOnGlobalTeam &&
                  availableTeams.length === 1 && (
                    <h1>{availableTeams[0].name}</h1>
                  )}
              </div>
            </div>
          </div>
          <div className={`${baseClass} button-wrap`}>
            {(isGlobalAdmin || isGlobalMaintainer) && teamId === 0 && (
              <Button
                onClick={() => console.log("Manage automations button click")}
                className={`${baseClass}__manage-automations button`}
                variant="brand"
              >
                <span>Manage automations</span>
              </Button>
            )}
          </div>
        </div>
        <div className={`${baseClass}__description`}>
          {renderTeamDescription()}
        </div>
        <div>
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
    </div>
  );
};

export default ManageSoftwarePage;
