import React from "react";
import { Link } from "react-router";

import { buildQueryStringFromParams, QueryParams } from "utilities/url";
import { pick } from "lodash";

interface ILinkWithContextProps {
  className: string;
  children: React.ReactChild | React.ReactChild[];
  query: QueryParams;
  to: string;
  withUrlQueryParams: string[];
}

const LinkWithContext = ({
  className,
  children,
  query,
  to,
  withUrlQueryParams: paramsToKeep,
}: ILinkWithContextProps): JSX.Element => {
  const queryString = buildQueryStringFromParams(pick(query, paramsToKeep));

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
