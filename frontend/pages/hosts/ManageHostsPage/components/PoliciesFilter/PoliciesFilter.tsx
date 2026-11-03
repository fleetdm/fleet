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
    label: "Pass",
    value: PolicyResponse.PASSING,
  },
  {
    disabled: false,
    label: "Fail",
    value: PolicyResponse.FAILING,
  },
];

const PoliciesFilter = ({
  policyResponse,
  onChange,
}: IPoliciesFilterProps): JSX.Element => {
  const value = policyResponse;

  return (
    <div className={baseClass}>
      <Dropdown
        value={value}
        className={`${baseClass}__status-filter`}
        options={POLICY_RESPONSE_OPTIONS}
        searchable={false}
        onChange={onChange}
        iconName="filter-alt"
      />
    </div>
  );
};

export default PoliciesFilter;
