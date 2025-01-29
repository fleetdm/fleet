import React from "react";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";

import { IHostUpcomingActivity } from "interfaces/activity";
import {
  IHostPastActivitiesResponse,
  IHostUpcomingActivitiesResponse,
} from "services/entities/activities";

import Card from "components/Card";
import TabsWrapper from "components/TabsWrapper";
import Spinner from "components/Spinner";
import TooltipWrapper from "components/TooltipWrapper";
import { ShowActivityDetailsHandler } from "components/ActivityItem/ActivityItem";

import PastActivityFeed from "./PastActivityFeed";
import UpcomingActivityFeed from "./UpcomingActivityFeed";

const baseClass = "activity-card";

const UpcomingTooltip = () => {
  return (
    <TooltipWrapper
      tipContent="Failure of one activity won't cancel other activities."
      className={`${baseClass}__upcoming-tooltip`}
    >
      Activities run as listed
    </TooltipWrapper>
  );
};

interface IActivityProps {
  activeTab: "past" | "upcoming";
  activities?: IHostPastActivitiesResponse | IHostUpcomingActivitiesResponse;
  isLoading?: boolean;
  isError?: boolean;
  upcomingCount: number;
  onChangeTab: (index: number, last: number, event: Event) => void;
  onNextPage: () => void;
  onPreviousPage: () => void;
  onShowDetails: ShowActivityDetailsHandler;
  onCancel: (activity: IHostUpcomingActivity) => void;
}

const Activity = ({
  activeTab,
  activities,
  isLoading,
  isError,
  upcomingCount,
  onChangeTab,
  onNextPage,
  onPreviousPage,
  onShowDetails,
  onCancel,
}: IActivityProps) => {
  return (
    <Card
      borderRadiusSize="xxlarge"
      includeShadow
      largePadding
      className={baseClass}
    >
      {isLoading && (
        <div className={`${baseClass}__loading-overlay`}>
          <Spinner />
        </div>
      )}
      <h2>Activity</h2>
      <TabsWrapper>
        <Tabs
          selectedIndex={activeTab === "past" ? 0 : 1}
          onSelect={onChangeTab}
        >
          <TabList>
            <Tab>Past</Tab>
            <Tab>
              Upcoming
              {!!upcomingCount && (
                <span className={`${baseClass}__upcoming-count`}>
                  {upcomingCount}
                </span>
              )}
            </Tab>
          </TabList>
          <TabPanel>
            <PastActivityFeed
              activities={activities as IHostPastActivitiesResponse | undefined}
              onShowDetails={onShowDetails}
              isError={isError}
              onNextPage={onNextPage}
              onPreviousPage={onPreviousPage}
            />
          </TabPanel>
          <TabPanel>
            <UpcomingTooltip />
            <UpcomingActivityFeed
              activities={
                activities as IHostUpcomingActivitiesResponse | undefined
              }
              onShowDetails={onShowDetails}
              onCancel={onCancel}
              isError={isError}
              onNextPage={onNextPage}
              onPreviousPage={onPreviousPage}
            />
          </TabPanel>
        </Tabs>
      </TabsWrapper>
    </Card>
  );
};

export default Activity;
