import React, { useEffect } from "react";
import { InjectedRouter, Link } from "react-router";
import classnames from "classnames";

import paths from "router/paths";

import Icon from "components/Icon/Icon";

const baseClass = "stacked-white-boxes";

interface IStackedWhiteBoxesProps {
  children?: JSX.Element;
  headerText?: string;
  className?: string;
  leadText?: string;
  previousLocation?: string;
  router?: InjectedRouter;
}

const StackedWhiteBoxes = ({
  children,
  headerText,
  className,
  leadText,
  previousLocation,
  router,
}: IStackedWhiteBoxesProps): JSX.Element => {
  const boxClass = classnames(baseClass, className);

  useEffect(() => {
    const closeWithEscapeKey = (e: KeyboardEvent) => {
      if (e.key === "Escape" && router) {
        router.push(paths.LOGIN);
      }
    };

    document.addEventListener("keydown", closeWithEscapeKey);

    return () => {
      document.removeEventListener("keydown", closeWithEscapeKey);
    };
  }, []);

  const renderBackButton = () => {
    if (!previousLocation) return false;

    return (
      <div className={`${baseClass}__back`}>
        <Link to={previousLocation} className={`${baseClass}__back-link`}>
          <Icon name="close" color="core-fleet-black" />
        </Link>
      </div>
    );
  };

  return (
    <div className={boxClass}>
      <div className={`${baseClass}__box`}>
        {renderBackButton()}
        {headerText && (
          <p className={`${baseClass}__header-text`}>{headerText}</p>
        )}
        {leadText && <p className={`${baseClass}__box-text`}>{leadText}</p>}
        {children}
      </div>
    </div>
  );
};

export default StackedWhiteBoxes;
