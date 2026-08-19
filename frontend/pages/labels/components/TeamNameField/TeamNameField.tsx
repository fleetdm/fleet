import React from "react";

import FormField from "components/forms/FormField";

interface ITeamNameFieldProps {
  name: string;
}

const TeamNameField = ({ name }: ITeamNameFieldProps) => {
  return (
    <FormField label="Fleet" name="fleet_name">
      <p>{name}</p>
    </FormField>
  );
};

export default TeamNameField;
