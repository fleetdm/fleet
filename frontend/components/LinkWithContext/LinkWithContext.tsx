import React from "react";
import { Link } from "react-router";

import { buildQueryStringFromParams, QueryParams } from "utilities/url";
import { pick } from "lodash";

interface ILinkWithContextProps {
  className: string;
  children: React.ReactChild | React.ReactChild[];
  currentQueryParams: QueryParams;
  to: string;
  withParams: {
    type: "query";
    names: string[];
  };
}

const LinkWithContext = ({
  className,
  children,
  currentQueryParams,
  to,
  withParams,
}: ILinkWithContextProps): JSX.Element => {
  let queryString = "";
  if (withParams.type === "query") {
    const newParams = pick(currentQueryParams, withParams.names);
    queryString = buildQueryStringFromParams(newParams);
  }
  return (
    <Link
      className={className}
      to={queryString.length ? `${to}?${queryString}` : to}
    >
      {children}
    </Link>
  );
};
export default LinkWithContext;
