import React, { useState, useEffect } from "react";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";
import moment from "moment";
import classnames from "classnames";
import FileSaver from "file-saver";
import { get } from "lodash";

// @ts-ignore
import convertToCSV from "utilities/convert_to_csv"; // @ts-ignore
import { ICampaign } from "interfaces/campaign";
import { ITarget } from "interfaces/target";

import Button from "components/buttons/Button"; // @ts-ignore
import Spinner from "components/Spinner";
import TabsWrapper from "components/TabsWrapper";
import InfoBanner from "components/InfoBanner";
import PolicyQueryListWrapper from "../PolicyQueriesListWrapper/PolicyQueriesListWrapper";
import PolicyQueriesErrorsListWrapper from "../PolicyQueriesErrorsListWrapper/PolicyQueriesErrorsListWrapper";

import DownloadIcon from "../../../../../../assets/images/icon-download-12x12@2x.png";

interface IQueryResultsProps {
  campaign: ICampaign;
  isQueryFinished: boolean;
  policyName?: string;
  onRunQuery: (evt: React.MouseEvent<HTMLButtonElement>) => void;
  onStopQuery: (evt: React.MouseEvent<HTMLButtonElement>) => void;
  setSelectedTargets: (value: ITarget[]) => void;
  goToQueryEditor: () => void;
}

const baseClass = "query-results";
const CSV_TITLE = "New Policy";
const PAGE_TITLES = {
  RUNNING: "Querying selected hosts",
  FINISHED: "Query finished",
};
const NAV_TITLES = {
  RESULTS: "Results",
  ERRORS: "Errors",
};

const QueryResults = ({
  campaign,
  isQueryFinished,
  policyName,
  onRunQuery,
  onStopQuery,
  setSelectedTargets,
  goToQueryEditor,
}: IQueryResultsProps): JSX.Element => {
  const { hosts: hostsOnline, hosts_count: hostsCount, errors } =
    campaign || {};

  const totalHostsOnline = get(campaign, ["totals", "online"], 0);
  const totalHostsOffline = get(campaign, ["totals", "offline"], 0);
  const totalRowsCount = get(campaign, ["query_results", "length"], 0);
  const onlineTotalText = `${totalRowsCount} result${
    totalRowsCount === 1 ? "" : "s"
  }`;
  const errorsTotalText = `${errors?.length || 0} result${
    errors?.length === 1 ? "" : "s"
  }`;

  const [pageTitle, setPageTitle] = useState<string>(PAGE_TITLES.RUNNING);
  const [navTabIndex, setNavTabIndex] = useState(0);

  useEffect(() => {
    if (isQueryFinished) {
      setPageTitle(PAGE_TITLES.FINISHED);
    } else {
      setPageTitle(PAGE_TITLES.RUNNING);
    }
  }, [isQueryFinished]);

  const onExportQueryResults = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();

    if (hostsOnline) {
      const hostsExport = hostsOnline.map((host) => {
        return {
          hostname: host.hostname,
          status:
            host.query_results && host.query_results.length ? "yes" : "no",
        };
      });
      const csv = convertToCSV(hostsExport);
      const formattedTime = moment(new Date()).format("MM-DD-YY hh-mm-ss");
      const filename = `${policyName || CSV_TITLE} (${formattedTime}).csv`;
      const file = new global.window.File([csv], filename, {
        type: "text/csv",
      });

      FileSaver.saveAs(file);
    }
  };

  const onExportErrorsResults = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();

    if (errors) {
      const csv = convertToCSV(errors);

      const formattedTime = moment(new Date()).format("MM-DD-YY hh-mm-ss");
      const filename = `${
        policyName || CSV_TITLE
      } Errors (${formattedTime}).csv`;
      const file = new global.window.File([csv], filename, {
        type: "text/csv",
      });

      FileSaver.saveAs(file);
    }
  };

  const onQueryDone = () => {
    setSelectedTargets([]);
    goToQueryEditor();
  };

  const renderTable = () => {
    const emptyResults =
      !hostsOnline || !hostsOnline.length || !hostsCount.successful;
    const hasNoResultsYet = !isQueryFinished && emptyResults;
    const finishedWithNoResults =
      isQueryFinished && (!hostsCount.successful || emptyResults);

    if (hasNoResultsYet) {
      return (
        <div className={`${baseClass}__loading-spinner`}>
          <Spinner />
        </div>
      );
    }

    if (finishedWithNoResults) {
      return (
        <p className="no-results-message">
          Your live query returned no results.
          <span>
            Expecting to see results? Check to see if the hosts you targeted
            reported &ldquo;Online&rdquo; or check out the &ldquo;Errors&rdquo;
            table.
          </span>
        </p>
      );
    }

    return (
      <div className={`${baseClass}__results-table-container`}>
        <InfoBanner>
          Host that responded with results are marked <strong>Yes</strong>.
          Hosts that responded with no results are marked <strong>No</strong>.
        </InfoBanner>
        <Button
          className={`${baseClass}__export-btn`}
          onClick={onExportQueryResults}
          variant="text-link"
        >
          <>
            Export hosts <img alt="" src={DownloadIcon} />
          </>
        </Button>
        <PolicyQueryListWrapper
          isLoading={false}
          policyHostsList={hostsOnline}
          resultsTitle="hosts"
        />
      </div>
    );
  };

  const renderErrorsTable = () => {
    return (
      <div className={`${baseClass}__error-table-container`}>
        <Button
          className={`${baseClass}__export-btn`}
          onClick={onExportErrorsResults}
          variant="text-link"
        >
          <>
            Export errors <img alt="" src={DownloadIcon} />
          </>
        </Button>
        <PolicyQueriesErrorsListWrapper
          isLoading={false}
          errorsList={errors}
          resultsTitle="errors"
        />
      </div>
    );
  };

  const renderFinishedButtons = () => (
    <div className={`${baseClass}__btn-wrapper`}>
      <Button
        className={`${baseClass}__done-btn`}
        onClick={onQueryDone}
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

  const renderStopQueryButton = () => (
    <div className={`${baseClass}__btn-wrapper`}>
      <Button
        className={`${baseClass}__stop-btn`}
        onClick={onStopQuery}
        variant="alert"
      >
        <>
          <Spinner isInButton />
          Stop
        </>
      </Button>
    </div>
  );

  const firstTabClass = classnames("react-tabs__tab", "no-count", {
    "errors-empty": !errors || errors?.length === 0,
  });

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
      {isQueryFinished ? renderFinishedButtons() : renderStopQueryButton()}
      <TabsWrapper>
        <Tabs selectedIndex={navTabIndex} onSelect={(i) => setNavTabIndex(i)}>
          <TabList>
            <Tab className={firstTabClass}>{NAV_TITLES.RESULTS}</Tab>
            <Tab disabled={!errors?.length}>
              {errors?.length > 0 && (
                <span className="count">{errors.length}</span>
              )}
              {NAV_TITLES.ERRORS}
            </Tab>
          </TabList>
          <TabPanel>{renderTable()}</TabPanel>
          <TabPanel>{renderErrorsTable()}</TabPanel>
        </Tabs>
      </TabsWrapper>
    </div>
  );
};

export default QueryResults;
