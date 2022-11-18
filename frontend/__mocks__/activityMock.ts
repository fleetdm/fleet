import { IActivity, ActivityType } from "interfaces/activity";

const DEFAULT_ACTIVITY_MOCK: IActivity = {
  created_at: "2022-11-03T17:22:14Z",
  id: 1,
  actor_full_name: "Rachel",
  actor_id: 1,
  actor_gravatar: "",
  actor_email: "rachel@fleetdm.com",
  type: ActivityType.EditedAgentOptions,
};

const createMockActivity = (overrides?: Partial<IActivity>): IActivity => {
  return { ...DEFAULT_ACTIVITY_MOCK, ...overrides };
};

export default createMockActivity;
