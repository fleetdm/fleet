import React from "react";

import CustomLink from "components/CustomLink";
import ErrorIcon from "../../../assets/images/icon-error-16x16@2x.png";

const baseClass = "data-error";

interface IDataErrorProps {
  card?: boolean;
}

const DataError = ({ card }: IDataErrorProps): JSX.Element => {
  return (
    <div className={`${baseClass}`}>
      <div className={`${baseClass}__${card ? "card" : "inner"}`}>
        <div className="info">
          <span className="info__header">
            <img src={ErrorIcon} alt="error icon" id="error-icon" />
            Something&apos;s gone wrong.
          </span>
          <span className="info__data">Refresh the page or log in again.</span>
          <span className="info__data">
            If this keeps happening, please&nbsp;
            <CustomLink
              url="https://github.com/fleetdm/fleet/issues/new/choose"
              text="file an issue"
              newTab
            />
          </span>
        </div>
      </div>
    </div>
  );
};

export default DataError;
