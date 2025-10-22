import React, { useCallback } from "react";
import { InjectedRouter } from "react-router";

import { getPathWithQueryParams } from "utilities/url";
import { SingleValue } from "react-select-5";
import { IDeviceSoftwareWithUiStatus } from "interfaces/software";
import { IGetDeviceSoftwareResponse } from "services/entities/device_user";

import Card from "components/Card";
import Spinner from "components/Spinner";
import EmptyTable from "components/EmptyTable";
import { ITableQueryData } from "components/TableContainer/TableContainer";

import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";

import SelfServiceTable from "../components/SelfServiceTable";
import SelfServiceHeader from "../components/SelfServiceHeader";
import SelfServiceFilters from "../components/SelfServiceFilters";
import SelfServiceTiles from "../components/SelfServiceTiles";
import { filterSoftwareByCategory } from "../helpers";

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
  tableConfig?: any;
  isLoading: boolean;
  isError: boolean;
  isFetching: boolean;
  isEmpty: boolean;
  isEmptySearch: boolean;
  router: InjectedRouter;
  pathname: string;
  isMobileView?: boolean;
  onClickInstallAction?: any;
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
  isMobileView,
  onClickInstallAction,
}: ISelfServiceCardProps) => {
  const initialSortHeader = queryParams.order_key || "name";
  const initialSortDirection = queryParams.order_direction || "asc";

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

  if (isLoading) return <Spinner includeContainer={false} />;
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

  const filteredSoftwareByCategory: IDeviceSoftwareWithUiStatus[] = filterSoftwareByCategory(
    enhancedSoftware || [],
    queryParams.category_id
  );

  // Search query filter required for mobile view only ( desktop view has filter built into TableContainer)
  const filteredSoftware = isMobileView
    ? filteredSoftwareByCategory.filter((software) => {
        const query = queryParams.query?.toLowerCase().trim() ?? "";
        if (!query) return true;
        return software.name.toLowerCase().includes(query);
      })
    : filteredSoftwareByCategory;

  if (isMobileView) {
    return (
      <>
        <SelfServiceHeader contactUrl={contactUrl} variant="mobile-header" />
        <div className={`${baseClass}__mobile-installers`}>
          <SelfServiceFilters
            query={queryParams.query}
            category_id={queryParams.category_id}
            onSearchQueryChange={onSearchQueryChange}
            onCategoriesDropdownChange={onCategoriesDropdownChange}
          />
          <SelfServiceTiles
            contactUrl={contactUrl}
            enhancedSoftware={filteredSoftware}
            isFetching={isFetching}
            isEmptySearch={
              enhancedSoftware.length > 0 && filteredSoftware.length === 0
            }
            onClickInstallAction={onClickInstallAction}
          />
        </div>
      </>
    );
  }
  return (
    <Card
      className={`${baseClass}__self-service-card`}
      borderRadiusSize="xxlarge"
      paddingSize="xlarge"
    >
      <SelfServiceHeader contactUrl={contactUrl} />
      <SelfServiceFilters
        query={queryParams.query}
        category_id={queryParams.category_id}
        onSearchQueryChange={onSearchQueryChange}
        onCategoriesDropdownChange={onCategoriesDropdownChange}
      />
      <SelfServiceTable
        baseClass={baseClass}
        contactUrl={contactUrl}
        queryParams={queryParams}
        enhancedSoftware={filteredSoftware}
        selfServiceData={selfServiceData}
        tableConfig={tableConfig}
        isFetching={isFetching}
        isEmptySearch={isEmptySearch}
        onSortChange={onSortChange}
        onClientSidePaginationChange={onClientSidePaginationChange}
      />
    </Card>
  );
};

export default SelfServiceCard;
