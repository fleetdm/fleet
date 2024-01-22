import React, { useRef, useState } from "react";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";
import { useQuery } from "react-query";

import ScriptDetailsModal from "pages/DashboardPage/cards/ActivityFeed/components/ScriptDetailsModal";
import { IActivityDetails } from "interfaces/activity";
import activitiesAPI, {
  IActivitiesResponse,
} from "services/entities/activities";

import Card from "components/Card";
import TabsWrapper from "components/TabsWrapper";
import Spinner from "components/Spinner";
import TooltipWrapper from "components/TooltipWrapper";

import PastActivityFeed from "./PastActivityFeed";
import UpcomingActivityFeed from "./UpcomingActivityFeed";

const baseClass = "activity-card";

const UpcomingTooltip = () => {
  return (
    <TooltipWrapper
      position="top-start"
      tipContent={
        <>
          <p>
            Upcoming activities will run as listed. Failure of one activity
            won’t cancel other activities.
          </p>
          <br />
          <p>Currently, only scripts are guaranteed to run in order.</p>
        </>
      }
      className={`${baseClass}__upcoming-tooltip`}
    >
      Activities run as listed
    </TooltipWrapper>
  );
};

interface IActivityProps {
  activities: any; // TODO: type
  isLoading: boolean;
  onChangeTab: (selectedTab: string) => void;
}

const DEFAULT_PAGE_SIZE = 8;

const Activity = ({ activities, isLoading, onChangeTab }: IActivityProps) => {
  const [pageIndex, setPageIndex] = useState(0);
  const [showScriptDetailsModal, setShowScriptDetailsModal] = useState(false);
  const scriptExecutionId = useRef("");

  const {
    data: activitiesData,
    error: errorActivities,
    isFetching: isFetchingActivities,
  } = useQuery<
    IActivitiesResponse,
    Error,
    IActivitiesResponse,
    Array<{
      scope: string;
      pageIndex: number;
      perPage: number;
    }>
  >(
    [{ scope: "past-activities", pageIndex, perPage: DEFAULT_PAGE_SIZE }],
    ({ queryKey: [{ pageIndex: page, perPage }] }) => {
      return activitiesAPI.loadNext(page, perPage);
    },
    {
      keepPreviousData: true,
      staleTime: 5000,
    }
  );

  const handleDetailsClick = (details: IActivityDetails) => {
    scriptExecutionId.current = details.script_execution_id ?? "";
    setShowScriptDetailsModal(true);
  };

  return (
    <Card borderRadiusSize="large" includeShadow className={baseClass}>
      {isLoading && (
        <div className={`${baseClass}__loading-overlay`}>
          <Spinner />
        </div>
      )}
      <h2>Activity</h2>
      <TabsWrapper>
        <Tabs>
          <TabList>
            <Tab>Past</Tab>
            {/* TODO: count from API */}
            <Tab>
              Upcoming <div className={`${baseClass}__upcoming-count`}>10</div>
            </Tab>
          </TabList>
          <TabPanel>
            <PastActivityFeed
              activities={activities}
              onDetailsClick={handleDetailsClick}
            />
          </TabPanel>
          <TabPanel>
            <UpcomingTooltip />
            <UpcomingActivityFeed
              activities={activities}
              onDetailsClick={handleDetailsClick}
            />
          </TabPanel>
        </Tabs>
      </TabsWrapper>
      {showScriptDetailsModal && (
        <ScriptDetailsModal
          scriptExecutionId={scriptExecutionId.current}
          onCancel={() => setShowScriptDetailsModal(false)}
        />
      )}
    </Card>
  );
};

export default Activity;
