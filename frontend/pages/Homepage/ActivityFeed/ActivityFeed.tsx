import React, { useEffect, useRef, useState } from "react";
import moment from "moment";

// @ts-ignore
import Fleet from "fleet";
import { find, lowerCase } from "lodash";

import { IActivity, IActivityDetails } from "interfaces/activity";

import Avatar from "components/Avatar";

const baseClass = "activity-feed";

// const feed = [
//   {
//     id: 1,
//     created_at: "2020-11-05T05:09:44Z",
//     actor_full_name: "Jane Doe",
//     actor_id: 1,
//     type: "created_pack",
//     details: {
//       pack_id: 1,
//       pack_name: "Cool pack",
//     },
//   },
//   {
//     id: 2,
//     created_at: "2020-11-05T05:09:44Z",
//     actor_full_name: "Jane Doe",
//     actor_id: 1,
//     type: "deleted_pack",
//     details: {
//       pack_name: "Cool pack",
//     },
//   },
//   {
//     id: 3,
//     created_at: "2020-11-05T05:09:44Z",
//     actor_full_name: "Jane Doe",
//     actor_id: 1,
//     type: "edited_pack",
//     details: {
//       pack_id: 1,
//       pack_name: "Cool pack",
//     },
//   },
//   {
//     id: 4,
//     created_at: "2020-11-05T05:09:44Z",
//     actor_full_name: "Jane Doe",
//     actor_id: 1,
//     type: "created_saved_query",
//     details: {
//       query_id: 1,
//       query_name: "Awesome query",
//     },
//   },
//   {
//     id: 5,
//     created_at: "2020-11-05T05:09:44Z",
//     actor_full_name: "Jane Doe",
//     actor_id: 1,
//     type: "deleted_saved_query",
//     details: {
//       query_name: "Awesome query",
//     },
//   },
//   {
//     id: 6,
//     created_at: "2020-11-05T05:09:44Z",
//     actor_full_name: "Jane Doe",
//     actor_id: 1,
//     type: "edited_saved_query",
//     details: {
//       query_id: 1,
//       query_name: "Awesome query",
//     },
//   },
//   {
//     id: 7,
//     created_at: "2020-11-05T05:09:44Z",
//     actor_full_name: "Jane Doe",
//     actor_id: 1,
//     type: "created_team",
//     details: {
//       team_id: 1,
//       team_name: "Walmart Pay",
//     },
//   },
//   {
//     id: 8,
//     created_at: "2020-11-05T05:09:44Z",
//     actor_full_name: "Jane Doe",
//     actor_id: 1,
//     type: "deleted_team",
//     details: {
//       team_name: "Walmart Pay",
//     },
//   },
//   {
//     id: 9,
//     created_at: "2020-11-05T05:09:44Z",
//     actor_full_name: "Jane Doe",
//     actor_id: 1,
//     type: "live_query",
//     details: {
//       targets_count: 12030,
//     },
//   },
//   {
//     id: 10,
//     created_at: "2020-11-05T05:09:44Z",
//     actor_full_name: "Jane Doe",
//     actor_id: 1,
//     type: "applied_pack_spec",
//   },
//   {
//     id: 11,
//     created_at: "2020-11-05T05:09:44Z",
//     actor_full_name: "Jane Doe",
//     actor_id: 1,
//     type: "applied_query_spec",
//   },
// ];

const ActivityFeed = (): JSX.Element => {
  const [activities, setActivities] = useState([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect((): void => {
    const getActivities = async (): Promise<void> => {
      const a = await Fleet.activities.loadNext();
      setActivities(a);
      setIsLoading(false);
    };

    getActivities();
  }, []);

  const getDetail = (activity: IActivity) => {
    const type = activity.type;
    const entityName = find(activity.details, (_, key) =>
      key.includes("_name")
    );
    if (type === "live_query") {
      return `ran a live query on ${activity.details?.targets_count} hosts`;
    }
    return !entityName ? (
      lowerCase(type)
    ) : (
      <span>
        {lowerCase(type)} <b>{entityName}</b>
      </span>
    );
  };

  // TODO gravatar; dotted line
  const renderActivityBlock = (activity: IActivity, i: number) => (
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
        <p className={`${baseClass}__details-topline`}>
          <b>{activity.actor_full_name}</b> {getDetail(activity)}.
        </p>
        <span className={`${baseClass}__details-bottomline`}>
          {moment(activity.created_at).fromNow()}
        </span>
      </div>
    </div>
  );

  return (
    <div className={baseClass}>
      {!isLoading &&
        activities.map((activity: IActivity, i: number) =>
          renderActivityBlock(activity, i)
        )}
    </div>
  );
};

export default ActivityFeed;
