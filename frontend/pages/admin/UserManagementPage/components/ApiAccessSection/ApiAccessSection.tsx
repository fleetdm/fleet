import React, { useState, useCallback } from "react";

import Radio from "components/forms/fields/Radio";
import TooltipWrapper from "components/TooltipWrapper";
import EndpointSelectorTable from "../EndpointSelectorTable";

const baseClass = "api-access-section";

enum ApiAccessType {
  AllEndpoints = "ALL_ENDPOINTS",
  SpecificEndpoints = "SPECIFIC_ENDPOINTS",
}

interface IApiAccessSectionProps {
  selectedEndpointIds: string[];
  onEndpointSelectionChange: (selectedIds: string[]) => void;
}

const ApiAccessSection = ({
  selectedEndpointIds,
  onEndpointSelectionChange,
}: IApiAccessSectionProps) => {
  const [accessType, setAccessType] = useState<ApiAccessType>(
    selectedEndpointIds.length > 0
      ? ApiAccessType.SpecificEndpoints
      : ApiAccessType.AllEndpoints
  );

  const handleAccessTypeChange = useCallback(
    (value: string) => {
      const newType = value as ApiAccessType;
      setAccessType(newType);
      if (newType === ApiAccessType.AllEndpoints) {
        onEndpointSelectionChange([]);
      }
    },
    [onEndpointSelectionChange]
  );

  return (
    <div className={baseClass}>
      <div className="form-field">
        <div className="form-field__label">API access</div>
        <Radio
          className={`${baseClass}__radio-input`}
          label="All API endpoints"
          id="all-endpoints"
          checked={accessType === ApiAccessType.AllEndpoints}
          value={ApiAccessType.AllEndpoints}
          name="api-access-type"
          onChange={handleAccessTypeChange}
        />
        <Radio
          className={`${baseClass}__radio-input`}
          label="Specific API endpoints"
          id="specific-endpoints"
          checked={accessType === ApiAccessType.SpecificEndpoints}
          value={ApiAccessType.SpecificEndpoints}
          name="api-access-type"
          onChange={handleAccessTypeChange}
        />
      </div>
      {accessType === ApiAccessType.SpecificEndpoints && (
        <div className={`${baseClass}__endpoint-selector`}>
          <div className="form-field">
            <div className="form-field__label">
              <TooltipWrapper tipContent="Specifying endpoints can narrow down a user's API access, but will not grant additional permissions otherwise forbidden by their role.">
                Select API endpoints
              </TooltipWrapper>
            </div>
          </div>
          <EndpointSelectorTable
            selectedEndpointIds={selectedEndpointIds}
            onSelectionChange={onEndpointSelectionChange}
          />
        </div>
      )}
    </div>
  );
};

export default ApiAccessSection;
