import React from "react";
import { Link } from "react-router";

import { buildQueryStringFromParams, QueryParams } from "utilities/url";
import { pick } from "lodash";
import { Params } from "react-router/lib/Router";

interface ILinkWithContextProps {
  className: string;
  children: React.ReactChild | React.ReactChild[];
  queryParams: QueryParams;
  routeParams: Params;
  to: string;
  withParams: { type: "query" | "route"; names: string[] };
}

const LinkWithContext = ({
  className,
  children,
  queryParams,
  routeParams,
  to,
  withParams,
}: ILinkWithContextProps): JSX.Element => {
  let queryString = "";
  if (withParams.type === "query") {
    const newParams = pick(queryParams, withParams.names);
    if (routeParams.team_id && newParams.team_id === undefined) {
      newParams.team_id = routeParams.team_id;
    }
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
