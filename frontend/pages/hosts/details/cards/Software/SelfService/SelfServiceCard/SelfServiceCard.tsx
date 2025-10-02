import React, { useCallback } from "react";
import { InjectedRouter } from "react-router";

import { getPathWithQueryParams } from "utilities/url";
import { SingleValue } from "react-select-5";
import { IDeviceSoftwareWithUiStatus } from "interfaces/software";
import { IGetDeviceSoftwareResponse } from "services/entities/device_user";

import Card from "components/Card";
import CardHeader from "components/CardHeader";
import Spinner from "components/Spinner";
import EmptyTable from "components/EmptyTable";
import EmptySoftwareTable from "pages/SoftwarePage/components/tables/EmptySoftwareTable";
import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";

import SearchField from "components/forms/fields/SearchField";
import DropdownWrapper, {
  CustomOptionType,
} from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import CustomLink from "components/CustomLink";

import CategoriesMenu from "./CategoriesMenu/CategoriesMenu";
import { filterSoftwareByCategory, CATEGORIES_NAV_ITEMS } from "./../helpers";

const baseClass = "software-self-service";

export interface SelfServiceQueryParams {
  query: string;
  category_id?: number;
  order_key: string;
  order_direction: "asc" | "desc";
  page: number;
  per_page: number;
}

export interface ISelfServiceCardProps {
  contactUrl: string;
  queryParams: SelfServiceQueryParams;
  enhancedSoftware: IDeviceSoftwareWithUiStatus[];
  selfServiceData?: IGetDeviceSoftwareResponse;
  tableConfig: any;
  isLoading: boolean;
  isError: boolean;
  isFetching: boolean;
  isEmpty: boolean;
  isEmptySearch: boolean;
  router: InjectedRouter;
  pathname: string;
}

const SelfServiceCard = ({
  contactUrl,
  queryParams,
  enhancedSoftware,
  selfServiceData,
  tableConfig,
  isLoading,
  isError,
  isFetching,
  isEmpty,
  isEmptySearch,
  router,
  pathname,
}: ISelfServiceCardProps) => {
  const initialSortHeader = queryParams.order_key || "name";
  const initialSortDirection = queryParams.order_direction || "asc";
  const initialSortPage = queryParams.page || 0;

  const onClientSidePaginationChange = useCallback(
    (page: number) => {
      router.push(
        getPathWithQueryParams(pathname, {
          query: queryParams.query,
          category_id: queryParams.category_id,
          order_key: initialSortHeader,
          order_direction: initialSortDirection,
          page,
        })
      );
    },
    [
      pathname,
      queryParams.query,
      queryParams.category_id,
      initialSortDirection,
      initialSortHeader,
      router,
    ]
  );

  const onSearchQueryChange = (value: string) => {
    router.push(
      getPathWithQueryParams(pathname, {
        query: value,
        category_id: queryParams.category_id,
        order_key: initialSortHeader,
        order_direction: initialSortDirection,
        page: 0, // Always reset to page 0 when searching
      })
    );
  };

  const onSortChange = ({ sortHeader, sortDirection }: ITableQueryData) => {
    router.push(
      getPathWithQueryParams(pathname, {
        ...queryParams,
        order_key: sortHeader,
        order_direction: sortDirection,
        query: queryParams.query !== undefined ? queryParams.query : undefined,
        category_id:
          queryParams.category_id !== undefined
            ? queryParams.category_id
            : undefined,
        page: 0, // Always reset to page 0 when sorting
      })
    );
  };

  const onCategoriesDropdownChange = (
    option: SingleValue<CustomOptionType>
  ) => {
    router.push(
      getPathWithQueryParams(pathname, {
        category_id: option?.value !== "undefined" ? option?.value : undefined,
        query: queryParams.query,
        order_key: initialSortHeader,
        order_direction: initialSortDirection,
        page: 0, // Always reset to page 0 when searching
      })
    );
  };

  const renderHeaderFilters = () => (
    <div className={`${baseClass}__header-filters`}>
      <SearchField
        placeholder="Search by name"
        onChange={onSearchQueryChange}
        defaultValue={queryParams.query}
      />
      <DropdownWrapper
        options={CATEGORIES_NAV_ITEMS.map((category) => ({
          ...category,
          value: String(category.id),
        }))}
        value={String(queryParams.category_id || 0)}
        onChange={onCategoriesDropdownChange}
        name="categories-dropdown"
        className={`${baseClass}__categories-dropdown`}
      />
    </div>
  );

  const renderCategoriesMenu = () => (
    <CategoriesMenu
      queryParams={queryParams}
      categories={CATEGORIES_NAV_ITEMS}
    />
  );

  if (isLoading) return <Spinner />;
  if (isError) return <EmptyTable header="Error loading software." />;

  // Empty state
  if ((isEmpty || !selfServiceData) && !isFetching) {
    return (
      <EmptyTable
        graphicName="empty-software"
        header="No self-service software available yet"
        info="Your organization didn’t add any self-service software."
      />
    );
  }

  return (
    <Card
      className={`${baseClass}__self-service-card`}
      borderRadiusSize="xxlarge"
      paddingSize="xlarge"
    >
      <CardHeader
        header="Self-service"
        subheader={
          <>
            Install organization-approved apps provided by your IT department.{" "}
            {contactUrl && (
              <span>
                If you need help,{" "}
                <CustomLink url={contactUrl} text="reach out to IT" newTab />
              </span>
            )}
          </>
        }
      />
      {renderHeaderFilters()}
      <div className={`${baseClass}__table`}>
        {renderCategoriesMenu()}
        <TableContainer
          columnConfigs={tableConfig}
          data={filterSoftwareByCategory(
            enhancedSoftware || [],
            queryParams.category_id
          )}
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
          pageSize={20}
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
                    <CustomLink
                      url={contactUrl}
                      text="Reach out to IT"
                      newTab
                    />
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
    </Card>
  );
};

export default SelfServiceCard;
