import React from "react";

import Tag from "components/Tag";

interface IApiEndpointCountTagProps {
  count: number;
}

const ApiEndpointCountTag = ({ count }: IApiEndpointCountTagProps) => (
  <Tag size="xsmall">{`${count} API endpoint${count === 1 ? "" : "s"}`}</Tag>
);

export default ApiEndpointCountTag;
