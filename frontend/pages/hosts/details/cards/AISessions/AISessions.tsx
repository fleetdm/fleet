import classnames from "classnames";
import React, { useCallback } from "react";
import { Row } from "react-table";

import Card from "components/Card";
import CardHeader from "components/CardHeader";
import DataError from "components/DataError";
import EmptyState from "components/EmptyState";
import Spinner from "components/Spinner";
import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import { IHostAISession } from "interfaces/ai_sessions";
import { IListSort } from "interfaces/list_options";
import { IGetHostAISessionsResponse } from "services/entities/hosts";

import generateTableConfig from "./AISessionsTableConfig";

const baseClass = "ai-sessions-card";

interface IAISessionsCardProps {
  data?: IGetHostAISessionsResponse;
  isLoading: boolean;
  /** The sessions shown are for the previous page, sort, or host while the
   * new ones load. */
  isPreviousData: boolean;
  isError: boolean;
  page: number;
  pageSize: number;
  sortHeader: string;
  sortDirection: string;
  className?: string;
  onSelectSession: (session: IHostAISession) => void;
  onNextPage: () => void;
  onPreviousPage: () => void;
  onSortChange: ({ order_key, order_direction }: IListSort) => void;
}

const tableConfig = generateTableConfig();

const AISessionsCard = ({
  data,
  isLoading,
  isPreviousData,
  isError,
  page,
  pageSize,
  sortHeader,
  sortDirection,
  className,
  onSelectSession,
  onNextPage,
  onPreviousPage,
  onSortChange,
}: IAISessionsCardProps) => {
  const onQueryChange = useCallback(
    (newQuery: ITableQueryData) => {
      // Disabling the buttons instead would hide the pagination on the first
      // page while the next one loads.
      if (isPreviousData && newQuery.pageIndex !== page) {
        return undefined;
      }
      switch (true) {
        case newQuery.pageIndex > page:
          return onNextPage();
        case newQuery.pageIndex < page:
          return onPreviousPage();
        case newQuery.sortHeader !== sortHeader ||
          newQuery.sortDirection !== sortDirection:
          return onSortChange({
            order_key: newQuery.sortHeader,
            order_direction: newQuery.sortDirection || "asc",
          });
        default:
          return undefined; // noop
      }
    },
    [
      isPreviousData,
      onNextPage,
      onPreviousPage,
      onSortChange,
      page,
      sortDirection,
      sortHeader,
    ]
  );

  const renderContent = () => {
    if (isError) {
      return <DataError verticalPaddingSize="pad-large" />;
    }

    if (isLoading || !data) {
      return <Spinner />;
    }

    // Page 0 only, so an out-of-range page still shows its pagination.
    if (!data.ai_sessions.length && page === 0) {
      return (
        <EmptyState
          header="No AI sessions"
          info="AI coding agent sessions on this host will appear here."
        />
      );
    }

    return (
      <TableContainer
        className="ai-sessions-table"
        columnConfigs={tableConfig}
        data={data.ai_sessions}
        emptyComponent={() => null}
        isAllPagesSelected={false}
        showMarkAllPages={false}
        isLoading={isPreviousData}
        disableMultiRowSelect
        onSelectSingleRow={(row: Row) => {
          if (!isPreviousData) {
            // The modal doesn't take focus, so the clicked row would keep it
            // and show its focus ring once Escape closes the modal.
            (document.activeElement as HTMLElement | null)?.blur();
            onSelectSession(row.original as IHostAISession);
          }
        }}
        renderCount={() => <TableCount name="sessions" count={data.count} />}
        pageSize={pageSize}
        pageIndex={page}
        defaultSortHeader={sortHeader}
        defaultSortDirection={sortDirection}
        onQueryChange={onQueryChange}
        disableNextPage={!data.meta.has_next_results}
        manualSortBy
      />
    );
  };

  return (
    <Card className={classnames(baseClass, className)} paddingSize="xlarge">
      <CardHeader
        header={
          <span className={`${baseClass}__header`}>
            AI sessions
            <span className={`${baseClass}__header-period`}>Last 30 days</span>
          </span>
        }
      />
      {renderContent()}
    </Card>
  );
};

export default AISessionsCard;
