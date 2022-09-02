import React from "react";

import ExternalLinkIcon from "../../../../../../assets/images/icon-external-link-12x12@2x.png";

const baseClass = "empty-state";

interface IEmptyStateProps {
  title: "software" | "users";
  reason?: "empty-search" | "disabled";
}

const EmptyState = ({ title, reason }: IEmptyStateProps): JSX.Element => {
  const formalTitle = () => {
    switch (title) {
      case "software":
        return "Software inventory";
      case "users":
        return "User collection";
      default:
        return "Data collection";
    }
  };

  switch (reason) {
    case "empty-search":
      return (
        <div className={`${baseClass} empty-${title} empty-search`}>
          <div className={`${baseClass}__inner`}>
            <div className={`${baseClass}__empty-filter-results`}>
              <h1>No {title} matched your search criteria.</h1>
              <p>Try a different search.</p>
            </div>
          </div>
        </div>
      );
    case "disabled":
      return (
        <div className={`${baseClass} empty-${title}`}>
          <div className={`${baseClass}__inner`}>
            <div className={`${baseClass}__disabled`}>
              <h1>{formalTitle()} has been disabled.</h1>
              <p>
                Check out the Fleet documentation for{" "}
                <a
                  href="https://fleetdm.com/docs/using-fleet/configuration-files#features"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  steps to enable this feature
                  <img src={ExternalLinkIcon} alt="Open external link" />
                </a>
              </p>
            </div>
          </div>
        </div>
      );
    default:
      return (
        <div className={`${baseClass} empty-${title}`}>
          <div className={`${baseClass}__inner`}>
            <div className={`${baseClass}__empty-list`}>
              <h1>
                No {title === "software" ? "installed software" : title}{" "}
                detected on this host.
              </h1>
              <p>
                Expecting to see {title}? Try again in a few seconds as the
                system catches up.
              </p>
            </div>
          </div>
        </div>
      );
  }
};

export default EmptyState;
