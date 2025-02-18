import React, { useState, useContext, useEffect, useCallback } from "react";
import { Row, Column } from "react-table";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";
import classnames from "classnames";
import FileSaver from "file-saver";
import { QueryContext } from "context/query";
import { useDebouncedCallback } from "use-debounce";

import {
  generateCSVFilename,
  generateCSVQueryResults,
} from "utilities/generate_csv";
import { SUPPORT_LINK } from "utilities/constants";
import { ICampaign, ICampaignError } from "interfaces/campaign";
import { ITarget } from "interfaces/target";

import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";
import TableContainer from "components/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import TabsWrapper from "components/TabsWrapper";
import ShowQueryModal from "components/modals/ShowQueryModal";
import QueryResultsHeading from "components/queries/queryResults/QueryResultsHeading";
import AwaitingResults from "components/queries/queryResults/AwaitingResults";
import InfoBanner from "components/InfoBanner";
import CustomLink from "components/CustomLink";

import generateColumnConfigsFromRows from "./QueryResultsTableConfig";

interface IQueryResultsProps {
  campaign: ICampaign;
  isQueryFinished: boolean;
  isQueryClipped: boolean;
  queryName?: string;
  onRunQuery: () => void;
  onStopQuery: (evt: React.MouseEvent<HTMLButtonElement>) => void;
  setSelectedTargets: (value: ITarget[]) => void;
  goToQueryEditor: () => void;
  targetsTotalCount: number;
}

const baseClass = "query-results";
const CSV_TITLE = "New Query";
const NAV_TITLES = {
  RESULTS: "Results",
  ERRORS: "Errors",
};

const QueryResults = ({
  campaign,
  isQueryFinished,
  isQueryClipped,
  queryName,
  onRunQuery,
  onStopQuery,
  setSelectedTargets,
  goToQueryEditor,
  targetsTotalCount,
}: IQueryResultsProps): JSX.Element => {
  const { lastEditedQueryBody } = useContext(QueryContext);

  const { hosts_count: hostsCount, query_results: queryResults, errors } =
    campaign || {};

  const [navTabIndex, setNavTabIndex] = useState(0);
  const [showQueryModal, setShowQueryModal] = useState(false);
  const [filteredResults, setFilteredResults] = useState<Row[]>([]);
  const [filteredErrors, setFilteredErrors] = useState<Row[]>([]);
  const [resultsColumnConfigs, setResultsColumnConfigs] = useState<Column[]>(
    []
  );
  const [errorColumnConfigs, setErrorColumnConfigs] = useState<
    Column<ICampaignError>[]
  >([]);
  const [queryResultsForTableRender, setQueryResultsForTableRender] = useState(
    queryResults
  );

  // immediately reset results
  const onRunAgain = useCallback(() => {
    setQueryResultsForTableRender([]);
    onRunQuery();
  }, [onRunQuery]);

  const debounceQueryResults = useDebouncedCallback(
    setQueryResultsForTableRender,
    1000,
    { maxWait: 2000 }
  );

  // This is throwing an error not to use hook within a useEffect
  useEffect(() => {
    debounceQueryResults(queryResults);
  }, [queryResults, debounceQueryResults]);

  useEffect(() => {
    if (queryResults && queryResults.length > 0) {
      const newResultsColumnConfigs = generateColumnConfigsFromRows(
        queryResults
      );
      // Update tableHeaders if new headers are found
      if (newResultsColumnConfigs !== resultsColumnConfigs) {
        setResultsColumnConfigs(newResultsColumnConfigs);
      }
    }
  }, [queryResults, lastEditedQueryBody]); // Cannot use tableHeaders as it will cause infinite loop with setTableHeaders

  useEffect(() => {
    if (errorColumnConfigs?.length === 0 && !!errors?.length) {
      setErrorColumnConfigs(generateColumnConfigsFromRows(errors));

      if (errorColumnConfigs && errorColumnConfigs.length > 0) {
        const newErrorColumnConfigs = generateColumnConfigsFromRows(errors);

        // Update errorTableHeaders if new headers are found
        if (newErrorColumnConfigs !== resultsColumnConfigs) {
          setErrorColumnConfigs(newErrorColumnConfigs);
        }
      }
    }
  }, [errors]); // Cannot use errorTableHeaders as it will cause infinite loop with setErrorTableHeaders

  const onExportQueryResults = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();
    FileSaver.saveAs(
      generateCSVQueryResults(
        filteredResults,
        generateCSVFilename(`${queryName || CSV_TITLE} - Results`),
        resultsColumnConfigs
      )
    );
  };

  const onExportErrorsResults = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();

    FileSaver.saveAs(
      generateCSVQueryResults(
        filteredErrors,
        generateCSVFilename(`${queryName || CSV_TITLE} - Errors`),
        errorColumnConfigs
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

  const renderCount = useCallback(
    (tableType: "errors" | "results") => {
      const count =
        tableType === "results"
          ? filteredResults.length
          : filteredErrors.length;

      return <TableCount name={tableType} count={count} />;
    },
    [filteredResults.length, filteredErrors.length]
  );

  const renderTableButtons = (tableType: "results" | "errors") => {
    return (
      <div className={`${baseClass}__results-cta`}>
        <Button
          className={`${baseClass}__show-query-btn`}
          onClick={onShowQueryModal}
          variant="text-icon"
        >
          <>
            Show query <Icon name="eye" />
          </>
        </Button>
        <Button
          className={`${baseClass}__export-btn`}
          onClick={
            tableType === "errors"
              ? onExportErrorsResults
              : onExportQueryResults
          }
          variant="text-icon"
        >
          <>
            Export {tableType}
            <Icon name="download" color="core-fleet-blue" />
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
          defaultSortHeader="host_display_name"
          columnConfigs={
            tableType === "results" ? resultsColumnConfigs : errorColumnConfigs
          }
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
            tableType === "results" ? setFilteredResults : setFilteredErrors
          }
          renderCount={() => renderCount(tableType)}
        />
      </div>
    );
  };

  const renderResultsTab = () => {
    // TODO - clean up these conditions
    const hasNoResultsYet =
      !isQueryFinished &&
      (!queryResults?.length || resultsColumnConfigs === null);
    const finishedWithNoResults = isQueryFinished && !queryResults?.length;

    if (hasNoResultsYet) {
      return <AwaitingResults />;
    }

    if (finishedWithNoResults) {
      return renderNoResults();
    }

    return renderTable(queryResultsForTableRender, "results");
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
        onClickRunAgain={onRunAgain}
        onClickStop={onStopQuery}
      />
      {isQueryClipped && (
        <InfoBanner
          color="yellow"
          cta={<CustomLink url={SUPPORT_LINK} text="Get help" newTab />}
        >
          <div>
            <b>Results clipped.</b> A sample of this query&apos;s results and
            errors is included below. Please target fewer hosts at once to build
            a full set of results.
          </div>
        </InfoBanner>
      )}
      <TabsWrapper>
        <Tabs selectedIndex={navTabIndex} onSelect={(i) => setNavTabIndex(i)}>
          <TabList>
            <Tab className={firstTabClass}>{NAV_TITLES.RESULTS}</Tab>
            <Tab disabled={!errors?.length}>
              <span>
                {errors?.length > 0 && (
                  <span className="count">
                    {errors.length.toLocaleString()}
                  </span>
                )}
                {NAV_TITLES.ERRORS}
              </span>
            </Tab>
          </TabList>
          <TabPanel>{renderResultsTab()}</TabPanel>
          <TabPanel>{renderErrorsTab()}</TabPanel>
        </Tabs>
      </TabsWrapper>
      {showQueryModal && (
        <ShowQueryModal
          query={lastEditedQueryBody}
          onCancel={onShowQueryModal}
        />
      )}
    </div>
  );
};

export default QueryResults;
