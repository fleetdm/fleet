import React from "react";
import classnames from "classnames";

import CustomLink from "components/CustomLink";
import Icon from "components/Icon";
import Graphic from "components/Graphic";

const baseClass = "data-error";

interface IDataErrorProps {
  /** the description text displayed under the header */
  description?: string;
  /** Excludes the link that asks user to create an issue. Defaults to `false` */
  excludeIssueLink?: boolean;
  children?: React.ReactNode;
  card?: boolean;
  className?: string;
  /** Flag to use the updated DataError design */
  useNew?: boolean;
}

const DEFAULT_DESCRIPTION = "Refresh the page or log in again.";

const DataError = ({
  description = DEFAULT_DESCRIPTION,
  excludeIssueLink = false,
  children,
  card,
  className,
  useNew = false,
}: IDataErrorProps): JSX.Element => {
  const classes = classnames(baseClass, className);
  if (useNew) {
    return (
      <div className={classes}>
        <div className={`${baseClass}__${card ? "card" : "inner-new"}`}>
          <Graphic name="data-error" />
          <div className={`${baseClass}__header`}>
            Something&apos;s gone wrong.
          </div>
          {children || (
            <>
              <div className={`${baseClass}__data`}>Refresh to try again.</div>
              {!excludeIssueLink && (
                <div className={`${baseClass}__data`}>
                  If this keeps happening please&nbsp;
                  <CustomLink
                    url="https://github.com/fleetdm/fleet/issues/new/choose"
                    text="file an issue"
                    newTab
                  />
                </div>
              )}
            </>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className={classes}>
      <div className={`${baseClass}__${card ? "card" : "inner"}`}>
        <div className="info">
          <span className="info__header">
            <Icon name="error" />
            Something&apos;s gone wrong.
          </span>

          <>
            {children || (
              <>
                <span className="info__data">{description}</span>
                {!excludeIssueLink && (
                  <span className="info__data">
                    If this keeps happening, please&nbsp;
                    <CustomLink
                      url="https://github.com/fleetdm/fleet/issues/new/choose"
                      text="file an issue"
                      newTab
                    />
                  </span>
                )}
              </>
            )}
          </>
        </div>
      </div>
    </div>
  );
};

export default DataError;
