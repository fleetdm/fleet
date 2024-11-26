import React, { useState } from "react";

import { ITeam } from "interfaces/team";
import { UserRole } from "interfaces/user";
import Checkbox from "components/forms/fields/Checkbox";
import { SingleValue } from "react-select-5";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import { roleOptions } from "../../helpers/userManagementHelpers";

interface ITeamCheckboxListItem extends ITeam {
  isChecked: boolean | undefined;
}

interface ISelectedTeamsFormProps {
  availableTeams: ITeam[];
  usersCurrentTeams: ITeam[];
  onFormChange: (teams: ITeam[]) => void;
  isApiOnly?: boolean;
}

const baseClass = "selected-teams-form";

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
  newValue: SingleValue<CustomOptionType> | boolean | undefined
): ITeamCheckboxListItem[] => {
  const prevItemIndex = prevTeamItems.findIndex((item) => item.id === teamId);
  const prevItem = prevTeamItems[prevItemIndex];

  if (typeof newValue === "boolean") {
    prevItem.isChecked = newValue;
  } else {
    prevItem.role = newValue?.value as UserRole;
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
    newValue: CustomOptionType | boolean
  ) => {
    setTeamsFormList((prevState) => {
      const updatedTeamFormList = updateFormState(prevState, teamId, newValue);
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
  isApiOnly,
}: ISelectedTeamsFormProps): JSX.Element => {
  const [teamsFormList, updateSelectedTeams] = useSelectedTeamState(
    availableTeams,
    usersCurrentTeams,
    onFormChange
  );

  return (
    <div className={`${baseClass} form`}>
      {teamsFormList.map((teamItem) => {
        const { isChecked, name, role, id } = teamItem;
        return (
          <div key={id} className={`${baseClass}__team-item`}>
            <Checkbox
              value={isChecked}
              name={name}
              onChange={(newValue: boolean) =>
                updateSelectedTeams(teamItem.id, newValue)
              }
            >
              {name}
            </Checkbox>
            <DropdownWrapper
              name={name}
              value={role}
              className={`${baseClass}__role-dropdown`}
              options={roleOptions({ isPremiumTier: true, isApiOnly })}
              isSearchable={false}
              onChange={(newValue: SingleValue<CustomOptionType>) =>
                updateSelectedTeams(teamItem.id, newValue as CustomOptionType)
              }
              // testId={`${name}-checkbox`}
            />
          </div>
        );
      })}
    </div>
  );
};

export default SelectedTeamsForm;
