import React from "react";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";

import { IActivityDetails } from "interfaces/activity";
import {
  IHostPastActivitiesResponse,
  IHostUpcomingActivitiesResponse,
} from "services/entities/activities";

import Card from "components/Card";
import TabsWrapper from "components/TabsWrapper";
import Spinner from "components/Spinner";
import TooltipWrapper from "components/TooltipWrapper";

import PastActivityFeed from "./PastActivityFeed";
import UpcomingActivityFeed from "./UpcomingActivityFeed";

const baseClass = "activity-card";

export interface IShowActivityDetailsData {
  type: string;
  details?: IActivityDetails;
}

export type ShowActivityDetailsHandler = (
  data: IShowActivityDetailsData
) => void;

const UpcomingTooltip = () => {
  return (
    <TooltipWrapper
      tipContent={
        <>
          Upcoming activities will run as listed. Failure of one activity
          won&apos;t cancel other activities.
          <br />
          <br />
          Currently, only scripts are guaranteed to run in order.
        </>
      }
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
}: IActivityProps) => {
  // TODO: add count to upcoming activities tab when available via API
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
              onDetailsClick={onShowDetails}
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
              onDetailsClick={onShowDetails}
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
