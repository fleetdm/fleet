import React from "react";
import { Link } from "react-router";
import classnames from "classnames";

import { buildQueryStringFromParams, QueryParams } from "utilities/url";
import { pick } from "lodash";

const baseClass = "link-with-context";

interface ILinkWithContextProps {
  children: React.ReactChild | React.ReactChild[];
  currentQueryParams: QueryParams;
  to: string;
  withParams: {
    type: "query";
    names: string[];
  };
  className?: string;
}

const LinkWithContext = ({
  children,
  currentQueryParams,
  to,
  withParams,
  className,
}: ILinkWithContextProps): JSX.Element => {
  const classNames = classnames(baseClass, className);

  let queryString = "";
  if (withParams.type === "query") {
    const newParams = pick(currentQueryParams, withParams.names);
    queryString = buildQueryStringFromParams(newParams);
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
