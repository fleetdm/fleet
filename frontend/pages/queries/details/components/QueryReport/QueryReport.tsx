import FileSaver from "file-saver";
import React, { useState, useMemo, useCallback } from "react";
import { Row, Column } from "react-table";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import EmptyState from "components/EmptyState";
import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import { generateResultsCountText } from "components/TableContainer/utilities/TableContainerUtils";
import { notify } from "components/ToastNotification";
import TooltipWrapper from "components/TooltipWrapper";
import { IQueryReport, IQueryReportResultRow } from "interfaces/query_report";
import PATHS from "router/paths";
import {
  generateCSVFilename,
  generateCSVQueryResults,
} from "utilities/generate_csv";

import generateReportColumnConfigsFromResults from "./QueryReportTableConfig";

interface IQueryReportProps {
  queryReport?: IQueryReport;
  queryId: number;
  queryName?: string;
  isClipped?: boolean;
  canLiveQuery?: boolean;
  isFetching?: boolean;
  pageIndex: number;
  pageSize: number;
  searchQuery: string;
  /** API order key, e.g. "host_name", "last_fetched" or a result column name. */
  sortHeader: string;
  sortDirection: string;
  onQueryChange: (newTableQuery: ITableQueryData) => void;
  /** Fetches every result matching the current search and sort, for export. */
  loadAllResults: () => Promise<IQueryReportResultRow[]>;
}

const baseClass = "query-report";
const CSV_TITLE = "Report";

// The table renames the host column to "Host"; the API sorts it as "host_name".
const HOST_TABLE_COLUMN_ID = "Host";
const HOST_API_ORDER_KEY = "host_name";

const toTableSortHeader = (apiOrderKey: string) =>
  apiOrderKey === HOST_API_ORDER_KEY ? HOST_TABLE_COLUMN_ID : apiOrderKey;
const toApiOrderKey = (tableSortHeader: string) =>
  tableSortHeader === HOST_TABLE_COLUMN_ID
    ? HOST_API_ORDER_KEY
    : tableSortHeader;

const flattenResults = (results: IQueryReportResultRow[]) => {
  return results.map((result: IQueryReportResultRow) => {
    const hostInfoColumns = {
      host_id: result.host_id,
      host_display_name: result.host_name,
      last_fetched: result.last_fetched,
    };

    // hostInfoColumns displays the host metadata that is returned with every query
    // result.columns are the variable columns returned by the API that differ per query
    return { ...hostInfoColumns, ...result.columns };
  });
};

const QueryReport = ({
  queryReport,
  queryId,
  queryName,
  isClipped,
  canLiveQuery,
  isFetching = false,
  pageIndex,
  pageSize,
  searchQuery,
  sortHeader,
  sortDirection,
  onQueryChange,
  loadAllResults,
}: IQueryReportProps): JSX.Element => {
  const [isExporting, setIsExporting] = useState(false);

  const results = useMemo(() => queryReport?.results ?? [], [queryReport]);
  const totalCount = queryReport?.count ?? results.length;

  const columnConfigs = useMemo<Column[]>(() => {
    if (results.length) {
      return generateReportColumnConfigsFromResults(
        flattenResults(results),
        queryId
      );
    }
    return [];
  }, [results, queryId]);

  const onExportQueryResults = async (
    evt: React.MouseEvent<HTMLButtonElement>
  ) => {
    evt.preventDefault();
    setIsExporting(true);
    try {
      const allResults = flattenResults(await loadAllResults());
      // Columns come from the full export, not the current page: a column only
      // some hosts return must still make it into the CSV header.
      const exportColumns = generateReportColumnConfigsFromResults(
        allResults,
        queryId
      );
      // generateCSVQueryResults reads rows the way react-table exposes them.
      const rows = allResults.map((original) => ({ original } as Row));
      FileSaver.saveAs(
        generateCSVQueryResults(
          rows,
          generateCSVFilename(`${queryName || CSV_TITLE} - Report`),
          exportColumns
        )
      );
    } catch (error) {
      console.error(error);
      notify.error("Could not export results. Please try again.", {
        response: error,
      });
    } finally {
      setIsExporting(false);
    }
  };

  const onTableQueryChange = useCallback(
    (newTableQuery: ITableQueryData) => {
      onQueryChange({
        ...newTableQuery,
        sortHeader: toApiOrderKey(newTableQuery.sortHeader),
      });
    },
    [onQueryChange]
  );

  const renderTableButtons = () => {
    return (
      <div className={`${baseClass}__results-cta`}>
        <Button
          className={`${baseClass}__export-btn`}
          onClick={onExportQueryResults}
          variant="secondary"
          size="small"
          icon="download"
          iconPosition="right"
          isLoading={isExporting}
          disabled={isExporting || totalCount === 0}
        >
          Export results
        </Button>
      </div>
    );
  };

  const renderResultsCount = useCallback(() => {
    if (isClipped) {
      return (
        <>
          <TooltipWrapper
            tipContent={
              <>
                This report is full. Hosts already in the report keep updating,
                but results from other hosts aren&apos;t saved. <br />
                <br />
                You can reset this report by updating the report&apos;s SQL, or
                by temporarily enabling the <b>discard data</b> setting and
                disabling it again.
              </>
            }
          >
            {generateResultsCountText("results", totalCount)}
          </TooltipWrapper>
        </>
      );
    }

    return <TableCount name="results" count={totalCount} />;
  }, [totalCount, isClipped]);

  const renderEmptyState = () => {
    if (searchQuery) {
      return (
        <EmptyState
          className={baseClass}
          header="No results match"
          info="Try a different search."
        />
      );
    }
    // Other empty states are handled in QueryDetailsPage.tsx and returned in lieu of QueryReport.tsx
    return (
      <EmptyState
        className={baseClass}
        header="Nothing to report yet"
        info={
          <>
            This report hasn&apos;t returned data yet.
            {canLiveQuery && (
              <>
                <br />
                Expecting to see results? Run a{" "}
                <CustomLink
                  url={PATHS.LIVE_REPORT(queryId)}
                  text="live report"
                />{" "}
                to troubleshoot.
              </>
            )}
          </>
        }
      />
    );
  };

  const renderTable = () => {
    return (
      <div className={`${baseClass}__results-table-container`}>
        <TableContainer
          columnConfigs={columnConfigs}
          data={flattenResults(results)}
          emptyComponent={renderEmptyState}
          defaultSortHeader={toTableSortHeader(sortHeader)}
          defaultSortDirection={sortDirection}
          defaultSearchQuery={searchQuery}
          pageIndex={pageIndex}
          pageSize={pageSize}
          disableNextPage={!queryReport?.meta?.has_next_results}
          totalCount={queryReport?.count}
          isLoading={isFetching}
          onQueryChange={onTableQueryChange}
          searchable
          inputPlaceHolder="Search results"
          showMarkAllPages={false}
          isAllPagesSelected={false}
          resultsTitle="results"
          customControl={renderTableButtons}
          renderCount={renderResultsCount}
          getRowId={(_row, index) => String(index)}
        />
      </div>
    );
  };

  return <div className={`${baseClass}__wrapper`}>{renderTable()}</div>;
};

export default QueryReport;
