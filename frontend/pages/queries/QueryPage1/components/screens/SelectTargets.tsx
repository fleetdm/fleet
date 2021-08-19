import React, { useState } from "react";
import { Dispatch } from "redux";
import { useQuery } from "react-query";
import { isEmpty, reduce, remove } from "lodash";

import {
  setSelectedTargets,
  setSelectedTargetsQuery, // @ts-ignore
} from "redux/nodes/components/QueryPages/actions";
import targetsAPI from "services/entities/targets";
import { ITarget, ITargetsResponse } from "interfaces/target";
import { ICampaign } from "interfaces/campaign";
import { ILabel } from "interfaces/label";
import { ITeam } from "interfaces/team";

 // @ts-ignore
import TargetsInput from "pages/queries/QueryPage1/components/TargetsInput";
import Button from "components/buttons/Button";
import PlusIcon from "../../../../../../assets/images/icon-plus-purple-32x32@2x.png";
import CheckIcon from "../../../../../../assets/images/icon-check-purple-32x32@2x.png";
import { Row } from "react-table";

interface ITargetPillSelectorProps {
  entity: ILabel | ITeam;
  isSelected: boolean;
  onClick: (value: ILabel | ITeam) => React.MouseEventHandler<HTMLButtonElement>;
};

interface ISelectTargetsProps {
  baseClass: string;
  typedQueryBody: string;
  selectedTargets: ITarget[];
  campaign: ICampaign | null;
  isBasicTier: boolean;
  queryIdForEdit: string | undefined;
  goToQueryEditor: () => void;
  goToRunQuery: () => void;
  dispatch: Dispatch;
};

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
    {entity.display_text}
    {entity.count}
  </button>
);

const SelectTargets = ({
  baseClass,
  typedQueryBody,
  selectedTargets,
  campaign,
  isBasicTier,
  queryIdForEdit,
  goToQueryEditor,
  goToRunQuery,
  dispatch,
}: ISelectTargetsProps) => {
  const [targetsCount, setTargetsCount] = useState<number>(0);
  const [targetsError, setTargetsError] = useState<string | null>(null);
  const [allHostsLabels, setAllHostsLabels] = useState<ILabel[] | null>(null);
  const [platformLabels, setPlatformLabels] = useState<ILabel[] | null>(null);
  const [linuxLabels, setLinuxLabels] = useState<ILabel[] | null>(null);
  const [teams, setTeams] = useState<ITeam[] | null>(null);
  const [otherLabels, setOtherLabels] = useState<ILabel[] | null>(null);
  const [selectedLabels, setSelectedLabels] = useState<any>([]);
  const [inputTabIndex, setInputTabIndex] = useState<number>(0);

  useQuery("targets", () => targetsAPI.loadAll({ query: typedQueryBody }), {
    refetchOnWindowFocus: false,
    onSuccess: (data: ITargetsResponse) => {
      const { labels, teams } = data.targets;
      const allHosts = remove(labels, ({ display_text: text }) => text === "All Hosts");
      const platforms = remove(labels, ({ label_type: type }) => type === "builtin");
      const other = labels;

      const linux = remove(platforms, ({ display_text: text }) => text.toLowerCase().includes('linux'));
      // used later when we need to send info
      setLinuxLabels(linux);
      
      // merge all linux OS
      const mergedLinux = reduce(linux, (result, value) => {
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
      }, {} as ILabel);
      
      platforms.push(mergedLinux);

      setAllHostsLabels(allHosts);
      setPlatformLabels(platforms);
      setTeams(teams);
      setOtherLabels(other);

      const labelCount = allHosts.length + platforms.length + teams.length + other.length;
      setInputTabIndex(labelCount || 0);
    }
  });

  const onFetchTargets = (
    targetSearchText: string,
    targetResponse: ITargetsResponse
  ) => {
    const { targets_count: responseTargetsCount } = targetResponse;

    dispatch(setSelectedTargetsQuery(targetSearchText));
    setTargetsCount(responseTargetsCount);

    return false;
  };

  // const onTargetSelect = (selected: ITarget | ITarget[]) => {
  //   setTargetsError(null);
  //   dispatch(setSelectedTargets(selectedTargets));

  //   return false;
  // };

  const handleSelectedLabels = (
    entity: ILabel | ITeam
  ) => (e: React.MouseEvent<HTMLButtonElement>): void => {
    e.preventDefault();
    let labels = selectedLabels;
    
    const index = labels.findIndex(({ id }: ILabel | ITeam) => id === entity.id);
    if (index > -1) {
      labels.splice(index, 1);
    } else {
      labels.push(entity);
    }

    setSelectedLabels([...labels]);
  };

  const renderTargetEntityList = (header: string, entityList: ILabel[] | ITeam[]): JSX.Element => (
    <>
      {header && <h3>{header}</h3>}
      <div className="selector-block">
        {entityList?.map((entity: ILabel | ITeam, i: number) => (
          <TargetPillSelector 
            key={i} 
            entity={entity} 
            isSelected={selectedLabels.some(({ id }: ILabel | ITeam) => id === entity.id)} 
            onClick={handleSelectedLabels}
          />
        ))}
      </div>
    </>
  );

  const handleRowSelect = (row: Row) => {
    console.log(row);
  };

  return (
    <div className={`${baseClass}__wrapper body-wrap`}>
      <h1>Select Targets</h1>
      <div className={`${baseClass}__target-selectors`}>
        {allHostsLabels && renderTargetEntityList("", allHostsLabels)}
        {platformLabels && renderTargetEntityList("Platforms", platformLabels)}
        {teams && renderTargetEntityList("Teams", teams)}
        {otherLabels && renderTargetEntityList("Labels", otherLabels)}
      </div>
      {/* <SelectTargetsDropdown
        error={targetsError}
        onFetchTargets={onFetchTargets}
        onSelect={onTargetSelect}
        selectedTargets={selectedTargets}
        targetsCount={targetsCount}
        label="Select targets"
        queryId={queryIdForEdit}
        isBasicTier={isBasicTier}
      /> */}
      <TargetsInput tabIndex={inputTabIndex} handleRowSelect={handleRowSelect} />
      <div className={`${baseClass}__button-wrap`}>
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
          disabled={!targetsCount}
          onClick={goToRunQuery}
        >
          Run
        </Button>
      </div>
    </div>
  );
};

export default SelectTargets;