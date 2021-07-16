import React, { useEffect } from "react";
// @ts-ignore
import Fleet from "fleet";
import { create, lowerCase } from "lodash";

import Avatar from "components/Avatar";

const baseClass = "activity-feed";

const activities = [
  {
    id: 1,
    created_at: "2020-11-05T05:09:44Z",
    actor_full_name: "Jane Doe",
    actor_id: 1,
    type: "created_pack",
    details: {
      pack_id: 1,
      pack_name: "Cool pack",
    },
  },
  {
    id: 2,
    created_at: "2020-11-05T05:09:44Z",
    actor_full_name: "Jane Doe",
    actor_id: 1,
    type: "deleted_pack",
    details: {
      pack_name: "Cool pack",
    },
  },
  {
    id: 3,
    created_at: "2020-11-05T05:09:44Z",
    actor_full_name: "Jane Doe",
    actor_id: 1,
    type: "edited_pack",
    details: {
      pack_id: 1,
      pack_name: "Cool pack",
    },
  },
  {
    id: 4,
    created_at: "2020-11-05T05:09:44Z",
    actor_full_name: "Jane Doe",
    actor_id: 1,
    type: "created_saved_query",
    details: {
      query_id: 1,
      query_name: "Awesome query",
    },
  },
  {
    id: 5,
    created_at: "2020-11-05T05:09:44Z",
    actor_full_name: "Jane Doe",
    actor_id: 1,
    type: "deleted_saved_query",
    details: {
      query_name: "Awesome query",
    },
  },
  {
    id: 6,
    created_at: "2020-11-05T05:09:44Z",
    actor_full_name: "Jane Doe",
    actor_id: 1,
    type: "edited_saved_query",
    details: {
      query_id: 1,
      query_name: "Awesome query",
    },
  },
  {
    id: 7,
    created_at: "2020-11-05T05:09:44Z",
    actor_full_name: "Jane Doe",
    actor_id: 1,
    type: "created_team",
    details: {
      team_id: 1,
      team_name: "Walmart Pay",
    },
  },
  {
    id: 8,
    created_at: "2020-11-05T05:09:44Z",
    actor_full_name: "Jane Doe",
    actor_id: 1,
    type: "deleted_team",
    details: {
      team_name: "Walmart Pay",
    },
  },
  {
    id: 9,
    created_at: "2020-11-05T05:09:44Z",
    actor_full_name: "Jane Doe",
    actor_id: 1,
    type: "live_query",
    details: {
      targets_count: 12030,
    },
  },
  {
    id: 10,
    created_at: "2020-11-05T05:09:44Z",
    actor_full_name: "Jane Doe",
    actor_id: 1,
    type: "applied_pack_spec",
  },
  {
    id: 11,
    created_at: "2020-11-05T05:09:44Z",
    actor_full_name: "Jane Doe",
    actor_id: 1,
    type: "applied_query_spec",
  },
];

interface IActivity {
  id: number;
  created_at: string;
  actor_full_name: string;
  actor_id: number;
  type: string;
  details?: {
    [key: string]: any;
  };
}

const ActivityFeed = (): JSX.Element => {
  useEffect(() => {
    const getActivities = async () => {
      const a = await Fleet.activities.loadNext();
      console.log(a);
    };

    getActivities();
  }, []);

  const renderActivityBlock = (
    { created_at, actor_full_name, type, details }: IActivity,
    i: number
  ) => (
    <div className={`${baseClass}__block`} key={i}>
      <Avatar
        className={`${baseClass}__avatar-image`}
        user={{
          gravatarURL:
            "https://www.gravatar.com/avatar/6a8b7225be7bd98b310d756efb312b69?d=blank&size=200",
        }}
        size="small"
      />
      <div className={`${baseClass}__details`}>
        <p>
          <b>{actor_full_name}</b> {lowerCase(type)}
        </p>
        <span></span>
      </div>
    </div>
  );

  return (
    <div className={baseClass}>
      {activities.map((activity: IActivity, i: number) =>
        renderActivityBlock(activity, i)
      )}
    </div>
  );
};

export default ActivityFeed;
