import React, { useRef, useState } from "react";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";

import ScriptDetailsModal from "pages/DashboardPage/cards/ActivityFeed/components/ScriptDetailsModal";
import { IActivityDetails } from "interfaces/activity";

import Card from "components/Card";
import TabsWrapper from "components/TabsWrapper";
import Spinner from "components/Spinner";
import TooltipWrapper from "components/TooltipWrapper";

import PastActivityFeed from "./PastActivityFeed";
import UpcomingActivityFeed from "./UpcomingActivityFeed";
import { IActivitiesResponse } from "services/entities/activities";

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
  activeTab: "past" | "upcoming";
  activities?: IActivitiesResponse; // TODO: type
  isLoading?: boolean;
  isError?: boolean;
  onChangeTab: (index: number, last: number, event: Event) => void;
  onNextPage: () => void;
  onPreviousPage: () => void;
}

const Activity = ({
  activeTab,
  activities,
  isLoading,
  isError,
  onChangeTab,
  onNextPage,
  onPreviousPage,
}: IActivityProps) => {
  const [showScriptDetailsModal, setShowScriptDetailsModal] = useState(false);
  const scriptExecutionId = useRef("");

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
        <Tabs
          selectedIndex={activeTab === "past" ? 0 : 1}
          onSelect={onChangeTab}
        >
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
              isError={isError}
              onNextPage={onNextPage}
              onPreviousPage={onPreviousPage}
            />
          </TabPanel>
          <TabPanel>
            <UpcomingTooltip />
            <UpcomingActivityFeed
              activities={activities}
              onDetailsClick={handleDetailsClick}
              isError={isError}
              onNextPage={onNextPage}
              onPreviousPage={onPreviousPage}
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
