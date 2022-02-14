import React, { useState } from "react";

import { ITeam } from "interfaces/team";
// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import Checkbox from "components/forms/fields/Checkbox";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";

interface ITeamCheckboxListItem extends ITeam {
  isChecked: boolean | undefined;
}

interface ISelectedTeamsFormProps {
  availableTeams: ITeam[];
  usersCurrentTeams: ITeam[];
  onFormChange: (teams: ITeam[]) => void;
}

const baseClass = "selected-teams-form";

const roles = [
  {
    disabled: false,
    label: "Observer",
    value: "observer",
  },
  {
    disabled: false,
    label: "Maintainer",
    value: "maintainer",
  },
  {
    disabled: false,
    label: "Admin",
    value: "admin",
  },
];

const generateFormListItems = (
  allTeams: ITeam[],
  currentTeams: ITeam[]
): ITeamCheckboxListItem[] => {
  return allTeams.map((team) => {
    const foundTeam = currentTeams.find(
      (currentTeam) => currentTeam.id === team.id
    );
    return {
      ...team,
      role: foundTeam ? foundTeam.role : "observer",
      isChecked: foundTeam !== undefined,
    };
  });
};

// Handles the generation of the form data. This is eventually passed up to the parent
// so we only want to send the selected teams. The user can change the dropdown of an
// unselected item, but the parent will not track it as it only cares about selected items.
const generateSelectedTeamData = (
  teamsFormList: ITeamCheckboxListItem[]
): ITeam[] => {
  return teamsFormList.reduce((selectedTeams: ITeam[], teamItem) => {
    if (teamItem.isChecked) {
      selectedTeams.push({
        description: teamItem.description,
        id: teamItem.id,
        host_count: teamItem.host_count,
        user_count: teamItem.user_count,
        name: teamItem.name,
        role: teamItem.role,
      });
    }
    return selectedTeams;
  }, []);
};

// handles the updating of the form items.
// updates either selected state or the dropdown status of an item.
const updateFormState = (
  prevTeamItems: ITeamCheckboxListItem[],
  teamId: number,
  newValue: any,
  updateType: string
): ITeamCheckboxListItem[] => {
  const prevItemIndex = prevTeamItems.findIndex((item) => item.id === teamId);
  const prevItem = prevTeamItems[prevItemIndex];

  if (updateType === "checkbox") {
    prevItem.isChecked = newValue;
  } else {
    prevItem.role = newValue;
  }

  return [...prevTeamItems];
};

const useSelectedTeamState = (
  allTeams: ITeam[],
  currentTeams: ITeam[],
  formChange: (teams: ITeam[]) => void
) => {
  const [teamsFormList, setTeamsFormList] = useState(() => {
    return generateFormListItems(allTeams, currentTeams);
  });

  const updateSelectedTeams = (
    teamId: number,
    newValue: any,
    updateType: string
  ) => {
    setTeamsFormList((prevState) => {
      const updatedTeamFormList = updateFormState(
        prevState,
        teamId,
        newValue,
        updateType
      );
      const selectedTeamsData = generateSelectedTeamData(updatedTeamFormList);
      formChange(selectedTeamsData);
      return updatedTeamFormList;
    });
  };

  return [teamsFormList, updateSelectedTeams] as const;
};

const SelectedTeamsForm = ({
  availableTeams,
  usersCurrentTeams,
  onFormChange,
}: ISelectedTeamsFormProps): JSX.Element => {
  const [teamsFormList, updateSelectedTeams] = useSelectedTeamState(
    availableTeams,
    usersCurrentTeams,
    onFormChange
  );

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__team-select-items`}>
        {teamsFormList.map((teamItem) => {
          const { isChecked, name, role, id } = teamItem;
          return (
            <div key={id} className={`${baseClass}__team-item`}>
              <Checkbox
                value={isChecked}
                name={name}
                onChange={(newValue: boolean) =>
                  updateSelectedTeams(teamItem.id, newValue, "checkbox")
                }
              >
                {name}
              </Checkbox>
              <Dropdown
                value={role}
                className={`${baseClass}__role-dropdown`}
                options={roles}
                searchable={false}
                onChange={(newValue: string) =>
                  updateSelectedTeams(teamItem.id, newValue, "dropdown")
                }
                testId={`${name}-checkbox`}
              />
            </div>
          );
        })}
      </div>
    </div>
  );
};

export default SelectedTeamsForm;
