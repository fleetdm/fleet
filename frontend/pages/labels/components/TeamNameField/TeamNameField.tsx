import React from "react";

import FormField from "components/forms/FormField";

const baseClass = "team-name-field";

interface ITeamNameFieldProps {
  name: string;
}

const TeamNameField = ({ name }: ITeamNameFieldProps) => {
  return (
    <div className={baseClass}>
      <FormField label="Team" name="team_name">
        <p>{name}</p>
      </FormField>
    </div>
  );
};

export default TeamNameField;
