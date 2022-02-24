import React, { useCallback, useEffect, useState } from "react";
import { Row } from "react-table";
import { forEach, isEmpty, remove, unionBy } from "lodash";
import { useDebouncedCallback } from "use-debounce/lib";

// @ts-ignore
import { formatSelectedTargetsForApi } from "fleet/helpers";
import useQueryTargets, { ITargetsQueryResponse } from "hooks/useQueryTargets";
import { ITarget } from "interfaces/target";
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
  goToQueryEditor: () => void;
  goToRunQuery: () => void;
  setSelectedTargets: React.Dispatch<React.SetStateAction<ITarget[]>>;
}

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
  goToQueryEditor,
  goToRunQuery,
  setSelectedTargets,
}: ISelectTargetsProps): JSX.Element => {
  const [allHostsLabels, setAllHostsLabels] = useState<ILabel[] | null>(null);
  const [platformLabels, setPlatformLabels] = useState<ILabel[] | null>(null);
  const [teams, setTeams] = useState<ITeam[] | null>(null);
  const [otherLabels, setOtherLabels] = useState<ILabel[] | null>(null);
  const [selectedLabels, setSelectedLabels] = useState<any>([]);
  const [inputTabIndex, setInputTabIndex] = useState<number>(0);
  const [searchText, setSearchText] = useState<string>("");
  const [debouncedSearchText, setDebouncedSearchText] = useState<string>("");
  const [isDebouncing, setIsDebouncing] = useState<boolean>(false);

  const DEBOUNCE_DELAY = 500;

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

  const setLabels = useCallback(
    (data: ITargetsQueryResponse) => {
      if (!allHostsLabels) {
        setAllHostsLabels(data.allHostsLabels || []);
        setPlatformLabels(data.platformLabels || []);
        setOtherLabels(data.otherLabels || []);
        setTeams(data.teams || []);
        setInputTabIndex(data.labelCount || 0);
      }
    },
    [allHostsLabels]
  );

  const {
    data: targets,
    isFetching: isTargetsFetching,
    isError: isTargetsError,
  } = useQueryTargets(
    [
      {
        scope: "SelectTargets",
        query: debouncedSearchText,
        queryId: null,
        selected: formatSelectedTargetsForApi(selectedTargets),
        includeLabels: !allHostsLabels,
      },
    ],
    {
      onSuccess: setLabels,
    }
  );

  const handleClickCancel = () => {
    setSelectedTargets([]);
    goToQueryEditor();
  };

  const handleSelectedLabels = (entity: ILabel | ITeam) => (
    e: React.MouseEvent<HTMLButtonElement>
  ): void => {
    e.preventDefault();
    const labels = selectedLabels;
    let newTargets = null;
    const prevTargets = selectedTargets;
    const removed = remove(labels, ({ uuid }) => uuid === entity.uuid);

    // visually show selection
    const isRemoval = removed.length > 0;
    if (isRemoval) {
      newTargets = labels;
    } else {
      labels.push(entity);

      // prepare the labels data
      forEach(labels, (label) => {
        label.target_type = "label_type" in label ? "labels" : "teams";
      });

      newTargets = unionBy(prevTargets, labels, "id");
    }

    setSelectedLabels([...labels]);
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
    remove(newTargets, (t) => t.id === hostTarget.id);

    setSelectedTargets([...newTargets]);
  };

  const renderTargetEntityList = (
    header: string,
    entityList: ILabel[] | ITeam[]
  ): JSX.Element => (
    <>
      {header && <h3>{header}</h3>}
      <div className="selector-block">
        {entityList?.map((entity: ILabel | ITeam) => {
          return (
            <TargetPillSelector
              key={entity.uuid}
              entity={entity}
              isSelected={selectedLabels.some(
                ({ uuid }: ILabel | ITeam) => uuid === entity.uuid
              )}
              onClick={handleSelectedLabels}
            />
          );
        })}
      </div>
    </>
  );

  if (!isTargetsError && isEmpty(searchText) && !allHostsLabels) {
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
        relatedHosts={targets?.relatedHosts || []}
        isTargetsLoading={isTargetsFetching || isDebouncing}
        selectedTargets={[...selectedTargets]}
        hasFetchError={isTargetsError}
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
          disabled={!targets?.targetsTotalCount}
          onClick={goToRunQuery}
        >
          Run
        </Button>
        <div className={`${baseClass}__targets-total-count`}>
          {!!targets?.targetsTotalCount && (
            <>
              <span>{targets?.targetsTotalCount}</span> targets selected&nbsp; (
              {targets?.targetsOnlinePercent}% online)
            </>
          )}
        </div>
      </div>
    </div>
  );
};

export default SelectTargets;
