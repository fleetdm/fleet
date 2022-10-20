import React from "react";

import Icon from "components/Icon";

interface IExternalLinkProps {
  url: string;
  text: string;
}

const ExternalLink = ({ url, text }: IExternalLinkProps): JSX.Element => {
  return (
    <a href={url} target="_blank" rel="noopener noreferrer">
      {text}
      <Icon name="external-link" />
    </a>
  );
};
export default ExternalLink;
