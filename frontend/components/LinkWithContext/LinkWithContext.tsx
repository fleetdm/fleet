import React, { useContext } from "react";
import { Link } from "react-router";

import { AppContext } from "context/app";

interface ILinkWithContextProps {
  className: string;
  children: React.ReactChild | React.ReactChild[];
  to: string;
}

const LinkWithContext = ({
  className,
  children,
  to,
}: ILinkWithContextProps): JSX.Element => {
  const { currentTeam } = useContext(AppContext);
  const url = currentTeam?.id ? `${to}?team_id=${currentTeam.id}` : to;

  return (
    <Link className={className} to={url}>
      {children}
    </Link>
  );
};
export default LinkWithContext;
