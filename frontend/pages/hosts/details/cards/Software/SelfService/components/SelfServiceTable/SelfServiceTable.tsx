import React from "react";
import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import EmptyState from "components/EmptyState";
import CustomLink from "components/CustomLink";
import { IDeviceSoftwareWithUiStatus } from "interfaces/software";
import { IGetDeviceSoftwareResponse } from "services/entities/device_user";

interface SelfServiceTableProps {
  baseClass: string;
  contactUrl: string;
  queryParams: {
    query: string;
    category_id?: number;
    order_key: string;
    order_direction: "asc" | "desc";
    page: number;
    per_page: number;
  };
  enhancedSoftware: IDeviceSoftwareWithUiStatus[];
  selfServiceData?: IGetDeviceSoftwareResponse;
  tableConfig: any;
  isFetching: boolean;
  onSortChange: (query: ITableQueryData) => void;
  onClientSidePaginationChange: (page: number) => void;
}

const SelfServiceTable = ({
  baseClass,
  contactUrl,
  queryParams,
  enhancedSoftware,
  selfServiceData,
  tableConfig,
  isFetching,
  onSortChange,
  onClientSidePaginationChange,
}: SelfServiceTableProps): JSX.Element => {
  const initialSortHeader = queryParams.order_key || "name";
  const initialSortDirection = queryParams.order_direction || "asc";
  const initialSortPage = queryParams.page || 0;
  // The table renders emptyComponent only when its post-filter data is empty.
  // If the user has a search query at that point, the search is what produced
  // the empty result; otherwise the category filter (or the initial dataset)
  // did. Derived here so the parent doesn't have to know about TableContainer's
  // internal search filter to compute this accurately.
  const isEmptySearch = !!queryParams.query;

  return (
    <div className={`${baseClass}__table`}>
      <TableContainer
        columnConfigs={tableConfig}
        data={enhancedSoftware}
        isLoading={isFetching}
        defaultSortHeader={initialSortHeader}
        defaultSortDirection={initialSortDirection}
        onQueryChange={onSortChange}
        pageIndex={initialSortPage}
        disableNextPage={selfServiceData?.meta.has_next_results === false}
        hideFooter={
          selfServiceData?.meta.has_next_results === false &&
          initialSortPage === 0
        }
        pageSize={9999}
        searchQuery={queryParams.query}
        searchQueryColumn="name"
        isClientSideFilter
        isClientSidePagination
        disableAutoResetPage
        onClientSidePaginationChange={onClientSidePaginationChange}
        emptyComponent={() =>
          isEmptySearch ? (
            <EmptyState
              header="No items match your search"
              info={
                <>
                  Not finding what you&apos;re looking for?{" "}
                  <CustomLink url={contactUrl} text="Reach out to IT" newTab />
                </>
              }
            />
          ) : (
            <EmptyState
              header="No items match the current search criteria"
              info="Expecting to see software? Check back later."
            />
          )
        }
        showMarkAllPages={false}
        isAllPagesSelected={false}
        disableTableHeader
        disableCount
      />
    </div>
  );
};

export default SelfServiceTable;
