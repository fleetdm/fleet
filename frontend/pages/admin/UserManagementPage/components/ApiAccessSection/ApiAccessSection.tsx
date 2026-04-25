import React, { useCallback } from "react";

import { IApiEndpointRef } from "interfaces/api_endpoint";
import Radio from "components/forms/fields/Radio";
import TooltipWrapper from "components/TooltipWrapper";
import ApiEndpointSelectorTable from "../ApiEndpointSelectorTable";

const baseClass = "api-access-section";

enum ApiAccessType {
  AllEndpoints = "ALL_ENDPOINTS",
  SpecificEndpoints = "SPECIFIC_ENDPOINTS",
}

interface IApiAccessSectionProps {
  isSpecificEndpoints: boolean;
  onAccessTypeChange: (isSpecific: boolean) => void;
  selectedEndpoints: IApiEndpointRef[];
  onEndpointSelectionChange: (endpoints: IApiEndpointRef[]) => void;
  error?: string | null;
}

const ApiAccessSection = ({
  isSpecificEndpoints,
  onAccessTypeChange,
  selectedEndpoints,
  onEndpointSelectionChange,
  error,
}: IApiAccessSectionProps) => {
  const handleAccessTypeChange = useCallback(
    (value: string) => {
      onAccessTypeChange(value === ApiAccessType.SpecificEndpoints);
    },
    [onAccessTypeChange]
  );

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__access-type-field form-field`}>
        <div className="form-field__label">API access</div>
        <Radio
          className={`${baseClass}__radio-input`}
          label="All API endpoints"
          id="all-endpoints"
          checked={!isSpecificEndpoints}
          value={ApiAccessType.AllEndpoints}
          name="api-access-type"
          onChange={handleAccessTypeChange}
        />
        <Radio
          className={`${baseClass}__radio-input`}
          label="Specific API endpoints"
          id="specific-endpoints"
          checked={isSpecificEndpoints}
          value={ApiAccessType.SpecificEndpoints}
          name="api-access-type"
          onChange={handleAccessTypeChange}
        />
      </div>
      {isSpecificEndpoints && (
        <div className={`${baseClass}__endpoint-selector`}>
          <div className="form-field">
            <div className="form-field__label">
              <TooltipWrapper tipContent="Specifying endpoints can narrow down a user's API access, but will not grant additional permissions otherwise forbidden by their role.">
                Select API endpoints
              </TooltipWrapper>
            </div>
          </div>
          <ApiEndpointSelectorTable
            selectedEndpoints={selectedEndpoints}
            onSelectionChange={onEndpointSelectionChange}
          />
          {error && (
            <div className="form-field__label form-field__label--error">
              {error}
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default ApiAccessSection;
