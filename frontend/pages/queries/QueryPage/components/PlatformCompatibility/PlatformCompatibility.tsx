import React from "react";

import TooltipWrapper from "components/TooltipWrapper";
import CompatibleIcon from "../../../../../../assets/images/icon-compatible-green-16x16@2x.png";
import IncompatibleIcon from "../../../../../../assets/images/icon-incompatible-red-16x16@2x.png";

const baseClass = "platform-compatibility";

const DISPLAY_ORDER = ["macOS", "Windows", "Linux"];

const formatPlatformsForDisplay = (compatiblePlatforms: string[]) => {
  return compatiblePlatforms.map((string) => {
    switch (string) {
      case "darwin":
        return "macOS";
      case "windows":
        return "Windows";
      case "linux":
        return "Linux";
      default:
        return string;
    }
  });
};

const displayIncompatibilityText = (compatiblePlatforms: string[]) => {
  if (compatiblePlatforms[0] === "Invalid query") {
    return "No platforms (check your query for a possible syntax error)";
  } else if (compatiblePlatforms[0] === "None") {
    return "No platforms (check your query for invalid tables or tables that are supported on different platforms)";
  }
  return null;
};

const PlatformCompatibility = ({
  compatiblePlatforms,
}: {
  compatiblePlatforms: string[];
}): JSX.Element => {
  compatiblePlatforms = formatPlatformsForDisplay(compatiblePlatforms);
  return (
    <span className={baseClass}>
      <b>
        <TooltipWrapper tipContent="Estimated compatiblity based on the tables used in the query">
          Compatible with:
        </TooltipWrapper>
      </b>
      {displayIncompatibilityText(compatiblePlatforms) ||
        (!!compatiblePlatforms.length &&
          DISPLAY_ORDER.map((platform) => {
            const isCompatible =
              compatiblePlatforms.includes(platform) ||
              compatiblePlatforms[0] === "No tables in query AST"; // If query has no tables but is still syntatically valid sql, we treat it as compatible with all platforms
            return (
              <span
                key={`platform-compatibility__${platform}`}
                className="platform"
              >
                {platform}{" "}
                <img
                  alt={isCompatible ? "compatible" : "incompatible"}
                  src={isCompatible ? CompatibleIcon : IncompatibleIcon}
                />
              </span>
            );
          }))}
    </span>
  );
};
export default PlatformCompatibility;
