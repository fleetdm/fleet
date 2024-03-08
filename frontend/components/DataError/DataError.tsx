import React from "react";
import classnames from "classnames";

import CustomLink from "components/CustomLink";
import Icon from "components/Icon";

const baseClass = "data-error";

interface IDataErrorProps {
  /** the description text displayed under the header */
  description?: string;
  /** Excludes the link that asks user to create an issue. Defaults to `false` */
  excludeIssueLink?: boolean;
  children?: React.ReactNode;
  card?: boolean;
  className?: string;
}

const DEFAULT_DESCRIPTION = "Refresh the page or log in again.";

const DataError = ({
  description = DEFAULT_DESCRIPTION,
  excludeIssueLink = false,
  children,
  card,
  className,
}: IDataErrorProps): JSX.Element => {
  const classes = classnames(baseClass, className);

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
