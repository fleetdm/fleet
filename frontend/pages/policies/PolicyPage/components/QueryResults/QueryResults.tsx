import React, { useState } from "react";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";
import classnames from "classnames";
import { format } from "date-fns";
import FileSaver from "file-saver";
import { get } from "lodash";

import convertToCSV from "utilities/convert_to_csv";
import { ICampaign } from "interfaces/campaign";
import { ITarget } from "interfaces/target";

import Button from "components/buttons/Button";
import TabsWrapper from "components/TabsWrapper";
import InfoBanner from "components/InfoBanner";
import QueryResultsHeading from "components/queries/queryResults/QueryResultsHeading";
import AwaitingResults from "components/queries/queryResults/AwaitingResults";

import PolicyQueryTable from "../PolicyQueriesTable/PolicyQueriesTable";
import PolicyQueriesErrorsTable from "../PolicyQueriesErrorsTable/PolicyQueriesErrorsTable";

import DownloadIcon from "../../../../../../assets/images/icon-download-12x12@2x.png";

interface IQueryResultsProps {
  campaign: ICampaign;
  isQueryFinished: boolean;
  policyName?: string;
  onRunQuery: (evt: React.MouseEvent<HTMLButtonElement>) => void;
  onStopQuery: (evt: React.MouseEvent<HTMLButtonElement>) => void;
  setSelectedTargets: (value: ITarget[]) => void;
  goToQueryEditor: () => void;
  targetsTotalCount: number;
}

const baseClass = "query-results";
const CSV_TITLE = "New Policy";
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
  targetsTotalCount,
}: IQueryResultsProps): JSX.Element => {
  const { hosts: hostsOnline, hosts_count: hostsCount, errors } =
    campaign || {};

  const totalRowsCount = get(campaign, ["hosts_count", "successful"], 0);

  const [navTabIndex, setNavTabIndex] = useState(0);

  const onExportQueryResults = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();

    if (hostsOnline) {
      const hostsExport = hostsOnline.map((host) => {
        return {
          host: host.display_name,
          status:
            host.query_results && host.query_results.length ? "yes" : "no",
        };
      });
      const csv = convertToCSV(hostsExport);
      const formattedTime = format(new Date(), "MM-dd-yy hh-mm-ss");
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

      const formattedTime = format(new Date(), "MM-dd-yy hh-mm-ss");
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
      return <AwaitingResults />;
    }

    if (finishedWithNoResults) {
      return (
        <p className="no-results-message">
          Your live query returned no results.
          <span>
            Expecting to see results? Check to see if the host
            {`${targetsTotalCount > 1 ? "s" : ""}`} you targeted reported
            &ldquo;Online&rdquo; or check out the &ldquo;Errors&rdquo; table.
          </span>
        </p>
      );
    }

    return (
      <div className={`${baseClass}__results-table-container`}>
        <InfoBanner>
          Hosts that responded with results are marked <strong>Yes</strong>.
          Hosts that responded with no results are marked <strong>No</strong>.
        </InfoBanner>
        <div className={`${baseClass}__results-table-header`}>
          <span className={`${baseClass}__results-count`}>
            {totalRowsCount} result{totalRowsCount !== 1 && "s"}
          </span>
          <div className={`${baseClass}__results-cta`}>
            <Button
              className={`${baseClass}__export-btn`}
              onClick={onExportQueryResults}
              variant="text-link"
            >
              <>
                Export results <img alt="" src={DownloadIcon} />
              </>
            </Button>
          </div>
        </div>
        <PolicyQueryTable
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
        <div className={`${baseClass}__errors-table-header`}>
          {errors && (
            <span className={`${baseClass}__error-count`}>
              {errors.length} error{errors.length !== 1 && "s"}
            </span>
          )}
          <div className={`${baseClass}__errors-cta`}>
            <Button
              className={`${baseClass}__export-btn`}
              onClick={onExportErrorsResults}
              variant="text-link"
            >
              <>
                Export errors <img alt="" src={DownloadIcon} />
              </>
            </Button>
          </div>
        </div>
        <PolicyQueriesErrorsTable
          isLoading={false}
          errorsList={errors}
          resultsTitle="errors"
        />
      </div>
    );
  };

  const firstTabClass = classnames("react-tabs__tab", "no-count", {
    "errors-empty": !errors || errors?.length === 0,
  });

  return (
    <div className={baseClass}>
      <QueryResultsHeading
        respondedHosts={hostsCount.total}
        targetsTotalCount={targetsTotalCount}
        isQueryFinished={isQueryFinished}
        onClickDone={onQueryDone}
        onClickRunAgain={onRunQuery}
        onClickStop={onStopQuery}
      />
      <TabsWrapper>
        <Tabs selectedIndex={navTabIndex} onSelect={(i) => setNavTabIndex(i)}>
          <TabList>
            <Tab className={firstTabClass}>{NAV_TITLES.RESULTS}</Tab>
            <Tab disabled={!errors?.length}>
              <span>
                {errors?.length > 0 && (
                  <span className="count">{errors.length}</span>
                )}
                {NAV_TITLES.ERRORS}
              </span>
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
