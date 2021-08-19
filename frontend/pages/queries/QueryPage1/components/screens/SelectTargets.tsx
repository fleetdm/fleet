import React, { useState } from "react";
import { Dispatch } from "redux";
import { useQuery } from "react-query";
import { Row } from "react-table";
import { forEach, isEmpty, reduce, remove, unionBy } from "lodash";

import {
  setSelectedTargets,
  setSelectedTargetsQuery, // @ts-ignore
} from "redux/nodes/components/QueryPages/actions";
import { formatSelectedTargetsForApi } from "fleet/helpers";
import targetsAPI from "services/entities/targets";
import { ITarget, ITargets, ITargetsAPIResponse } from "interfaces/target";
import { ICampaign } from "interfaces/campaign";
import { ILabel } from "interfaces/label";
import { ITeam } from "interfaces/team";
import { IHost } from "interfaces/host";
import { useDeepEffect } from "utilities/hooks";

// @ts-ignore
import TargetsInput from "pages/queries/QueryPage1/components/TargetsInput";
import Button from "components/buttons/Button";
import PlusIcon from "../../../../../../assets/images/icon-plus-purple-32x32@2x.png";
import CheckIcon from "../../../../../../assets/images/icon-check-purple-32x32@2x.png";

interface ITargetPillSelectorProps {
  entity: ILabel | ITeam;
  isSelected: boolean;
  onClick: (
    value: ILabel | ITeam
  ) => React.MouseEventHandler<HTMLButtonElement>;
}

interface ISelectTargetsProps {
  baseClass: string;
  selectedTargets: ITarget[];
  campaign: ICampaign | null;
  isBasicTier: boolean;
  queryIdForEdit: string | undefined;
  goToQueryEditor: () => void;
  goToRunQuery: () => void;
  dispatch: Dispatch;
}

interface IModifiedUseQueryTargetsResponse {
  results: IHost[] | ITargets;
  targetsCount: number;
  onlineCount: number;
}

const TargetPillSelector = ({
  entity,
  isSelected,
  onClick,
}: ITargetPillSelectorProps): JSX.Element => (
  <button
    className="target-pill-selector"
    data-selected={isSelected}
    onClick={(e) => onClick(entity)(e)}
  >
    <img alt="" src={isSelected ? CheckIcon : PlusIcon} />
    <span className="selector-name">{entity.display_text}</span>
    <span className="selector-count">{entity.count}</span>
  </button>
);

const SelectTargets = ({
  baseClass,
  selectedTargets,
  campaign,
  isBasicTier,
  queryIdForEdit,
  goToQueryEditor,
  goToRunQuery,
  dispatch,
}: ISelectTargetsProps) => {
  const [targetsTotalCount, setTargetsTotalCount] = useState<number | null>(
    null
  );
  const [targetsOnlinePercent, setTargetsOnlinePercent] = useState<number>(0);
  const [targetsError, setTargetsError] = useState<string | null>(null);
  const [allHostsLabels, setAllHostsLabels] = useState<ILabel[] | null>(null);
  const [platformLabels, setPlatformLabels] = useState<ILabel[] | null>(null);
  const [linuxLabels, setLinuxLabels] = useState<ILabel[] | null>(null);
  const [teams, setTeams] = useState<ITeam[] | null>(null);
  const [otherLabels, setOtherLabels] = useState<ILabel[] | null>(null);
  const [selectedLabels, setSelectedLabels] = useState<any>([]);
  const [inputTabIndex, setInputTabIndex] = useState<number>(0);
  const [searchText, setSearchText] = useState<string>("");
  const [relatedHosts, setRelatedHosts] = useState<IHost[]>([]);

  const { status } = useQuery(
    ["targetsFromSearch", searchText, [...selectedTargets]], // triggers query on change
    () =>
      targetsAPI.loadAll({
        query: searchText,
        queryId: null,
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
          const allHosts = remove(
            labels,
            ({ display_text: text }) => text === "All Hosts"
          );
          const platforms = remove(
            labels,
            ({ label_type: type }) => type === "builtin"
          );
          const other = labels;

          const linux = remove(platforms, ({ display_text: text }) =>
            text.toLowerCase().includes("linux")
          );
          // used later when we need to send info
          setLinuxLabels(linux);

          // merge all linux OS
          const mergedLinux = reduce(
            linux,
            (result, value) => {
              if (isEmpty(result)) {
                return {
                  ...value,
                  name: "Linux",
                  display_text: "Linux",
                  description: "All Linux hosts",
                  label_type: "custom_frontend",
                };
              }

              result.count += value.count;
              result.hosts_count += value.hosts_count;
              return result;
            },
            {} as ILabel
          );

          platforms.push(mergedLinux);

          // setRelatedHosts([...hosts]);
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
          setTargetsOnlinePercent(onlineCount / targetsCount);
        }
      },
    }
  );

  const handleSelectedLabels = (entity: ILabel | ITeam) => (
    e: React.MouseEvent<HTMLButtonElement>
  ): void => {
    e.preventDefault();
    let labels = selectedLabels;
    let newTargets = null;
    let removed = [];
    const targets = selectedTargets;

    if (entity.name === "Linux") {
      removed = remove(labels, ({ name }) => name.includes("Linux"));
    } else {
      removed = remove(labels, ({ id }) => id === entity.id);
    }

    // visually show selection
    const isRemoval = removed.length > 0;
    if (isRemoval) {
      newTargets = labels;
    } else {
      labels.push(entity);

      // now prepare the labels data
      const linuxFakeIndex = labels.findIndex(
        ({ name }: any) => name === "Linux"
      );
      if (linuxFakeIndex > -1) {
        // use the official linux labels instead
        labels.splice(linuxFakeIndex, 1);
        labels = labels.concat(linuxLabels);
      }

      forEach(labels, (label) => {
        label.target_type = "label_type" in label ? "labels" : "teams";
      });

      newTargets = unionBy(targets, labels, "id");
    }

    setSelectedLabels([...labels]);
    dispatch(setSelectedTargets([...newTargets]));
  };

  const handleRowSelect = (row: Row) => {
    const targets = selectedTargets;
    const hostTarget = row.original as any; // intentional

    hostTarget.target_type = "hosts";

    targets.push(hostTarget as IHost);
    dispatch(setSelectedTargets([...targets]));
  };

  const removeHostsFromTargets = (value: number[]) => {
    const targets = selectedTargets;

    forEach(value, (id) => {
      remove(targets, (target) => target.id === id);
    });
    
    dispatch(setSelectedTargets([...targets]));
  };

  const renderTargetEntityList = (
    header: string,
    entityList: ILabel[] | ITeam[]
  ): JSX.Element => (
    <>
      {header && <h3>{header}</h3>}
      <div className="selector-block">
        {entityList?.map((entity: ILabel | ITeam) => (
          <TargetPillSelector
            key={entity.id}
            entity={entity}
            isSelected={selectedLabels.some(
              ({ id }: ILabel | ITeam) => id === entity.id
            )}
            onClick={handleSelectedLabels}
          />
        ))}
      </div>
    </>
  );

  return (
    <div className={`${baseClass}__wrapper body-wrap`}>
      <h1>Select Targets</h1>
      <div className={`${baseClass}__target-selectors`}>
        {allHostsLabels && renderTargetEntityList("", allHostsLabels)}
        {platformLabels && renderTargetEntityList("Platforms", platformLabels)}
        {teams && renderTargetEntityList("Teams", teams)}
        {otherLabels && renderTargetEntityList("Labels", otherLabels)}
      </div>
      <TargetsInput
        tabIndex={inputTabIndex}
        searchText={searchText}
        relatedHosts={[...relatedHosts]}
        selectedTargets={[...selectedTargets]}
        setSearchText={setSearchText}
        handleRowSelect={handleRowSelect}
        onPrimarySelectActionClick={removeHostsFromTargets}
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
