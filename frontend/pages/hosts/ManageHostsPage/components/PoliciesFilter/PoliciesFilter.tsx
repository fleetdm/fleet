import React from "react";

import { PolicyResponse } from "utilities/constants";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";

interface IPoliciesFilterProps {
  policyResponse: PolicyResponse;
  onChange: (selectedFilter: PolicyResponse) => void;
}

const baseClass = "policies-filter";

const POLICY_RESPONSE_OPTIONS = [
  {
    disabled: false,
    label: "Yes",
    value: PolicyResponse.PASSING,
  },
  {
    disabled: false,
    label: "No",
    value: PolicyResponse.FAILING,
  },
];

const PoliciesFilter = ({
  policyResponse,
  onChange,
}: IPoliciesFilterProps): JSX.Element => {
  const value = policyResponse;

  return (
    <div className={`${baseClass}__policies-block`}>
      <Dropdown
        value={value}
        className={`${baseClass}__status_dropdown`}
        options={POLICY_RESPONSE_OPTIONS}
        searchable={false}
        onChange={onChange}
      />
    </div>
  );
};

export default PoliciesFilter;
