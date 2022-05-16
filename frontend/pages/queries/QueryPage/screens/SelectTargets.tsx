import React, { useContext, useEffect, useState } from "react";
import { Row } from "react-table";
import { useQuery } from "react-query";
import { useDebouncedCallback } from "use-debounce/lib";

import { AppContext } from "context/app";
import { QueryContext } from "context/query";

import { IHost } from "interfaces/host";
import { ILabel, ILabelSummary } from "interfaces/label";
import {
  ITarget,
  ISelectLabel,
  ISelectTeam,
  ISelectTargetsEntity,
  ISelectedTargets,
} from "interfaces/target";
import { ITeam } from "interfaces/team";

import labelsAPI, { ILabelsSummaryResponse } from "services/entities/labels";
import targetsAPI, {
  ITargetsCountResponse,
  ITargetsSearchResponse,
} from "services/entities/targets";
import teamsAPI, { ILoadTeamsResponse } from "services/entities/teams";
import { formatSelectedTargetsForApi } from "utilities/helpers";

import PageError from "components/DataError";
import TargetsInput from "components/TargetsInput";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
import TooltipWrapper from "components/TooltipWrapper";
import PlusIcon from "../../../../../assets/images/icon-plus-purple-32x32@2x.png";
import CheckIcon from "../../../../../assets/images/icon-check-purple-32x32@2x.png";

interface ITargetPillSelectorProps {
  entity: ISelectLabel | ISelectTeam;
  isSelected: boolean;
  onClick: (
    value: ISelectLabel | ISelectTeam
  ) => React.MouseEventHandler<HTMLButtonElement>;
}

interface ISelectTargetsProps {
  baseClass: string;
  queryIdForEdit: number | null;
  selectedTargets: ITarget[];
  targetedHosts: IHost[];
  targetedLabels: ILabel[];
  targetedTeams: ITeam[];
  goToQueryEditor: () => void;
  goToRunQuery: () => void;
  setSelectedTargets: React.Dispatch<React.SetStateAction<ITarget[]>>;
  setTargetedHosts: React.Dispatch<React.SetStateAction<IHost[]>>;
  setTargetedLabels: React.Dispatch<React.SetStateAction<ILabel[]>>;
  setTargetedTeams: React.Dispatch<React.SetStateAction<ITeam[]>>;

  setTargetsTotalCount: React.Dispatch<React.SetStateAction<number>>;
  targetsTotalCount: number; // why is this here?
}

interface ITargetsQueryKey {
  scope: string;
  query_id?: number | null;
  query?: string | null;
  selected?: ISelectedTargets | null;
}

const DEBOUNCE_DELAY = 500;
const STALE_TIME = 60000;

const isLabel = (entity: ISelectTargetsEntity) => "label_type" in entity;
const isHost = (entity: ISelectTargetsEntity) => "hostname" in entity;

const parseLabels = (list?: ILabelSummary[]) => {
  const all = list?.filter((l) => l.name === "All Hosts") || [];
  const platforms =
    list?.filter(
      (l) =>
        l.name === "macOS" || l.name === "MS Windows" || l.name === "All Linux"
    ) || [];
  const other = list?.filter((l) => l.label_type === "regular") || [];

  return { all, platforms, other };
};

const TargetPillSelector = ({
  entity,
  isSelected,
  onClick,
}: ITargetPillSelectorProps): JSX.Element => {
  const displayText = () => {
    switch (entity.name) {
      case "All Hosts":
        return "All hosts";
      case "All Linux":
        return "Linux";
      default:
        return entity.name || "Missing display name"; // TODO
    }
  };

  return (
    <button
      className="target-pill-selector"
      data-selected={isSelected}
      onClick={(e) => onClick(entity)(e)}
    >
      <img
        className={isSelected ? "check-icon" : "plus-icon"}
        alt=""
        src={isSelected ? CheckIcon : PlusIcon}
      />
      <span className="selector-name">{displayText()}</span>
      {/* <span className="selector-count">{entity.count}</span> */}
    </button>
  );
};

const SelectTargets = ({
  baseClass,
  queryIdForEdit,
  selectedTargets,
  targetedHosts,
  targetedLabels,
  targetedTeams,
  targetsTotalCount,
  goToQueryEditor,
  goToRunQuery,
  setSelectedTargets,
  setTargetedHosts,
  setTargetedLabels,
  setTargetedTeams,
  setTargetsTotalCount,
}: ISelectTargetsProps): JSX.Element => {
  const { isPremiumTier } = useContext(AppContext);
  const { selectedTargetsByQueryId, setSelectedTargetsByQueryId } = useContext(
    QueryContext
  );

  const [allHosts, setAllHosts] = useState<ILabelSummary[] | null>(null);
  const [platforms, setPlatforms] = useState<ILabelSummary[] | null>(null);
  const [otherLabels, setOtherLabels] = useState<ILabelSummary[] | null>(null);
  const [inputTabIndex, setInputTabIndex] = useState<number | null>(null);
  const [searchText, setSearchText] = useState<string>("");
  const [debouncedSearchText, setDebouncedSearchText] = useState<string>("");
  const [isDebouncing, setIsDebouncing] = useState<boolean>(false);

  const debounceSearch = useDebouncedCallback(
    (search: string) => {
      setDebouncedSearchText(search);
      setIsDebouncing(false);
    },
    DEBOUNCE_DELAY,
    { trailing: true }
  );

  useEffect(() => {
    setIsDebouncing(true);
    debounceSearch(searchText);
  }, [searchText]);

  // useEffect(() => {
  //   if (queryIdForEdit) {
  //     const selected = selectedTargetsByQueryId?.[queryIdForEdit];
  //     selected && setTargetedHosts([...selected.hosts]);
  //     selected && setTargetedLabels([...selected.labels]);
  //     selected && setTargetedTeams([...selected.teams]);
  //   }
  // });

  const {
    data: labels,
    error: errorLabels,
    isLoading: isLoadingLabels,
  } = useQuery<ILabelsSummaryResponse, Error, ILabelSummary[]>(
    ["labelsSummary"],
    labelsAPI.summary,
    {
      select: (data) => data.labels,
      staleTime: STALE_TIME, // TODO: confirm
    }
  );

  useEffect(() => {
    const parsed = parseLabels(labels);
    setAllHosts(parsed.all);
    setPlatforms(parsed.platforms);
    setOtherLabels(parsed.other);
  }, [labels]);

  const {
    data: teams,
    error: errorTeams,
    isLoading: isLoadingTeams,
  } = useQuery<ILoadTeamsResponse, Error, ITeam[]>(
    ["teams"],
    () => teamsAPI.loadAll(),
    {
      select: (data) => data.teams,
      enabled: isPremiumTier,
      staleTime: STALE_TIME, // TODO: confirm
    }
  );

  useEffect(() => {
    if (
      inputTabIndex === null &&
      allHosts &&
      platforms &&
      otherLabels &&
      teams
    ) {
      setInputTabIndex(
        allHosts.length +
          platforms.length +
          otherLabels.length +
          teams.length || 0
      );
    }
  }, [inputTabIndex, allHosts, platforms, otherLabels, teams]);

  const {
    data: searchResults,
    isFetching: isFetchingSearchResults,
    error: errorSearchResults,
  } = useQuery<ITargetsSearchResponse, Error, IHost[], ITargetsQueryKey[]>(
    [
      {
        scope: "targetsSearch", // TODO: shared scope?
        query_id: queryIdForEdit,
        query: debouncedSearchText,
        selected: formatSelectedTargetsForApi(selectedTargets),
      },
    ],
    ({ queryKey }) => {
      const { query_id, query, selected } = queryKey[0];
      return targetsAPI.search({
        query_id: query_id || null,
        query: query || "",
        selected_host_ids: selected?.hosts || null,
      });
    },
    {
      select: (data) => data.targets.hosts,
      enabled: !!debouncedSearchText,
      // staleTime: 5000, // TODO: try stale time if further performance optimizations are needed
    }
  );

  const { data: counts } = useQuery<
    ITargetsCountResponse,
    Error,
    ITargetsCountResponse,
    ITargetsQueryKey[]
  >(
    [
      {
        scope: "targetsCount", // Note: Scope is shared with QueryPage?
        query_id: queryIdForEdit,
        selected: formatSelectedTargetsForApi(selectedTargets),
      },
    ],
    ({ queryKey }) => {
      const { query_id, selected } = queryKey[0];
      return targetsAPI.count({ query_id, selected: selected || null });
    },
    {
      enabled: !!selectedTargets.length,
      onSuccess: (data) => {
        setTargetsTotalCount(data.targets_count || 0);
      },
      staleTime: STALE_TIME, // TODO: confirm
    }
  );

  useEffect(() => {
    const selected = [...targetedHosts, ...targetedLabels, ...targetedTeams];
    setSelectedTargets(selected);
    if (queryIdForEdit) {
      setSelectedTargetsByQueryId(
        queryIdForEdit,
        formatSelectedTargetsForApi(selected)
      );
    }
  }, [targetedHosts, targetedLabels, targetedTeams]);

  const handleClickCancel = () => {
    // setSelectedTargets([]);
    goToQueryEditor();
  };

  const handleButtonSelect = (selectedEntity: ISelectTargetsEntity) => (
    e: React.MouseEvent<HTMLButtonElement>
  ): void => {
    e.preventDefault();

    const prevTargets: ISelectTargetsEntity[] = isLabel(selectedEntity)
      ? targetedLabels
      : targetedTeams;

    // if the target was previously selected, we want to remove it now
    const newTargets = prevTargets.filter((t) => t.id !== selectedEntity.id);
    // if the length remains the same, the target was not previously selected so we want to add it now
    prevTargets.length === newTargets.length && newTargets.push(selectedEntity);

    isLabel(selectedEntity)
      ? setTargetedLabels(newTargets as ILabel[])
      : setTargetedTeams(newTargets as ITeam[]);
  };

  const handleRowSelect = (row: Row) => {
    const selectedHost = { ...row.original } as IHost;
    const newTargets = [...targetedHosts];

    newTargets.push(selectedHost);
    setTargetedHosts(newTargets);
    setSearchText("");
  };

  const handleRowRemove = (row: Row) => {
    const removedHost = { ...row.original } as IHost;
    const newTargets = targetedHosts.filter((t) => t.id !== removedHost.id);

    setTargetedHosts(newTargets);
  };

  // TODO: selections being saved but aren't rendering on initial mount?
  const renderTargetEntityList = (
    header: string,
    entityList: ISelectLabel[] | ISelectTeam[]
  ): JSX.Element => {
    return (
      <>
        {header && <h3>{header}</h3>}
        <div className="selector-block">
          {entityList?.map((entity: ISelectLabel | ISelectTeam) => {
            const targetList = isLabel(entity) ? targetedLabels : targetedTeams;
            return (
              <TargetPillSelector
                key={`${isLabel(entity) ? "label" : "team"}__${entity.id}`}
                entity={entity}
                isSelected={targetList.some((t) => t.id === entity.id)}
                onClick={handleButtonSelect}
              />
            );
          })}
        </div>
      </>
    );
  };

  const renderTargetsCount = (): JSX.Element | null => {
    if (!counts) {
      return null;
    }
    const { targets_count: total, targets_online: online } = counts;
    const onlinePercentage =
      targetsTotalCount > 0 ? Math.round((online / total) * 100) : 0;

    return (
      <>
        <span>{total}</span>&nbsp;hosts targeted&nbsp; ({onlinePercentage}
        %&nbsp;
        <TooltipWrapper
          tipContent={`
                Hosts are online if they<br /> have recently checked <br />into Fleet`}
        >
          online
        </TooltipWrapper>
        ){" "}
      </>
    );
  };

  if (isLoadingLabels || (isPremiumTier && isLoadingTeams)) {
    return (
      <div className={`${baseClass}__wrapper body-wrap`}>
        <h1>Select targets</h1>
        <div className={`${baseClass}__page-loading`}>
          <Spinner />
        </div>
      </div>
    );
  }

  if (errorLabels || errorTeams) {
    return (
      <div className={`${baseClass}__wrapper body-wrap`}>
        <h1>Select targets</h1>
        <PageError />
      </div>
    );
  }

  return (
    <div className={`${baseClass}__wrapper body-wrap`}>
      <h1>Select targets</h1>
      <div className={`${baseClass}__target-selectors`}>
        {!!allHosts?.length && renderTargetEntityList("", allHosts)}
        {!!platforms?.length && renderTargetEntityList("Platforms", platforms)}
        {!!teams?.length && renderTargetEntityList("Teams", teams)}
        {!!otherLabels?.length && renderTargetEntityList("Labels", otherLabels)}
      </div>
      <TargetsInput
        tabIndex={inputTabIndex || 0}
        searchText={searchText}
        searchResults={searchResults ? [...searchResults] : []} // TODO: why spread?
        isTargetsLoading={isFetchingSearchResults || isDebouncing}
        targetedHosts={[...targetedHosts]} // TODO: why spread?
        hasFetchError={!!errorSearchResults}
        setSearchText={setSearchText}
        handleRowSelect={handleRowSelect}
        handleRowRemove={handleRowRemove}
      />
      <div className={`${baseClass}__targets-button-wrap`}>
        <Button
          className={`${baseClass}__btn`}
          onClick={handleClickCancel}
          variant="text-link"
        >
          Cancel
        </Button>
        <Button
          className={`${baseClass}__btn`}
          type="button"
          variant="blue-green"
          disabled={!targetsTotalCount} // TODO: confirm
          onClick={goToRunQuery}
        >
          Run
        </Button>
        <div className={`${baseClass}__targets-total-count`}>
          {renderTargetsCount()}
        </div>
      </div>
    </div>
  );
};

export default SelectTargets;
