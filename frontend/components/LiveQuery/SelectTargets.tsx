import React, { useContext, useEffect, useState } from "react";
import { Row } from "react-table";
import { useQuery } from "react-query";
import { useDebouncedCallback } from "use-debounce";

import { AppContext } from "context/app";

import { IHost } from "interfaces/host";
import { ILabel, ILabelSummary } from "interfaces/label";
import {
  ITarget,
  ISelectLabel,
  ISelectTeam,
  ISelectTargetsEntity,
  ISelectedTargetsForApi,
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
import TargetsInput from "components/LiveQuery/TargetsInput";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
import TooltipWrapper from "components/TooltipWrapper";
import Icon from "components/Icon";

interface ITargetPillSelectorProps {
  entity: ISelectLabel | ISelectTeam;
  isSelected: boolean;
  onClick: (
    value: ISelectLabel | ISelectTeam
  ) => React.MouseEventHandler<HTMLButtonElement>;
}

interface ISelectTargetsProps {
  baseClass: string;
  queryId?: number | null;
  selectedTargets: ITarget[];
  targetedHosts: IHost[];
  targetedLabels: ILabel[];
  targetedTeams: ITeam[];
  goToQueryEditor: () => void;
  goToRunQuery: () => void;
  setSelectedTargets: // TODO: Refactor policy targets to streamline selectedTargets/selectedTargetsByType
  | React.Dispatch<React.SetStateAction<ITarget[]>> // Used for policies page level useState hook
    | ((value: ITarget[]) => void); // Used for queries app level QueryContext
  setTargetedHosts: React.Dispatch<React.SetStateAction<IHost[]>>;
  setTargetedLabels: React.Dispatch<React.SetStateAction<ILabel[]>>;
  setTargetedTeams: React.Dispatch<React.SetStateAction<ITeam[]>>;
  setTargetsTotalCount: React.Dispatch<React.SetStateAction<number>>;
}

interface ILabelsByType {
  allHosts: ILabelSummary[];
  platforms: ILabelSummary[];
  other: ILabelSummary[];
}

interface ITargetsQueryKey {
  scope: string;
  query_id?: number | null;
  query?: string | null;
  selected?: ISelectedTargetsForApi | null;
}

const DEBOUNCE_DELAY = 500;
const STALE_TIME = 60000;

const isLabel = (entity: ISelectTargetsEntity) => "label_type" in entity;
const isAllHosts = (entity: ISelectTargetsEntity) =>
  "label_type" in entity &&
  entity.name === "All Hosts" &&
  entity.label_type === "builtin";

const parseLabels = (list?: ILabelSummary[]) => {
  const allHosts = list?.filter((l) => l.name === "All Hosts") || [];
  const platforms =
    list?.filter(
      (l) =>
        l.name === "macOS" ||
        l.name === "MS Windows" ||
        l.name === "All Linux" ||
        l.name === "chrome"
    ) || [];
  const other = list?.filter((l) => l.label_type === "regular") || [];

  return { allHosts, platforms, other };
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
      case "chrome":
        return "ChromeOS";
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
      <Icon name={isSelected ? "check" : "plus"} />
      <span className="selector-name">{displayText()}</span>
      {/* <span className="selector-count">{entity.count}</span> */}
    </button>
  );
};

const SelectTargets = ({
  baseClass,
  queryId,
  selectedTargets,
  targetedHosts,
  targetedLabels,
  targetedTeams,
  goToQueryEditor,
  goToRunQuery,
  setSelectedTargets,
  setTargetedHosts,
  setTargetedLabels,
  setTargetedTeams,
  setTargetsTotalCount,
}: ISelectTargetsProps): JSX.Element => {
  const { isPremiumTier } = useContext(AppContext);

  const [labels, setLabels] = useState<ILabelsByType | null>(null);
  const [inputTabIndex, setInputTabIndex] = useState<number | null>(null);
  const [searchText, setSearchText] = useState("");
  const [debouncedSearchText, setDebouncedSearchText] = useState("");
  const [isDebouncing, setIsDebouncing] = useState(false);

  const debounceSearch = useDebouncedCallback(
    (search: string) => {
      setDebouncedSearchText(search);
      setIsDebouncing(false);
    },
    DEBOUNCE_DELAY,
    { trailing: true }
  );

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

  const {
    data: labelsSummary,
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

  const {
    data: searchResults,
    isFetching: isFetchingSearchResults,
    error: errorSearchResults,
  } = useQuery<ITargetsSearchResponse, Error, IHost[], ITargetsQueryKey[]>(
    [
      {
        scope: "targetsSearch", // TODO: shared scope?
        query_id: queryId,
        query: debouncedSearchText,
        selected: formatSelectedTargetsForApi(selectedTargets),
      },
    ],
    ({ queryKey }) => {
      const { query_id, query, selected } = queryKey[0];
      return targetsAPI.search({
        query_id: query_id || null,
        query: query || "",
        excluded_host_ids: selected?.hosts || null,
      });
    },
    {
      select: (data) => data.hosts,
      enabled: !!debouncedSearchText,
      // staleTime: 5000, // TODO: try stale time if further performance optimizations are needed
    }
  );

  const {
    data: counts,
    error: errorCounts,
    isFetching: isFetchingCounts,
  } = useQuery<
    ITargetsCountResponse,
    Error,
    ITargetsCountResponse,
    ITargetsQueryKey[]
  >(
    [
      {
        scope: "targetsCount", // Note: Scope is shared with QueryPage?
        query_id: queryId,
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
  }, [targetedHosts, targetedLabels, targetedTeams]);

  useEffect(() => {
    labelsSummary && setLabels(parseLabels(labelsSummary));
  }, [labelsSummary]);

  useEffect(() => {
    if (inputTabIndex === null && labelsSummary && teams) {
      setInputTabIndex(labelsSummary.length + teams.length || 0);
    }
  }, [inputTabIndex, labelsSummary, teams]);

  useEffect(() => {
    setIsDebouncing(true);
    debounceSearch(searchText);
  }, [searchText]);

  const handleClickCancel = () => {
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
    let newTargets = prevTargets.filter((t) => t.id !== selectedEntity.id);
    // if the length remains the same, the target was not previously selected so we want to add it now
    prevTargets.length === newTargets.length && newTargets.push(selectedEntity);

    // Logic when to deselect/select "all hosts" when using more granulated filters
    // If "all hosts" is selected
    if (isAllHosts(selectedEntity)) {
      // and "all hosts" is already selected, deselect it
      if (targetedLabels.some((t) => isAllHosts(t))) {
        newTargets = [];
      } // else deselect everything but "all hosts"
      else {
        newTargets = [selectedEntity];
      }
      setTargetedTeams([]);
      setTargetedHosts([]);
    }
    // else deselect "all hosts"
    else {
      if (targetedLabels.some((t) => isAllHosts(t))) {
        setTargetedLabels([]);
      }
      newTargets = newTargets.filter((t) => !isAllHosts(t));
    }

    isLabel(selectedEntity)
      ? setTargetedLabels(newTargets as ILabel[])
      : setTargetedTeams(newTargets as ITeam[]);
  };

  const handleRowSelect = (row: Row) => {
    const selectedHost = row.original as IHost;
    setTargetedHosts((prevHosts) => prevHosts.concat(selectedHost));
    setSearchText("");

    // If "all hosts" is already selected when using host target picker, deselect "all hosts"
    if (targetedLabels.some((t) => isAllHosts(t))) {
      setTargetedLabels([]);
    }
  };

  const handleRowRemove = (row: Row) => {
    const removedHost = row.original as IHost;
    setTargetedHosts((prevHosts) =>
      prevHosts.filter((h) => h.id !== removedHost.id)
    );
  };

  const onClickRun = () => {
    setTargetsTotalCount(counts?.targets_count || 0);
    goToRunQuery();
  };

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
    if (isFetchingCounts) {
      return (
        <>
          <Spinner
            size="x-small"
            includeContainer={false}
            centered={false}
            className={`${baseClass}__count-spinner`}
          />
          <i style={{ color: "#8b8fa2" }}>Counting hosts</i>
        </>
      );
    }

    if (errorCounts) {
      return (
        <b style={{ color: "#d66c7b", margin: 0 }}>
          There was a problem counting hosts. Please try again later.
        </b>
      );
    }

    if (!counts) {
      return null;
    }

    const { targets_count: total, targets_online: online } = counts;
    const onlinePercentage = () => {
      if (total === 0) {
        return 0;
      }
      // If at least 1 host is online, displays <1% instead of 0%
      const roundPercentage =
        Math.round((online / total) * 100) === 0
          ? "<1"
          : Math.round((online / total) * 100);
      return roundPercentage;
    };

    return (
      <>
        <b>{total.toLocaleString()}</b>&nbsp;host{total > 1 ? `s` : ``}{" "}
        targeted&nbsp; ({onlinePercentage()}
        %&nbsp;
        <TooltipWrapper
          tipContent={
            <>
              Hosts are online if they <br />
              have recently checked <br />
              into Fleet.
            </>
          }
        >
          online
        </TooltipWrapper>
        ){" "}
      </>
    );
  };

  if (isLoadingLabels || (isPremiumTier && isLoadingTeams)) {
    return (
      <div className={`${baseClass}__wrapper`}>
        <h1>Select targets</h1>
        <div className={`${baseClass}__page-loading`}>
          <Spinner />
        </div>
      </div>
    );
  }

  if (errorLabels || errorTeams) {
    return (
      <div className={`${baseClass}__wrapper`}>
        <h1>Select targets</h1>
        <PageError />
      </div>
    );
  }

  return (
    <div className={`${baseClass}__wrapper`}>
      <h1>Select targets</h1>
      <div className={`${baseClass}__target-selectors`}>
        {!!labels?.allHosts.length &&
          renderTargetEntityList("", labels.allHosts)}
        {!!labels?.platforms?.length &&
          renderTargetEntityList("Platforms", labels.platforms)}
        {!!teams?.length && renderTargetEntityList("Teams", teams)}
        {!!labels?.other?.length &&
          renderTargetEntityList("Labels", labels.other)}
      </div>
      <TargetsInput
        tabIndex={inputTabIndex || 0}
        searchText={searchText}
        searchResults={searchResults || []}
        isTargetsLoading={isFetchingSearchResults || isDebouncing}
        targetedHosts={targetedHosts}
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
          disabled={isFetchingCounts || !counts?.targets_count} // TODO: confirm
          onClick={onClickRun}
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
