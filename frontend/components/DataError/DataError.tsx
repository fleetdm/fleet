import React from "react";

import CustomLink from "components/CustomLink";
import Icon from "components/Icon";

const baseClass = "data-error";

interface IDataErrorProps {
  card?: boolean;
  message?: JSX.Element | string;
}

const DataError = ({ card, message }: IDataErrorProps): JSX.Element => {
  return (
    <div className={`${baseClass}`}>
      <div className={`${baseClass}__${card ? "card" : "inner"}`}>
        <div className="info">
          <span className="info__header">
            <Icon name="alert" />
            Something&apos;s gone wrong.
          </span>
          {message ? (
            { message }
          ) : (
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
        </div>
      </div>
    </div>
  );
};

export default DataError;
