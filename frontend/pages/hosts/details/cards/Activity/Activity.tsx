import classnames from "classnames";
import React from "react";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";

import { ShowActivityDetailsHandler } from "components/ActivityItem/ActivityItem";
import Card from "components/Card";
import CardHeader from "components/CardHeader";
import Spinner from "components/Spinner";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import TooltipWrapper from "components/TooltipWrapper";
import { IHostUpcomingActivity } from "interfaces/activity";
import {
  IHostPastActivitiesResponse,
  IHostUpcomingActivitiesResponse,
} from "services/entities/activities";
import { IGetCommandsResponse } from "services/entities/command";

import CommandFeed from "./CommandFeed";
import {
  CancelCommandHandler,
  ShowCommandDetailsHandler,
} from "./CommandItem/CommandItem";
import MDMCommandsToggle from "./MDMCommandsToggle";
import PastActivityFeed from "./PastActivityFeed";
import UpcomingActivityFeed from "./UpcomingActivityFeed";

const baseClass = "host-activity-card";

const UpcomingTooltip = () => {
  return (
    <TooltipWrapper
      tipContent={
        <>
          Software and scripts always run in order. Each waits until the
          previous one runs successfully or fails all retries.
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
  /** When true, the toggle can't be flipped. */
  isMDMCommandsToggleDisabled?: boolean;
  /** Shown on hover over the toggle's label, e.g. to explain why it's
   * disabled. */
  mdmCommandsToggleTooltip?: JSX.Element | string;
  onChangeTab: (index: number, last: number, event: Event) => void;
  onNextPage: () => void;
  onPreviousPage: () => void;
  onShowDetails: ShowActivityDetailsHandler;
  onShowCommandDetails: ShowCommandDetailsHandler;
  onCancel: (activity: IHostUpcomingActivity) => void;
  /** When provided, cancelable pending MDM commands in the Upcoming tab
   * render a cancel button. */
  onCancelCommand?: CancelCommandHandler;
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
  isMDMCommandsToggleDisabled = false,
  mdmCommandsToggleTooltip,
  onChangeTab,
  onNextPage,
  onPreviousPage,
  onShowDetails,
  onShowCommandDetails,
  onCancel,
  onCancelCommand,
  onShowMDMCommands,
  onHideMDMCommands,
}: IActivityProps) => {
  const classNames = classnames(baseClass, className);

  const commandCount = commands?.count ?? 0;

  // The commands queries keep previous data across host navigations, so
  // `commands` can still hold the last host's response on a host we can't
  // fetch commands for. Only ever show a feed we also show the toggle for.
  const canShowCommandFeed = showMDMCommandsToggle && showMDMCommands;

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
                disabled={isMDMCommandsToggleDisabled}
                labelTooltip={mdmCommandsToggleTooltip}
                onToggleMDMCommands={
                  showMDMCommands ? onHideMDMCommands : onShowMDMCommands
                }
              />
            )}
            {canShowCommandFeed && commands ? (
              <CommandFeed
                commands={commands}
                emptyDescription="Completed MDM commands will appear here."
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
                disabled={isMDMCommandsToggleDisabled}
                labelTooltip={mdmCommandsToggleTooltip}
                onToggleMDMCommands={
                  showMDMCommands ? onHideMDMCommands : onShowMDMCommands
                }
              />
            )}
            {canShowCommandFeed && commands ? (
              <CommandFeed
                commands={commands}
                emptyDescription="Pending MDM commands will appear here."
                onShowDetails={onShowCommandDetails}
                onNextPage={onNextPage}
                onPreviousPage={onPreviousPage}
                onCancelCommand={onCancelCommand}
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
