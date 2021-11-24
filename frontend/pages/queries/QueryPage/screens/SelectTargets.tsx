import React, { useState } from "react";
import { useQuery } from "react-query";
import { Row } from "react-table";
import { filter, forEach, isEmpty, remove, unionWith } from "lodash";

// @ts-ignore
import { formatSelectedTargetsForApi } from "fleet/helpers";
import targetsAPI from "services/entities/targets";
import { ITarget, ITargets, ITargetsAPIResponse } from "interfaces/target";
import { ILabel } from "interfaces/label";
import { ITeam } from "interfaces/team";
import { IHost } from "interfaces/host";

// @ts-ignore
import TargetsInput from "components/TargetsInput";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
import PlusIcon from "../../../../../assets/images/icon-plus-purple-32x32@2x.png";
import CheckIcon from "../../../../../assets/images/icon-check-purple-32x32@2x.png";
import ExternalURLIcon from "../../../../../assets/images/icon-external-url-12x12@2x.png";
import ErrorIcon from "../../../../../assets/images/icon-error-16x16@2x.png";

interface ISelectHost extends IHost {
  target_type?: string;
}

interface ISelectLabel extends ILabel {
  target_type?: string;
}

interface ISelectTeam extends ITeam {
  target_type?: string;
}

type ISelectTargetsEntity = ISelectHost | ISelectLabel | ISelectTeam;

interface ITargetPillSelectorProps {
  entity: ISelectLabel | ISelectTeam;
  isSelected: boolean;
  onClick: (
    value: ISelectLabel | ISelectTeam
  ) => React.MouseEventHandler<HTMLButtonElement>;
}

interface ISelectTargetsProps {
  baseClass: string;
  selectedTargets: ISelectTargetsEntity[];
  queryIdForEdit: number | null;
  goToQueryEditor: () => void;
  goToRunQuery: () => void;
  setSelectedTargets: React.Dispatch<React.SetStateAction<ITarget[]>>;
}

interface IModifiedUseQueryTargetsResponse {
  results: IHost[] | ITargets;
  targetsCount: number;
  onlineCount: number;
}

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
        return entity.display_text;
    }
  };

  return (
    <button
      className="target-pill-selector"
      data-selected={isSelected}
      onClick={(e) => onClick(entity)(e)}
    >
      <img alt="" src={isSelected ? CheckIcon : PlusIcon} />
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
}: ISelectTargetsProps) => {
  const [targetsTotalCount, setTargetsTotalCount] = useState<number | null>(
    null
  );
  const [targetsOnlinePercent, setTargetsOnlinePercent] = useState<number>(0);
  const [allHostsLabels, setAllHostsLabels] = useState<ILabel[] | null>(null);
  const [platformLabels, setPlatformLabels] = useState<ILabel[] | null>(null);
  const [teams, setTeams] = useState<ITeam[] | null>(null);
  const [otherLabels, setOtherLabels] = useState<ILabel[] | null>(null);
  const [selectedLabels, setSelectedLabels] = useState<ISelectTargetsEntity[]>(
    []
  );
  const [inputTabIndex, setInputTabIndex] = useState<number>(0);
  const [searchText, setSearchText] = useState<string>("");
  const [relatedHosts, setRelatedHosts] = useState<IHost[]>([]);

  const { isLoading: isTargetsLoading, isError: isTargetsError } = useQuery(
    // triggers query on change
    ["targetsFromSearch", searchText, [...selectedTargets]],
    () =>
      targetsAPI.loadAll({
        query: searchText,
        queryId: queryIdForEdit,
        selected: formatSelectedTargetsForApi(selectedTargets) as any,
      }),
    {
      refetchOnWindowFocus: false,

      // only retrieve the whole targets object once
      // we will only update related hosts when a search query fires
      select: (data: ITargetsAPIResponse) =>
        allHostsLabels
          ? {
              results: data.targets.hosts,
              targetsCount: data.targets_count,
              onlineCount: data.targets_online,
            }
          : {
              results: data.targets,
              targetsCount: data.targets_count,
              onlineCount: data.targets_online,
            },
      onSuccess: ({
        results,
        targetsCount,
        onlineCount,
      }: IModifiedUseQueryTargetsResponse) => {
        if ("labels" in results) {
          // this will only run once
          const { labels, teams: targetTeams } = results as ITargets;
          const allHosts = filter(
            labels,
            ({ display_text: text }) => text === "All Hosts"
          );
          const platforms = filter(
            labels,
            ({ display_text: text }) =>
              text === "macOS" || text === "MS Windows" || text === "All Linux"
          );
          const other = filter(
            labels,
            ({ label_type: type }) => type === "regular"
          );

          setAllHostsLabels(allHosts);
          setPlatformLabels(platforms);
          setTeams(targetTeams);
          setOtherLabels(other);

          const labelCount =
            allHosts.length +
            platforms.length +
            targetTeams.length +
            other.length;
          setInputTabIndex(labelCount || 0);
        } else if (searchText === "") {
          setRelatedHosts([]);
        } else {
          // this will always update as the user types
          setRelatedHosts([...results] as IHost[]);
        }

        setTargetsTotalCount(targetsCount);
        if (targetsCount > 0) {
          setTargetsOnlinePercent(
            Math.round((onlineCount / targetsCount) * 100)
          );
        }
      },
    }
  );

  const handleSelectedLabels = (selectedLabel: ISelectTargetsEntity) => (
    e: React.MouseEvent<HTMLButtonElement>
  ): void => {
    e.preventDefault();

    let targets = selectedTargets;
    const labels = selectedLabels;
    const removed = remove(labels, (label) =>
      isSameSelectTargetsEntity(label, selectedLabel)
    );

    // visually show selection
    const isRemoval = removed.length > 0;
    if (isRemoval) {
      targets = targets.filter(
        (target) => !isSameSelectTargetsEntity(target, selectedLabel)
      );
    } else {
      labels.push(selectedLabel);

      // prepare the labels data
      forEach(labels, (label) => {
        label.target_type = "label_type" in label ? "labels" : "teams";
      });

      targets = unionWith(targets, labels, isSameSelectTargetsEntity);
    }

    setSelectedLabels([...labels]);
    setSelectedTargets([...targets]);
  };

  const handleRowSelect = (row: Row) => {
    const targets = selectedTargets;
    const hostTarget = row.original as any; // intentional so we can add to the object

    hostTarget.target_type = "hosts";

    targets.push(hostTarget as IHost);
    setSelectedTargets([...targets]);
    setSearchText("");
  };

  const handleRowRemove = (row: Row) => {
    const targets = selectedTargets;
    const hostTarget = row.original as ITarget;
    remove(targets, (t) => t.id === hostTarget.id && t.target_type === "hosts");

    setSelectedTargets([...targets]);
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
              isSelected={selectedLabels.some((label: ISelectTargetsEntity) =>
                isSameSelectTargetsEntity(label, entity)
              )}
              onClick={handleSelectedLabels}
            />
          ))}
        </div>
      </>
    );
  };

  if (isEmpty(searchText) && isTargetsLoading) {
    return (
      <div className={`${baseClass}__wrapper body-wrap`}>
        <h1>Select targets</h1>
        <div className={`${baseClass}__page-loading`}>
          <Spinner />
        </div>
      </div>
    );
  }

  if (isEmpty(searchText) && isTargetsError) {
    return (
      <div className={`${baseClass}__wrapper body-wrap`}>
        <h1>Select targets</h1>
        <div className={`${baseClass}__page-error`}>
          <h4>
            <img alt="" src={ErrorIcon} />
            Something&apos;s gone wrong.
          </h4>
          <p>Refresh the page or log in again.</p>
          <p>
            If this keeps happening please{" "}
            <a
              className="file-issue-link"
              target="_blank"
              rel="noopener noreferrer"
              href="https://github.com/fleetdm/fleet/issues/new/choose"
            >
              file an issue <img alt="" src={ExternalURLIcon} />
            </a>
          </p>
        </div>
      </div>
    );
  }

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
        tabIndex={inputTabIndex}
        searchText={searchText}
        relatedHosts={[...relatedHosts]}
        isTargetsLoading={isTargetsLoading}
        selectedTargets={[...selectedTargets]}
        hasFetchError={isTargetsError}
        setSearchText={setSearchText}
        handleRowSelect={handleRowSelect}
        handleRowRemove={handleRowRemove}
      />
      <div className={`${baseClass}__targets-button-wrap`}>
        <Button
          className={`${baseClass}__btn`}
          onClick={goToQueryEditor}
          variant="text-link"
        >
          Cancel
        </Button>
        <Button
          className={`${baseClass}__btn`}
          type="button"
          variant="blue-green"
          disabled={!targetsTotalCount}
          onClick={goToRunQuery}
        >
          Run
        </Button>
        <div className={`${baseClass}__targets-total-count`}>
          {!!targetsTotalCount && (
            <>
              <span>{targetsTotalCount}</span> targets selected&nbsp; (
              {targetsOnlinePercent}% online)
            </>
          )}
        </div>
      </div>
    </div>
  );
};

export default SelectTargets;
