import React, { useState } from "react";
import { Row } from "react-table";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";
import classnames from "classnames";
import { format } from "date-fns";
import FileSaver from "file-saver";

import convertToCSV from "utilities/convert_to_csv";
import { ICampaign } from "interfaces/campaign";
import { ITarget } from "interfaces/target";

import Button from "components/buttons/Button";
import TableContainer from "components/TableContainer";
import TabsWrapper from "components/TabsWrapper";
import QueryResultsHeading from "components/queries/queryResults/QueryResultsHeading";
import AwaitingResults from "components/queries/queryResults/AwaitingResults";

import ShowQueryModal from "./ShowQueryModal";
import resultsTableHeaders from "./QueryResultsTableConfig";

import DownloadIcon from "../../../../../../assets/images/icon-download-12x12@2x.png";
import EyeIcon from "../../../../../../assets/images/icon-eye-16x16@2x.png";

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
const NAV_TITLES = {
  RESULTS: "Results",
  ERRORS: "Errors",
};

const reorderCSVFields = (fields: string[]) => {
  const result = fields.filter((field) => field !== "host_hostname");
  result.unshift("host_hostname");

  return result;
};

const generateExportCSVFile = (rows: Row[], filename: string) => {
  return new global.window.File(
    [
      convertToCSV(
        rows.map((r) => r.original),
        reorderCSVFields
      ),
    ],
    filename,
    {
      type: "text/csv",
    }
  );
};

const generateExportFilename = (descriptor: string) => {
  return `${descriptor} (${format(new Date(), "MM-dd-yy hh-mm-ss")}).csv`;
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

  const [navTabIndex, setNavTabIndex] = useState(0);
  const [showQueryModal, setShowQueryModal] = useState(false);
  const [filteredResults, setFilteredResults] = useState<Row[]>([]);
  const [filteredErrors, setFilteredErrors] = useState<Row[]>([]);

  const onExportQueryResults = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();

    FileSaver.saveAs(
      generateExportCSVFile(
        filteredResults,
        generateExportFilename(CSV_QUERY_TITLE)
      )
    );
  };

  const onExportErrorsResults = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();

    FileSaver.saveAs(
      generateExportCSVFile(
        filteredErrors,
        generateExportFilename(`${CSV_QUERY_TITLE} Errors`)
      )
    );
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
          Expecting to see results? Check to see if the host
          {`${targetsTotalCount > 1 ? "s" : ""}`} you targeted reported
          &ldquo;Online&rdquo; or check out the &ldquo;Errors&rdquo; table.
        </span>
      </p>
    );
  };

  const renderTableButtons = (tableType: "results" | "errors") => {
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
            tableType === "errors"
              ? onExportErrorsResults
              : onExportQueryResults
          }
          variant="text-link"
        >
          <>
            {`Export ${tableType}`}{" "}
            <img alt={`Export ${tableType}`} src={DownloadIcon} />
          </>
        </Button>
      </div>
    );
  };

  const renderTable = (
    tableData: unknown[],
    tableType: "errors" | "results"
  ) => {
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
          resultsTitle={tableType}
          customControl={() => renderTableButtons(tableType)}
          setExportRows={
            tableType === "errors" ? setFilteredErrors : setFilteredResults
          }
        />
      </div>
    );
  };

  const renderResultsTab = () => {
    const hasNoResultsYet = !isQueryFinished && !queryResults?.length;
    const finishedWithNoResults =
      isQueryFinished && (!queryResults?.length || !hostsCount.successful);

    if (hasNoResultsYet) {
      return <AwaitingResults />;
    }

    if (finishedWithNoResults) {
      return renderNoResults();
    }

    return renderTable(queryResults, "results");
  };

  const renderErrorsTab = () => renderTable(errors, "errors");

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
          <TabPanel>{renderResultsTab()}</TabPanel>
          <TabPanel>{renderErrorsTab()}</TabPanel>
        </Tabs>
      </TabsWrapper>
      {showQueryModal && <ShowQueryModal onCancel={onShowQueryModal} />}
    </div>
  );
};

export default QueryResults;
