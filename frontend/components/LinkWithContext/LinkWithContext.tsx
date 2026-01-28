import React from "react";
import { Link } from "react-router";
import classnames from "classnames";

import { buildQueryStringFromParams, QueryParams } from "utilities/url";
import { pick } from "lodash";

const baseClass = "link-with-context";

interface ILinkWithContextProps {
  children: React.ReactNode;
  currentQueryParams: QueryParams;
  to: string;
  withParams: {
    type: "query";
    names: string[];
  };
  className?: string;
  disabled?: boolean;
}

const LinkWithContext = ({
  children,
  currentQueryParams,
  to,
  withParams,
  className,
  disabled = false,
}: ILinkWithContextProps): JSX.Element => {
  const classNames = classnames(baseClass, className);

  let queryString = "";
  if (withParams.type === "query") {
    const newParams = pick(currentQueryParams, withParams.names);
    queryString = buildQueryStringFromParams(newParams);
  }

  if (disabled) {
    return <span className={classNames}>{children}</span>;
  }

  return (
    <Link
      className={classNames}
      to={queryString.length ? `${to}?${queryString}` : to}
    >
      {children}
    </Link>
  );
};
export default LinkWithContext;
