import React, { useState, useEffect } from "react";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";
import classnames from "classnames";
import { format } from "date-fns";
import FileSaver from "file-saver";
import { filter } from "lodash";

import convertToCSV from "utilities/convert_to_csv";
import { ICampaign } from "interfaces/campaign";
import { ITarget } from "interfaces/target";

import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
import TableContainer from "components/TableContainer";
import TabsWrapper from "components/TabsWrapper";
import TooltipWrapper from "components/TooltipWrapper";
import ShowQueryModal from "./ShowQueryModal";
import DownloadIcon from "../../../../../../assets/images/icon-download-12x12@2x.png";
import EyeIcon from "../../../../../../assets/images/icon-eye-16x16@2x.png";

import resultsTableHeaders from "./QueryResultsTableConfig";

interface IQueryResultsProps {
  campaign: ICampaign;
  isQueryFinished: boolean;
  onRunQuery: (evt: React.MouseEvent<HTMLButtonElement>) => void;
  onStopQuery: (evt: React.MouseEvent<HTMLButtonElement>) => void;
  setSelectedTargets: (value: ITarget[]) => void;
  goToQueryEditor: () => void;
  targetsTotalCount: number;
}

const baseClass = "query-results";
const CSV_QUERY_TITLE = "Query Results";
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
  onRunQuery,
  onStopQuery,
  setSelectedTargets,
  goToQueryEditor,
  targetsTotalCount,
}: IQueryResultsProps): JSX.Element => {
  const { hosts_count: hostsCount, query_results: queryResults, errors } =
    campaign || {};

  const [pageTitle, setPageTitle] = useState<string>(PAGE_TITLES.RUNNING);
  const [navTabIndex, setNavTabIndex] = useState(0);
  const [
    targetsRespondedPercent,
    setTargetsRespondedPercent,
  ] = useState<number>(0);
  const [showQueryModal, setShowQueryModal] = useState<boolean>(false);

  useEffect(() => {
    const calculatePercent =
      targetsTotalCount > 0
        ? Math.round((campaign.hosts_count.total / targetsTotalCount) * 100)
        : 0;
    setTargetsRespondedPercent(calculatePercent);
  }, [campaign]);

  useEffect(() => {
    if (isQueryFinished) {
      setPageTitle(PAGE_TITLES.FINISHED);
    } else {
      setPageTitle(PAGE_TITLES.RUNNING);
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

      const formattedTime = format(new Date(), "MM-dd-yy hh-mm-ss");
      const filename = `${CSV_QUERY_TITLE} (${formattedTime}).csv`;
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

      const formattedTime = format(new Date(), "MM-dd-yy hh-mm-ss");
      const filename = `${CSV_QUERY_TITLE} Errors (${formattedTime}).csv`;
      const file = new global.window.File([csv], filename, {
        type: "text/csv",
      });

      FileSaver.saveAs(file);
    }
  };

  const onExport = (type: "hosts" | "errors") => (
    evt: React.MouseEvent<HTMLButtonElement>
  ) => {
    evt.preventDefault();

    const exportData = type === "errors" ? errors : queryResults;
    if (!exportData) {
      // This case should never happen because UI hides the export button if there are no results
      return;
    }

    const csv = convertToCSV(exportData, (fields: string[]) => {
      const result = filter(fields, (f) => f !== "host_hostname");
      result.unshift("host_hostname");
      return result;
    });
    const formattedTime = format(new Date(), "MM-dd-yy hh-mm-ss");
    const filename = `${CSV_QUERY_TITLE} ${
      type === "errors" ? "Errors" : ""
    } (${formattedTime}).csv`;
    const file = new global.window.File([csv], filename, {
      type: "text/csv",
    });

    FileSaver.saveAs(file);
  };

  const onShowQueryModal = () => {
    setShowQueryModal(!showQueryModal);
  };

  const onQueryDone = () => {
    setSelectedTargets([]);
    goToQueryEditor();
  };

  const renderNoResults = () => {
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
  };

  const renderTableButtons = (type: "hosts" | "errors") => {
    return (
      <div className={`${baseClass}__results-cta`}>
        <Button
          className={`${baseClass}__show-query-btn`}
          onClick={onShowQueryModal}
          variant="text-link"
        >
          <>
            Show query <img alt="Show query" src={EyeIcon} />
          </>
        </Button>
        <Button
          className={`${baseClass}__export-btn`}
          onClick={
            type === "errors" ? onExportErrorsResults : onExportQueryResults
          }
          variant="text-link"
        >
          <>
            {`Export ${type}`} <img alt={`Export ${type}`} src={DownloadIcon} />
          </>
        </Button>
      </div>
    );
  };

  const renderTable = (tableData: unknown[], type: "hosts" | "errors") => {
    return (
      <div className={`${baseClass}__results-table-container`}>
        <TableContainer
          columns={resultsTableHeaders(tableData || [])}
          data={tableData || []}
          emptyComponent={renderNoResults}
          isLoading={false}
          isClientSidePagination
          isClientSideFilter
          isMultiColumnFilter
          showMarkAllPages={false}
          isAllPagesSelected={false}
          resultsTitle={type}
          customControl={() => renderTableButtons(type)}
        />
      </div>
    );
  };

  const renderResultsTab = () => {
    const hasNoResultsYet = !isQueryFinished && !queryResults?.length;
    const finishedWithNoResults =
      isQueryFinished && (!queryResults?.length || !hostsCount.successful);

    if (hasNoResultsYet) {
      return <Spinner />;
    }

    if (finishedWithNoResults) {
      return renderNoResults();
    }

    return renderTable(queryResults, "hosts");
  };

  const renderErrorsTab = () => renderTable(errors, "errors");

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
          <span>{targetsTotalCount}</span>&nbsp;hosts targeted&nbsp; (
          {targetsRespondedPercent}%&nbsp;
          <TooltipWrapper
            tipContent={`
                Hosts that respond may<br /> return results, errors, or <br />no results`}
          >
            responded
          </TooltipWrapper>
          )
        </div>
      </div>
      {isQueryFinished ? renderFinishedButtons() : renderStopQueryButton()}
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
          <TabPanel>{renderResultsTab()}</TabPanel>
          <TabPanel>{renderErrorsTab()}</TabPanel>
        </Tabs>
      </TabsWrapper>
      {showQueryModal && <ShowQueryModal onCancel={onShowQueryModal} />}
    </div>
  );
};

export default QueryResults;
