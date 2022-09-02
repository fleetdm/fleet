import React from "react";

import ExternalLinkIcon from "../../../../../../assets/images/icon-external-link-12x12@2x.png";

const baseClass = "empty-state";

interface IEmptyStateProps {
  title: "software" | "users" | "munki-issues";
  reason?: "empty-search" | "disabled" | "none-detected";
}

const EmptyState = ({ title, reason }: IEmptyStateProps): JSX.Element => {
  const formalTitle = () => {
    switch (title) {
      case "software":
        return "Software inventory";
      case "users":
        return "User collection";
      case "munki-issues":
        return "Munki issues";
      default:
        return "Data collection";
    }
  };

  const renderEmptyState = () => {
    switch (reason) {
      case "empty-search":
        return (
          <div className={`${baseClass}__empty-filter-results`}>
            <h2>No {title} matched your search criteria.</h2>
            <p>Try a different search.</p>
          </div>
        );
      case "disabled":
        return (
          <div className={`${baseClass}__disabled`}>
            <h2>{formalTitle()} has been disabled.</h2>
            <p>
              Check out the Fleet documentation for{" "}
              <a
                href="https://fleetdm.com/docs/using-fleet/configuration-files#features"
                target="_blank"
                rel="noopener noreferrer"
              >
                steps to enable this feature
                <img alt="External link" src={ExternalLinkIcon} />
              </a>
            </p>
          </div>
        );
      case "none-detected":
        return (
          <div className={`${baseClass}__none-detected`}>
            <h2>No {formalTitle()} detected</h2>
            <p>
              {title === "munki-issues" &&
                "The last time Munki ran on this host, no issues were reported."}
            </p>
          </div>
        );
      default:
        return (
          <div className={`${baseClass}__empty-list`}>
            <h2>
              No {title === "software" ? "installed software" : title} detected
              on this host.
            </h2>
            <p>
              Expecting to see {title}? Try again in a few seconds as the system
              catches up.
            </p>
          </div>
        );
    }
  };

  return (
    <div className={`${baseClass} empty-${title}`}>
      <div className={`${baseClass}__inner`}>{renderEmptyState()}</div>
    </div>
  );
};

export default EmptyState;
