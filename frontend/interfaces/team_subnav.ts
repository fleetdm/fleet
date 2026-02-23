import { InjectedRouter } from "react-router";

export interface ITeamSubnavProps {
  location: {
    pathname: string;
    search: string;
    hash?: string;
    query: { team_id?: string };
  };
  router: InjectedRouter;
}
