import React, { useState, useEffect } from "react";
import { Dispatch } from "redux";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";
import moment from "moment";
import FileSaver from "file-saver";
import { filter, get } from "lodash";

import PATHS from "router/paths"; // @ts-ignore
import convertToCSV from "utilities/convert_to_csv";
import { ICampaign } from "interfaces/campaign";

import Button from "components/buttons/Button";
import { push } from "react-router-redux";

interface IQueryResultsProps {
  campaign: ICampaign;
  isQueryFinished: boolean;
  onRunQuery: (evt: React.MouseEvent<HTMLButtonElement>) => void;
  onStopQuery: (evt: React.MouseEvent<HTMLButtonElement>) => void;
  dispatch: Dispatch;
}

const baseClass = "query-results";
const PAGE_TITLES = {
  RUNNING: "Querying selected hosts",
  FINISHED: "Query finished",
};

const queryResultsNavTitles: string[] = ["Results", "Errors"];

const QueryResults = ({
  campaign,
  isQueryFinished,
  onRunQuery,
  onStopQuery,
  dispatch,
}: IQueryResultsProps) => {
  const { hosts_count: hostsCount, query_results: queryResults, errors } =
    campaign || {};

  const totalHostsOnline = get(campaign, ["totals", "online"], 0);
  const totalHostsOffline = get(campaign, ["totals", "offline"], 0);
  const totalHostsCount = get(campaign, ["totals", "count"], 0);
  const totalRowsCount = get(campaign, ["query_results", "length"], 0);
  const campaignIsEmpty = !hostsCount.successful && hostsCount.successful !== 0;
  const onlineTotalText = `${totalRowsCount} result${
    totalRowsCount === 1 ? "" : "s"
  }`;
  const errorsTotalText = `${errors?.length || 0} result${
    errors?.length === 1 ? "" : "s"
  }`;

  const [pageTitle, setPageTitle] = useState<string>(PAGE_TITLES.RUNNING);
  const [csvQueryName, setCsvQueryName] = useState<string>("Query Results");
  const [navTabIndex, setNavTabIndex] = useState(0);

  useEffect(() => {
    if (isQueryFinished) {
      setPageTitle(PAGE_TITLES.RUNNING);
    } else {
      setPageTitle(PAGE_TITLES.FINISHED);
    }
  }, [isQueryFinished]);

  const onExportQueryResults = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();

    if (queryResults) {
      const csv = convertToCSV(queryResults, (fields: string[]) => {
        const result = filter(fields, (f) => f !== "host_hostname");
        result.unshift("host_hostname");

        return result;
      });

      const formattedTime = moment(new Date()).format("MM-DD-YY hh-mm-ss");
      const filename = `${csvQueryName} (${formattedTime}).csv`;
      const file = new global.window.File([csv], filename, {
        type: "text/csv",
      });

      FileSaver.saveAs(file);
    }
  };

  const onExportErrorsResults = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();

    if (errors) {
      const csv = convertToCSV(errors, (fields: string[]) => {
        const result = filter(fields, (f) => f !== "host_hostname");
        result.unshift("host_hostname");

        return result;
      });

      const formattedTime = moment(new Date()).format("MM-DD-YY hh-mm-ss");
      const filename = `${csvQueryName} Errors (${formattedTime}).csv`;
      const file = new global.window.File([csv], filename, {
        type: "text/csv",
      });

      FileSaver.saveAs(file);
    }
  };

  const doneAndRunQueryBtns = (
    <div className={`${baseClass}__btn-wrapper`}>
      <Button
        className={`${baseClass}__done-btn`}
        onClick={() => dispatch(push(PATHS.MANAGE_QUERIES))}
        variant="brand"
      >
        Done
      </Button>
      <Button
        className={`${baseClass}__run-btn`}
        onClick={onRunQuery}
        variant="blue-green"
      >
        Run again
      </Button>
    </div>
  );

  const stopQueryBtn = (
    <div className={`${baseClass}__btn-wrapper`}>
      <Button
        className={`${baseClass}__stop-btn`}
        onClick={onStopQuery}
        variant="alert"
      >
        Stop
      </Button>
    </div>
  );

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__wrapper`}>
        <h1>{pageTitle}</h1>
        <div className={`${baseClass}__text-wrapper`}>
          <span className={`${baseClass}__text-online`}>
            Online: {totalHostsOnline} hosts / {onlineTotalText}
          </span>
          <span className={`${baseClass}__text-offline`}>
            Offline: {totalHostsOffline} hosts / 0 results
          </span>
          <span className={`${baseClass}__text-error`}>
            Errors: {hostsCount.failed} hosts / {errorsTotalText}
          </span>
        </div>
      </div>
      {isQueryFinished ? doneAndRunQueryBtns : stopQueryBtn}
      {isQueryFinished && (
        <div className={`${baseClass}__nav-header`}>
          <Tabs selectedIndex={navTabIndex} onSelect={(i) => setNavTabIndex(i)}>
            <TabList>
              {queryResultsNavTitles.map((title) => {
                // Bolding text when the tab is active causes a layout shift
                // so we add a hidden pseudo element with the same text string
                return (
                  <Tab key={title} data-text={title}>
                    {title}
                  </Tab>
                );
              })}
            </TabList>
            <TabPanel>
              <div>Results Table</div>
            </TabPanel>
            <TabPanel>
              <div>Errors Table</div>
            </TabPanel>
          </Tabs>
        </div>
      )}
    </div>
  );
};

export default QueryResults;
