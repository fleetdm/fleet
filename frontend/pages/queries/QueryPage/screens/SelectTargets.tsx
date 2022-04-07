import React, { useCallback, useContext, useEffect, useState } from "react";
import { Row } from "react-table";
import { useQuery } from "react-query";
import { filter, forEach, isEmpty, remove, unionWith } from "lodash";
import { useDebouncedCallback } from "use-debounce/lib";
import { v4 as uuidv4 } from "uuid";

import { formatSelectedTargetsForApi } from "fleet/helpers";
import { QueryContext } from "context/query";
import useQueryTargets, { ITargetsQueryResponse } from "hooks/useQueryTargets";
import target, {
  ITarget,
  ISelectLabel,
  ISelectTeam,
  ISelectTargetsEntity,
  ISelectedTargets,
} from "interfaces/target";
import { ILabel } from "interfaces/label";
import { ITeam } from "interfaces/team";
import { IHost } from "interfaces/host";
import targetsAPI, { ITargetsCount } from "services/entities/targets";
import teamsAPI from "services/entities/teams";

// @ts-ignore
import TargetsInput from "components/TargetsInput";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
import TooltipWrapper from "components/TooltipWrapper";
import PlusIcon from "../../../../../assets/images/icon-plus-purple-32x32@2x.png";
import CheckIcon from "../../../../../assets/images/icon-check-purple-32x32@2x.png";
import ExternalURLIcon from "../../../../../assets/images/icon-external-url-12x12@2x.png";
import ErrorIcon from "../../../../../assets/images/icon-error-16x16@2x.png";

interface ITargetPillSelectorProps {
  entity: ISelectLabel | ISelectTeam;
  isSelected: boolean;
  onClick: (
    value: ISelectLabel | ISelectTeam
  ) => React.MouseEventHandler<HTMLButtonElement>;
}

interface ISelectTargetsProps {
  baseClass: string;
  selectedTargets: ITarget[];
  queryIdForEdit: number | null;
  goToQueryEditor: () => void;
  goToRunQuery: () => void;
  setSelectedTargets: React.Dispatch<React.SetStateAction<ITarget[]>>;
  setTargetsTotalCount: React.Dispatch<React.SetStateAction<number>>;
  targetsTotalCount: number;
}

const DEBOUNCE_DELAY = 500;
const STALE_TIME = 60000;

const isSameSelectTargetsEntity = (
  e1: ISelectTargetsEntity,
  e2: ISelectTargetsEntity
) => e1.id === e2.id && e1.target_type === e2.target_type;

const TargetPillSelector = ({
  entity,
  isSelected,
  onClick,
}: ITargetPillSelectorProps): JSX.Element => {
  const displayText = () => {
    switch (entity.display_text) {
      case "All Hosts":
        return "All hosts";
      case "All Linux":
        return "Linux";
      default:
        return entity.display_text || entity.name || "Missing display name"; // TODO
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
      <span className="selector-count">{entity.count}</span>
    </button>
  );
};

const SelectTargets = ({
  baseClass,
  selectedTargets,
  queryIdForEdit,
  goToQueryEditor,
  goToRunQuery,
  setSelectedTargets,
  setTargetsTotalCount,
  targetsTotalCount,
}: ISelectTargetsProps): JSX.Element => {
  const { selectedTargetsByQueryId, setSelectedTargetsByQueryId } = useContext(
    QueryContext
  );

  const [allHostsLabels, setAllHostsLabels] = useState<ILabel[] | null>(null);
  const [platformLabels, setPlatformLabels] = useState<ILabel[] | null>(null);
  const [teams, setTeams] = useState<ITeam[] | null>(null);
  const [otherLabels, setOtherLabels] = useState<ILabel[] | null>(null);
  const [selectedLabels, setSelectedLabels] = useState<ISelectTargetsEntity[]>(
    []
  );
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
    setSelectedTargetsByQueryId(queryIdForEdit || 0, {
      hosts: [200],
      labels: [6, 10],
      teams: [1],
    });
    return setSelectedTargetsByQueryId(queryIdForEdit || 0, {
      hosts: [200],
      labels: [6, 10],
      teams: [1],
    });
  }, []);

  useEffect(() => {
    setIsDebouncing(true);
    debounceSearch(searchText);
  }, [searchText]);

  const parseLabels = (labels: ILabel[]) => {
    const all = filter(
      labels,
      ({ display_text: text }) => text === "All Hosts"
    ).map((label) => ({ ...label, uuid: uuidv4() }));

    const platform = filter(
      labels,
      ({ display_text: text }) =>
        text === "macOS" || text === "MS Windows" || text === "All Linux"
    ).map((label) => ({ ...label, uuid: uuidv4() }));

    const other = filter(
      labels,
      ({ label_type: type }) => type === "regular"
    ).map((label) => ({ ...label, uuid: uuidv4() }));

    return {
      all,
      platform,
      other,
    };
  };

  interface ILabelSummary {
    id: number;
    name: string;
    label_type: string;
    display_text?: string;
  }

  const {
    data: labels,
    isFetching: isFetchingLabels,
    error: errorLabels,
  } = useQuery<Record<"labels", ILabel[]>, Error>(
    [
      {
        scope: "labelsSummary", // TODO: shared scope?
      },
    ],
    targetsAPI.labels,
    {
      onSuccess: (data) => {
        const { all, platform, other } = parseLabels(data.labels);
        setAllHostsLabels(all || []);
        setPlatformLabels(platform || []);
        setOtherLabels(other || []);
      },
    }
  );

  useQuery<Record<"teams", ITeam[]>>(["teams"], () => teamsAPI.loadAll(), {
    onSuccess: (data) => {
      setTeams(data.teams.map((team) => ({ ...team, uuid: uuidv4() })));
    },
  });

  useEffect(() => {
    if (
      inputTabIndex === null &&
      allHostsLabels &&
      platformLabels &&
      otherLabels &&
      teams
    ) {
      setInputTabIndex(
        allHostsLabels.length +
          platformLabels.length +
          otherLabels.length +
          teams.length || 0
      );
    }
  }, [inputTabIndex, allHostsLabels, platformLabels, otherLabels, teams]);

  const {
    data: searchResults,
    isFetching: isFetchingSearchResults,
    error: errorSearchResults,
  } = useQuery<
    Record<"hosts", IHost[]>,
    Error,
    IHost[],
    {
      scope: "targetsSearch";
      search: string;
      // selected: ISelectedTargets | null; // TODO: Do we need to filter out hosts in preselected labels/teams?
    }[]
  >(
    [
      {
        scope: "targetsSearch", // TODO: shared scope?
        search: searchText,
        // selected: selectedTargetsByQueryId?.[queryIdForEdit || 0] || null,
      },
    ],
    ({ queryKey }) => targetsAPI.search(queryKey[0].search),
    {
      select: (data) => data.hosts,
    }
  );

  interface ITargetsQueryKey {
    scope: string;
    queryId: number | null;
    selected: ISelectedTargets | null;
  }

  const {
    data: counts,
    isFetching: isFetchingCounts,
    error: errorCounts,
  } = useQuery<ITargetsCount, Error, ITargetsCount, ITargetsQueryKey[]>(
    [
      {
        scope: "targetsCount", // Note: Scope is shared with QueryPage?
        queryId: queryIdForEdit, // TODO: How is this used by the backend? Can this be removed?
        // selected: selectedTargetsByQueryId?.[queryIdForEdit || 0] || null,
        selected: formatSelectedTargetsForApi(selectedTargets),
      },
    ],
    ({ queryKey }) => targetsAPI.count(queryKey[0].selected),
    {
      onSuccess: (data) => {
        setTargetsTotalCount(data.targets_count || 0);
      },
    }
  );

  const handleClickCancel = () => {
    // setSelectedTargets([]);
    goToQueryEditor();
  };

  const handleSelectedLabels = (selectedLabel: ISelectTargetsEntity) => (
    e: React.MouseEvent<HTMLButtonElement>
  ): void => {
    e.preventDefault();

    let newTargets = selectedTargets;
    const newLabels = selectedLabels;
    const removed = remove(newLabels, (label) =>
      isSameSelectTargetsEntity(label, selectedLabel)
    );

    // visually show selection
    const isRemoval = removed.length > 0;
    if (isRemoval) {
      newTargets = newTargets.filter(
        (t) => !isSameSelectTargetsEntity(t, selectedLabel)
      );
    } else {
      newLabels.push(selectedLabel);

      // prepare the labels data
      forEach(newLabels, (label) => {
        label.target_type = "label_type" in label ? "labels" : "teams";
      });

      newTargets = unionWith(newTargets, newLabels, isSameSelectTargetsEntity);
    }

    setSelectedLabels([...newLabels]);
    setSelectedTargets([...newTargets]);
  };

  const handleRowSelect = (row: Row) => {
    const newTargets = selectedTargets;
    const hostTarget = row.original as any; // intentional so we can add to the object

    hostTarget.target_type = "hosts";

    newTargets.push(hostTarget as IHost);
    setSelectedTargets([...newTargets]);
    setSearchText("");
  };

  const handleRowRemove = (row: Row) => {
    const newTargets = selectedTargets;
    const hostTarget = row.original as ITarget;
    remove(newTargets, (t) => t.id === hostTarget.id && "hostname" in t); // check if `hostname` is present to confirm target is of type `IHost`

    setSelectedTargets([...newTargets]);
  };

  const renderTargetEntityList = (
    header: string,
    entityList: ISelectLabel[] | ISelectTeam[]
  ): JSX.Element => {
    return (
      <>
        {header && <h3>{header}</h3>}
        <div className="selector-block">
          {entityList?.map((entity: ISelectLabel | ISelectTeam) => (
            <TargetPillSelector
              key={`target_${entity.target_type}_${entity.id}`}
              entity={entity}
              isSelected={selectedLabels.some((label) =>
                isSameSelectTargetsEntity(label, entity)
              )}
              onClick={handleSelectedLabels}
            />
          ))}
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

  // if (!isTargetsError && isEmpty(searchText) && !allHostsLabels) {
  //   return (
  //     <div className={`${baseClass}__wrapper body-wrap`}>
  //       <h1>Select targets</h1>
  //       <div className={`${baseClass}__page-loading`}>
  //         <Spinner />
  //       </div>
  //     </div>
  //   );
  // }

  if (inputTabIndex === null) {
    return (
      <div className={`${baseClass}__wrapper body-wrap`}>
        <h1>Select targets</h1>
        <div className={`${baseClass}__page-loading`}>
          <Spinner />
        </div>
      </div>
    );
  }

  // if (isEmpty(searchText) && isTargetsError) {
  //   return (
  //     <div className={`${baseClass}__wrapper body-wrap`}>
  //       <h1>Select targets</h1>
  //       <div className={`${baseClass}__page-error`}>
  //         <h4>
  //           <img alt="" src={ErrorIcon} />
  //           Something&apos;s gone wrong.
  //         </h4>
  //         <p>Refresh the page or log in again.</p>
  //         <p>
  //           If this keeps happening please{" "}
  //           <a
  //             className="file-issue-link"
  //             target="_blank"
  //             rel="noopener noreferrer"
  //             href="https://github.com/fleetdm/fleet/issues/new/choose"
  //           >
  //             file an issue <img alt="" src={ExternalURLIcon} />
  //           </a>
  //         </p>
  //       </div>
  //     </div>
  //   );
  // }

  return (
    <div className={`${baseClass}__wrapper body-wrap`}>
      <h1>Select targets</h1>
      <div className={`${baseClass}__target-selectors`}>
        {allHostsLabels &&
          allHostsLabels.length > 0 &&
          renderTargetEntityList("", allHostsLabels)}
        {platformLabels &&
          platformLabels.length > 0 &&
          renderTargetEntityList("Platforms", platformLabels)}
        {teams && teams.length > 0 && renderTargetEntityList("Teams", teams)}
        {otherLabels &&
          otherLabels.length > 0 &&
          renderTargetEntityList("Labels", otherLabels)}
      </div>
      <TargetsInput
        tabIndex={inputTabIndex || 0}
        searchText={searchText}
        relatedHosts={searchResults || []}
        isTargetsLoading={isFetchingSearchResults || isDebouncing}
        selectedTargets={[...selectedTargets]}
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
          // disabled={!searchResults?.targetsTotalCount}
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
