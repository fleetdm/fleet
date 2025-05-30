import React from "react";
import classnames from "classnames";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";

import { IHostUpcomingActivity } from "interfaces/activity";
import {
  IHostPastActivitiesResponse,
  IHostUpcomingActivitiesResponse,
} from "services/entities/activities";

import Card from "components/Card";
import CardHeader from "components/CardHeader";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
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
  className?: string;
  upcomingCount: number;
  canCancelActivities: boolean;
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
  className,
  upcomingCount,
  canCancelActivities,
  onChangeTab,
  onNextPage,
  onPreviousPage,
  onShowDetails,
  onCancel,
}: IActivityProps) => {
  const classNames = classnames(baseClass, className);

  return (
    <Card
      borderRadiusSize="xxlarge"
      paddingSize="xlarge"
      includeShadow
      className={classNames}
    >
      {isLoading && (
        <div className={`${baseClass}__loading-overlay`}>
          <Spinner centered />
        </div>
      )}
      <CardHeader header="Activity" />
      <TabNav>
        <Tabs
          selectedIndex={activeTab === "past" ? 0 : 1}
          onSelect={onChangeTab}
        >
          <TabList>
            <Tab>
              <TabText>Past</TabText>
            </Tab>
            <Tab>
              <TabText count={upcomingCount}>Upcoming</TabText>
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
              canCancelActivities={canCancelActivities}
            />
          </TabPanel>
        </Tabs>
      </TabNav>
    </Card>
  );
};

export default Activity;
