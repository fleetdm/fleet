import React from "react";
import classnames from "classnames";

import CustomLink from "components/CustomLink";
import Icon from "components/Icon";

const baseClass = "data-error";

interface IDataErrorProps {
  children?: React.ReactNode;
  card?: boolean;
  className?: string;
}

const DataError = ({
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
            <Icon name="alert" />
            Something&apos;s gone wrong.
          </span>

          <>
            {children || (
              <>
                <span className="info__data">
                  Refresh the page or log in again.
                </span>
                <span className="info__data">
                  If this keeps happening, please&nbsp;
                  <CustomLink
                    url="https://github.com/fleetdm/fleet/issues/new/choose"
                    text="file an issue"
                    newTab
                  />
                </span>
              </>
            )}
          </>
        </div>
      </div>
    </div>
  );
};

export default DataError;
