import React, { useState } from "react";
import { Dispatch } from "redux";
import { useQuery } from "react-query";
import { Row } from "react-table";
import { forEach, isEmpty, reduce, remove, unionBy, uniqBy, xorBy } from "lodash";

import {
  setSelectedTargets,
  setSelectedTargetsQuery, // @ts-ignore
} from "redux/nodes/components/QueryPages/actions";
import { formatSelectedTargetsForApi } from "fleet/helpers";
import targetsAPI from "services/entities/targets";
import { ITarget, ITargets, ITargetsResponse } from "interfaces/target";
import { ICampaign } from "interfaces/campaign";
import { ILabel } from "interfaces/label";
import { ITeam } from "interfaces/team";
import { IHost } from "interfaces/host";

 // @ts-ignore
import TargetsInput from "pages/queries/QueryPage1/components/TargetsInput";
import Button from "components/buttons/Button";
import PlusIcon from "../../../../../../assets/images/icon-plus-purple-32x32@2x.png";
import CheckIcon from "../../../../../../assets/images/icon-check-purple-32x32@2x.png";

interface ITargetPillSelectorProps {
  entity: ILabel | ITeam;
  isSelected: boolean;
  onClick: (value: ILabel | ITeam) => React.MouseEventHandler<HTMLButtonElement>;
};

interface ISelectTargetsProps {
  baseClass: string;
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
  selectedTargets,
  campaign,
  isBasicTier,
  queryIdForEdit,
  goToQueryEditor,
  goToRunQuery,
  dispatch,
}: ISelectTargetsProps) => {
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

  useQuery(["targetsFromSearch", searchText], () => targetsAPI.loadAll({ query: searchText }), {
    refetchOnWindowFocus: false,

    // only retrieve the whole targets object once
    // we will only update related hosts when a search query fires
    select: (data: ITargetsResponse) => allHostsLabels ? data.targets.hosts : data.targets,
    onSuccess: (data: IHost[] | ITargets) => {
      if ("labels" in data) {
        // this will only run once
        const { hosts, labels, teams } = data as ITargets;
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
  
        // setRelatedHosts([...hosts]);
        setAllHostsLabels(allHosts);
        setPlatformLabels(platforms);
        setTeams(teams);
        setOtherLabels(other);
  
        const labelCount = allHosts.length + platforms.length + teams.length + other.length;
        setInputTabIndex(labelCount || 0);
      } else if (searchText === "") {
        setRelatedHosts([]);
      } else {
        // this will always update as the user types
        setRelatedHosts([...data] as IHost[]);
      }
    }
  });

  const handleSelectedLabels = (
    entity: ILabel | ITeam
  ) => (e: React.MouseEvent<HTMLButtonElement>): void => {
    e.preventDefault();
    let labels = selectedLabels;
    let targets = selectedTargets;
    let newTargets = null;
    let removed = [];

    if (entity.name === "Linux") {
      removed = remove(labels, ({ name }) => name.includes('Linux'));
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
      const linuxFakeIndex = labels.findIndex(({ name }: any) => name === "Linux");
      if (linuxFakeIndex > -1) {
        // use the official linux labels instead
        labels.splice(linuxFakeIndex, 1);
        labels = labels.concat(linuxLabels);
      } 
      
      forEach(labels, (label) => {
        label["target_type"] = "label_type" in label ? "labels" : "teams";
      });

      newTargets = unionBy(targets, labels, "id");
    }
    
    setSelectedLabels([...labels]);
    dispatch(setSelectedTargets([...newTargets]));
  };
  
  const handleRowSelect = (row: Row) => {
    const targets = selectedTargets;
    const hostTarget = row.original as any; // intentional

    hostTarget["target_type"] = "hosts";

    targets.push(hostTarget as IHost);
    dispatch(setSelectedTargets([...targets]));
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
      />
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
          disabled={selectedTargets.length === 0}
          onClick={goToRunQuery}
        >
          Run
        </Button>
      </div>
    </div>
  );
};

export default SelectTargets;