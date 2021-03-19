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

interface ISelectedTeamsForm {
  teams: ITeam[];
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

const useCheckboxListStateManagement = (checkboxListItems: ICheckboxListItem[]): [ICheckboxListItem[], (itemId: number) => void] => {
  const [checkboxItems, setCheckboxItems] = useState(checkboxListItems);

  const updateCheckboxList = (itemId: number) => {
    setCheckboxItems((prevState) => {
      const selectedCheckbox = checkboxItems.find(checkbox => checkbox.id === itemId) as ICheckboxListItem;
      const updatedCheckbox = {
        ...selectedCheckbox,
        isChecked: !selectedCheckbox.isChecked,
      };

      // this is replacing the checkbox item object with the updatedCheckbox we just created.
      const newState = prevState.map((currentItem) => {
        return currentItem.id === itemId ? updatedCheckbox : currentItem;
      });
      return newState;
    });
  };

  return [checkboxItems, updateCheckboxList];
};

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

  // TODO: add functionality for selected teams.
  return [];

  // allTeams.map((team) => {
  //
  // });
};

const SelectedTeamsForm = (props: ISelectedTeamsForm): JSX.Element => {
  const { teams } = props;
  const checkboxListItems = generateListItems(teams, []);
  const [listItems, updateCheckboxList] = useCheckboxListStateManagement(checkboxListItems);

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__team-select-items`}>
        {listItems.map((checkboxItem) => {
          return (
            <div className={`${baseClass}__team-item`}>
              <Checkbox
                name={checkboxItem.name}
                onChange={() => updateCheckboxList(checkboxItem.id)}
                value={checkboxItem.name}
              >
                {checkboxItem.name}
              </Checkbox>
              <Dropdown
                className={`${baseClass}__role-dropdown`}
                options={roles}
                searchable={false}
              />
            </div>
          );
        })}
      </div>
    </div>
  );
};

export default SelectedTeamsForm;
