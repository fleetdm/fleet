import { IUser } from "interfaces/user";
import { ITeam } from "interfaces/team";

export const teamStub: ITeam = {
  description: "This is the test team",
  host_count: 10,
  id: 1,
  name: "Test Team",
  user_count: 5,
};

export const userTeamStub: ITeam = {
  ...teamStub,
  role: "observer",
};

export const userStub: IUser = {
  id: 1,
  name: "Gnar Mike",
  email: "hi@gnar.dog",
  role: "Observer",
  global_role: null,
  api_only: false,
  force_password_reset: false,
  gravatar_url: "https://image.com",
  sso_enabled: false,
  teams: [{ ...userTeamStub }],
};

export default {
  userStub,
};
