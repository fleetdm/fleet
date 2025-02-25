import React from "react";

import {
  DisplayPlatform,
  QueryableDisplayPlatform,
  QueryablePlatform,
} from "interfaces/platform";
import { PLATFORM_DISPLAY_NAMES } from "utilities/constants";

import TooltipWrapper from "components/TooltipWrapper";
import Icon from "components/Icon";

interface IPlatformCompatibilityProps {
  compatiblePlatforms: any[] | null;
  error: Error | null;
}

const baseClass = "platform-compatibility";

const DISPLAY_ORDER = [
  "macOS",
  "Windows",
  "Linux",
  "ChromeOS",
] as QueryableDisplayPlatform[];

const ERROR_NO_COMPATIBLE_TABLES = Error("no tables in query");

const formatPlatformsForDisplay = (
  compatiblePlatforms: QueryablePlatform[]
): DisplayPlatform[] => {
  return compatiblePlatforms.map((str) => PLATFORM_DISPLAY_NAMES[str] || str);
};

const displayIncompatibilityText = (err: Error) => {
  switch (err) {
    case ERROR_NO_COMPATIBLE_TABLES:
      return (
        <span>
          No platforms (check your query for invalid tables or tables that are
          supported on different platforms)
        </span>
      );
    default:
      return (
        <span>No platforms (check your query for a possible syntax error)</span>
      );
  }
};

const PlatformCompatibility = ({
  compatiblePlatforms,
  error,
}: IPlatformCompatibilityProps): JSX.Element | null => {
  if (!compatiblePlatforms) {
    return null;
  }

  const displayPlatforms = formatPlatformsForDisplay(compatiblePlatforms);

  const renderCompatiblePlatforms = () => {
    if (error || !compatiblePlatforms?.length) {
      return displayIncompatibilityText(error || ERROR_NO_COMPATIBLE_TABLES);
    }

    return DISPLAY_ORDER.map((platform) => {
      const isCompatible = displayPlatforms.includes(platform);
      return (
        <span key={`platform-compatibility__${platform}`} className="platform">
          <Icon
            name={isCompatible ? "check" : "close"}
            className={
              isCompatible ? "compatible-platform" : "incompatible-platform"
            }
            color={isCompatible ? "status-success" : "status-error"}
            size="small"
          />
          {platform}
        </span>
      );
    });
  };

  return (
    <div className={baseClass}>
      <b>
        <TooltipWrapper
          tipContent={
            <>
              Estimated compatibility based on the <br />
              tables used in the query. Check the <br />
              table documentation (schema) to verify <br />
              compatibility of individual columns.
              <br />
              <br />
              Only live queries are supported on ChromeOS.
              <br />
              <br />
              Querying iPhones & iPads is not supported.
            </>
          }
        >
          Compatible with:
        </TooltipWrapper>
      </b>
      {renderCompatiblePlatforms()}
    </div>
  );
};
export default PlatformCompatibility;
