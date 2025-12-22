import React from "react";
import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import EmptyTable from "components/EmptyTable";
import EmptySoftwareTable from "pages/SoftwarePage/components/tables/EmptySoftwareTable";
import CustomLink from "components/CustomLink";
import { IDeviceSoftwareWithUiStatus } from "interfaces/software";
import { IGetDeviceSoftwareResponse } from "services/entities/device_user";
import { CATEGORIES_NAV_ITEMS } from "../../helpers";
import CategoriesMenu from "../CategoriesMenu/CategoriesMenu";

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
  isEmptySearch: boolean;
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
  isEmptySearch,
  onSortChange,
  onClientSidePaginationChange,
}: SelfServiceTableProps): JSX.Element => {
  const initialSortHeader = queryParams.order_key || "name";
  const initialSortDirection = queryParams.order_direction || "asc";
  const initialSortPage = queryParams.page || 0;

  return (
    <div className={`${baseClass}__table`}>
      <CategoriesMenu
        queryParams={queryParams}
        categories={CATEGORIES_NAV_ITEMS}
      />
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
            <EmptyTable
              graphicName="empty-search-question"
              header="No items match your search"
              info={
                <>
                  Not finding what you&apos;re looking for?{" "}
                  <CustomLink url={contactUrl} text="Reach out to IT" newTab />
                </>
              }
            />
          ) : (
            <EmptySoftwareTable />
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
