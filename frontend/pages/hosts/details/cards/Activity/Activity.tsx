import React from "react";
import classnames from "classnames";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";

import { IHostUpcomingActivity } from "interfaces/activity";
import {
  IHostPastActivitiesResponse,
  IHostUpcomingActivitiesResponse,
} from "services/entities/activities";
import { IGetCommandsResponse } from "services/entities/command";

import Card from "components/Card";
import CardHeader from "components/CardHeader";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import Spinner from "components/Spinner";
import TooltipWrapper from "components/TooltipWrapper";
import { ShowActivityDetailsHandler } from "components/ActivityItem/ActivityItem";

import PastActivityFeed from "./PastActivityFeed";
import UpcomingActivityFeed from "./UpcomingActivityFeed";
import MDMCommandsToggle from "./MDMCommandsToggle";
import PastCommandFeed from "./PastCommandFeed";
import UpcomingCommandFeed from "./UpcomingCommandFeed";
import CommandFeed from "./CommandFeed";
import { ShowCommandDetailsHandler } from "./CommandItem/CommandItem";

const baseClass = "host-activity-card";

const UpcomingTooltip = () => {
  return (
    <TooltipWrapper
      tipContent={
        <>
          Failure of one activity won&apos;t cancel other activities.
          <br />
          <br />
          Currently, only software and scripts are guaranteed to run in order.
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
  showMDMCommandsToggle: boolean;
  showMDMCommands: boolean;
  activities?: IHostPastActivitiesResponse | IHostUpcomingActivitiesResponse;
  commands?: IGetCommandsResponse;
  isLoading?: boolean;
  isError?: boolean;
  className?: string;
  /** The count displayed in the Upcoming tab. It consists of the amount of
   * upcoming activities and mdm commands. */
  upcomingCount: number;
  canCancelActivities: boolean;
  onChangeTab: (index: number, last: number, event: Event) => void;
  onNextPage: () => void;
  onPreviousPage: () => void;
  onShowDetails: ShowActivityDetailsHandler;
  onShowCommandDetails: ShowCommandDetailsHandler;
  onCancel: (activity: IHostUpcomingActivity) => void;
  onShowMDMCommands: () => void;
  onHideMDMCommands: () => void;
}

const Activity = ({
  activeTab,
  showMDMCommandsToggle,
  showMDMCommands,
  activities,
  commands,
  isLoading,
  isError,
  className,
  upcomingCount,
  canCancelActivities,
  onChangeTab,
  onNextPage,
  onPreviousPage,
  onShowDetails,
  onShowCommandDetails,
  onCancel,
  onShowMDMCommands,
  onHideMDMCommands,
}: IActivityProps) => {
  const classNames = classnames(baseClass, className);

  const commandCount = commands?.count ?? 0;

  return (
    <Card
      borderRadiusSize="xxlarge"
      paddingSize="xlarge"
      className={classNames}
    >
      {isLoading && (
        <div className={`${baseClass}__loading-overlay`}>
          <Spinner centered />
        </div>
      )}
      <div className={`${baseClass}__header`}>
        <CardHeader header="Activity" />
        {activeTab === "upcoming" && <UpcomingTooltip />}
      </div>
      <TabNav secondary>
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
          <TabPanel className={`${baseClass}__tab-panel`}>
            {showMDMCommandsToggle && (
              <MDMCommandsToggle
                showMDMCommands={showMDMCommands}
                onToggleMDMCommands={
                  showMDMCommands ? onHideMDMCommands : onShowMDMCommands
                }
              />
            )}
            {showMDMCommands && commands ? (
              <CommandFeed
                commands={commands}
                onShowDetails={onShowCommandDetails}
                onNextPage={onNextPage}
                onPreviousPage={onPreviousPage}
              />
            ) : (
              <PastActivityFeed
                activities={
                  activities as IHostPastActivitiesResponse | undefined
                }
                onShowDetails={onShowDetails}
                isError={isError}
                onNextPage={onNextPage}
                onPreviousPage={onPreviousPage}
              />
            )}
          </TabPanel>
          <TabPanel className={`${baseClass}__tab-panel`}>
            {showMDMCommandsToggle && (
              <MDMCommandsToggle
                showMDMCommands={showMDMCommands}
                commandCount={commandCount}
                onToggleMDMCommands={
                  showMDMCommands ? onHideMDMCommands : onShowMDMCommands
                }
              />
            )}
            {showMDMCommands && commands ? (
              <CommandFeed
                commands={commands}
                onShowDetails={onShowCommandDetails}
                onNextPage={onNextPage}
                onPreviousPage={onPreviousPage}
              />
            ) : (
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
            )}
          </TabPanel>
        </Tabs>
      </TabNav>
    </Card>
  );
};

export default Activity;
