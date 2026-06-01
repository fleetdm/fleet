import React, { useState, useContext, useEffect, useCallback } from "react";
import { CellProps, HeaderProps, Row, Column } from "react-table";
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
import {
  ICampaign,
  ICampaignError,
  ICampaignPerformanceStats,
} from "interfaces/campaign";
import { ITarget } from "interfaces/target";
import { PerformanceImpactIndicatorValue } from "interfaces/schedulable_query";

import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";
import TableContainer from "components/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import ShowQueryModal from "components/modals/ShowQueryModal";
import LiveResultsHeading from "components/queries/LiveResults/LiveResultsHeading";
import AwaitingResults from "components/queries/LiveResults/AwaitingResults";
import EmptyState from "components/EmptyState";
import InfoBanner from "components/InfoBanner";
import CustomLink from "components/CustomLink";
import PerformanceImpactCell from "components/TableContainer/DataTable/PerformanceImpactCell";
import TooltipWrapper from "components/TooltipWrapper";

import LinkCell from "components/TableContainer/DataTable/LinkCell";
import DefaultColumnFilter from "components/TableContainer/DataTable/DefaultColumnFilter";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import PATHS from "router/paths";

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
  // set during target selection, persisted through each step of the flow
  targetsTotalCount: number;
}

const baseClass = "query-results";
const CSV_TITLE = "New Report";
const NAV_TITLES = {
  RESULTS: "Results",
  ERRORS: "Errors",
  PERFORMANCE: "Performance",
};

const getPerformanceIndicator = (stats: ICampaignPerformanceStats) => {
  const cpuTotal = stats.user_time + stats.system_time;
  if (cpuTotal < 2000) {
    return PerformanceImpactIndicatorValue.MINIMAL;
  }
  if (cpuTotal < 4000) {
    return PerformanceImpactIndicatorValue.CONSIDERABLE;
  }
  return PerformanceImpactIndicatorValue.EXCESSIVE;
};

const perfColumnConfigs: Column<ICampaignPerformanceStats>[] = [
  {
    id: "Host",
    Header: (headerProps: HeaderProps<ICampaignPerformanceStats>) => (
      <HeaderCell value="Host" isSortedDesc={headerProps.column.isSortedDesc} />
    ),
    accessor: "host_display_name",
    Cell: (cellProps: CellProps<ICampaignPerformanceStats>) => {
      const hostID = cellProps.row.original.host_id;
      return (
        <LinkCell
          value={cellProps.cell.value}
          path={PATHS.HOST_DETAILS(hostID)}
        />
      );
    },
    Filter: DefaultColumnFilter,
    disableSortBy: false,
    sortType: "caseInsensitive",
  },
  {
    id: "performance_impact",
    Header: () => (
      <TooltipWrapper tipContent="The performance impact of this query on the host.">
        Performance impact
      </TooltipWrapper>
    ),
    disableSortBy: true,
    accessor: (row) => getPerformanceIndicator(row),
    Cell: (cellProps: CellProps<ICampaignPerformanceStats>) => (
      <PerformanceImpactCell
        value={{
          indicator: cellProps.cell.value,
        }}
        isHostSpecific
      />
    ),
  },
];

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

  const {
    uiHostCounts,
    serverHostCounts,
    queryResults,
    errors,
    performanceStats,
  } = campaign || {};

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
    const hostVerb = targetsTotalCount === 1 ? "host is" : "hosts are";
    const errorsMessage = errors?.length ? (
      <>
        {" "}
        or review the <strong>Errors</strong> tab for details
      </>
    ) : null;
    return (
      <EmptyState
        header="No results returned"
        info={
          <>
            Check whether the {hostVerb} online{errorsMessage}.
          </>
        }
      />
    );
  };

  const renderCount = useCallback(
    (tableType: "errors" | "results" | "performance") => {
      const countByType = {
        results: filteredResults.length,
        errors: filteredErrors.length,
        performance: performanceStats?.length ?? 0,
      };

      return <TableCount name={tableType} count={countByType[tableType]} />;
    },
    [filteredResults.length, filteredErrors.length, performanceStats?.length]
  );

  const renderTableButtons = (
    tableType: "results" | "errors" | "performance"
  ) => {
    const exportHandlers: Record<string, typeof onExportQueryResults> = {
      results: onExportQueryResults,
      errors: onExportErrorsResults,
    };

    return (
      <div className={`${baseClass}__results-cta`}>
        <Button
          className={`${baseClass}__show-query-btn`}
          onClick={onShowQueryModal}
          variant="inverse"
        >
          <>
            Show query <Icon name="eye" />
          </>
        </Button>
        {exportHandlers[tableType] && (
          <Button
            className={`${baseClass}__export-btn`}
            onClick={exportHandlers[tableType]}
            variant="inverse"
          >
            <>
              Export {tableType}
              <Icon name="download" />
            </>
          </Button>
        )}
      </div>
    );
  };

  const renderTable = (
    tableData: unknown[],
    tableType: "errors" | "results" | "performance"
  ) => {
    const columnConfigsMap = {
      results: resultsColumnConfigs,
      errors: errorColumnConfigs,
      performance: perfColumnConfigs,
    };
    const setExportRowsMap: Record<string, typeof setFilteredResults> = {
      results: setFilteredResults,
      errors: setFilteredErrors,
    };

    return (
      <div className={`${baseClass}__results-table-container`}>
        <TableContainer
          defaultSortHeader="host_display_name"
          columnConfigs={columnConfigsMap[tableType]}
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
          setExportRows={setExportRowsMap[tableType]}
          renderCount={() => renderCount(tableType)}
          getRowId={(_row, index) => String(index)}
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

  const renderPerformanceTab = () =>
    renderTable(performanceStats, "performance");

  const firstTabClass = classnames("react-tabs__tab", "no-count", {
    "errors-empty": !errors || errors?.length === 0,
  });

  return (
    <div className={baseClass}>
      <LiveResultsHeading
        numHostsTargeted={targetsTotalCount}
        numHostsResponded={uiHostCounts.total}
        numHostsRespondedResults={serverHostCounts.countOfHostsWithResults}
        numHostsRespondedNoErrorsAndNoResults={
          serverHostCounts.countOfHostsWithNoResults
        }
        numHostsRespondedErrors={uiHostCounts.failed}
        isFinished={isQueryFinished}
        onClickClose={onQueryDone}
        onClickRunAgain={onRunAgain}
        onClickStop={onStopQuery}
      />
      {isQueryClipped && (
        <InfoBanner
          color="yellow"
          cta={<CustomLink url={SUPPORT_LINK} text="Get help" newTab />}
        >
          <div>
            <b>Results clipped.</b> A sample of this report&apos;s results and
            errors is included below. Please target fewer hosts at once to build
            a full set of results.
          </div>
        </InfoBanner>
      )}
      <TabNav>
        <Tabs selectedIndex={navTabIndex} onSelect={(i) => setNavTabIndex(i)}>
          <TabList>
            <Tab className={firstTabClass}>{NAV_TITLES.RESULTS}</Tab>
            <Tab disabled={!errors?.length}>
              <TabText count={errors?.length} countVariant="alert">
                {NAV_TITLES.ERRORS}
              </TabText>
            </Tab>
            <Tab disabled={!performanceStats?.length}>
              {NAV_TITLES.PERFORMANCE}
            </Tab>
          </TabList>
          <TabPanel>{renderResultsTab()}</TabPanel>
          <TabPanel>{renderErrorsTab()}</TabPanel>
          <TabPanel>{renderPerformanceTab()}</TabPanel>
        </Tabs>
      </TabNav>
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
