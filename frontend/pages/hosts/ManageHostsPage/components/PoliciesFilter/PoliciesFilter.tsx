import React from "react";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import { PolicyResponse } from "utilities/constants";

interface IPoliciesFilterProps {
  policyResponse: PolicyResponse;
  onChange: (selectedFilter: PolicyResponse) => void;
}

const baseClass = "policies-filter";

const POLICY_RESPONSE_OPTIONS = [
  {
    disabled: false,
    label: "Passing",
    value: PolicyResponse.PASSING,
    helpText: "Hosts that passed the last time they checked into Fleet.",
  },
  {
    disabled: false,
    label: "Failing",
    value: PolicyResponse.FAILING,
    helpText: "Hosts that failed the last time they checked into Fleet.",
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
