import { IActivity, ActivityType } from "interfaces/activity";

const DEFAULT_ACTIVITY_MOCK: IActivity = {
  created_at: "2022-11-03T17:22:14Z",
  id: 1,
  actor_full_name: "Test User",
  actor_id: 1,
  actor_gravatar: "",
  actor_email: "test@example.com",
  fleet_initiated: false,
  type: ActivityType.EditedAgentOptions,
};

const createMockActivity = (overrides?: Partial<IActivity>): IActivity => {
  return { ...DEFAULT_ACTIVITY_MOCK, ...overrides };
};

export default createMockActivity;
