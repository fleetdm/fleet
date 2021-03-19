import PropTypes from 'prop-types';
import React, { useState } from 'react';

import ITeam from 'interfaces/team';
// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import Checkbox from 'components/forms/fields/Checkbox';
// @ts-ignore
import Dropdown from 'components/forms/fields/Dropdown';

interface ICheckboxListItem {
  name: string;
  id: number;
  isChecked: boolean | undefined;
}

interface ISelectedTeamsFormProps {
  availableTeams: ITeam[];
  selectedTeams: ITeam[];
  onFormChange: (teams: ITeam[]) => void;
}

const baseClass = 'selected-teams-form';

const roles = [
  {
    label: 'observer',
    value: 'observer',
  },
  {
    label: 'maintainer',
    value: 'maintainer',
  },
];

const generateListItems = (allTeams: ITeam[], selectedTeams: ITeam[]): ICheckboxListItem[] => {
  if (selectedTeams.length === 0) {
    return allTeams.map((team) => {
      const { name, id } = team;
      return {
        name,
        id,
        isChecked: false,
      };
    });
  }

  // TODO: add functionality editing for selected teams.
  return [];
};

const SelectedTeamsForm = (props: ISelectedTeamsFormProps): JSX.Element => {
  const { availableTeams, selectedTeams, onFormChange } = props;
  const checkboxListItems = generateListItems(availableTeams, selectedTeams);

  const onChangeInput = (checkboxItem: ICheckboxListItem): void => {
    const updatedTeam = selectedTeams.find(team => team.id === checkboxItem.id);

    // TODO: figure out logic of newly checked and updated checked.
    // if (updatedTeam === undefined) {
    //
    // } else {
    //
    // }
    // onFormChange(selectedTeam);
  };

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__team-select-items`}>
        {checkboxListItems.map((checkboxItem) => {
          return (
            <div className={`${baseClass}__team-item`}>
              <Checkbox
                name={checkboxItem.name}
                onChange={() => onChangeInput(checkboxItem)}
                value={checkboxItem.name}
              >
                {checkboxItem.name}
              </Checkbox>
              <Dropdown
                className={`${baseClass}__role-dropdown`}
                options={roles}
                searchable={false}
                onChange={() => onChangeInput(checkboxItem)}
              />
            </div>
          );
        })}
      </div>
    </div>
  );
};

export default SelectedTeamsForm;
