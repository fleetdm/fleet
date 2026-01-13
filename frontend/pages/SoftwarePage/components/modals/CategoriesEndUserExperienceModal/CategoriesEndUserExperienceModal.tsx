/** Mobile preview modal uses a screenshot
 * Non-mobile now uses HTML/CSS instead for
 * maintainability as the self-selvice UI changes
 *
 * Currently only shown from the edit UI, though wired through the Add UI
 * Users currently can set categories only when editing a curent installer
 */

import React, { useContext } from "react";

import { Column } from "react-table";
import { noop } from "lodash";
import { AppContext } from "context/app";
import { IHeaderProps } from "interfaces/datatable_config";

import TableContainer from "components/TableContainer";
import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Card from "components/Card";
import SelfServiceHeader from "pages/hosts/details/cards/Software/SelfService/components/SelfServiceHeader";
import SearchField from "components/forms/fields/SearchField";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import CategoriesMenu from "pages/hosts/details/cards/Software/SelfService/components/CategoriesMenu";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import { CATEGORIES_NAV_ITEMS } from "pages/hosts/details/cards/Software/SelfService/helpers";

import SoftwareIcon from "../../icons/SoftwareIcon";
import PreviewSelfServiceMobileIcon from "../../../../../../assets/images/preview-self-service-mobile-icon.png";

const baseClass = "categories-end-user-experience-preview-modal";

interface ISoftwareRow {
  name: React.ReactNode;
}

type ITableHeaderProps = IHeaderProps<ISoftwareRow>;

const columns: Column<ISoftwareRow>[] = [
  {
    Header: (cellProps: ITableHeaderProps) => (
      <HeaderCell value="Name" isSortedDesc={cellProps.column.isSortedDesc} />
    ),
    accessor: "name",
    disableSortBy: true,
  },
];

type ExampleSoftware = {
  title: string;
};

const EXAMPLE_SOFTWARE_ROWS: ExampleSoftware[] = [
  { title: "1Password" },
  { title: "Adobe Acrobat Reader" },
  { title: "Box Drive" },
];

const getData = (
  name: string,
  displayName: string,
  iconUrl: string | null,
  source?: string,
  previewIcon?: JSX.Element
): ISoftwareRow[] => {
  const currentSoftwareRow: ISoftwareRow = {
    name: (
      <SoftwareNameCell
        name={name}
        display_name={displayName}
        source={source}
        iconUrl={iconUrl}
        pageContext="deviceUser"
        isSelfService
        previewIcon={previewIcon}
      />
    ),
  };

  // Filters out the current software from the example rows to avoid duplication
  const exampleSoftwareRows: ISoftwareRow[] = EXAMPLE_SOFTWARE_ROWS.filter(
    (item) => item.title !== name
  ).map((item) => ({
    name: (
      <SoftwareNameCell
        name={item.title}
        source="apps"
        pageContext="deviceUser"
        isSelfService
      />
    ),
  }));

  return [currentSoftwareRow, ...exampleSoftwareRows];
};

const EmptyState = () => <div>No software found</div>;

interface BasicSoftwareTableProps {
  name: string;
  displayName: string;
  source?: string;
  iconUrl?: string | null;
  /** Render a preview icon instead for edit icon preview */
  previewIcon?: JSX.Element;
}

export const BasicSoftwareTable = ({
  name,
  displayName,
  source,
  iconUrl = null,
  previewIcon,
}: BasicSoftwareTableProps) => {
  return (
    <TableContainer<ISoftwareRow>
      columnConfigs={columns}
      data={getData(name, displayName, iconUrl, source, previewIcon)}
      isLoading={false}
      emptyComponent={EmptyState}
      showMarkAllPages={false}
      isAllPagesSelected={false}
      searchable={false}
      disablePagination
      disableCount
      disableTableHeader
    />
  );
};

interface ICategoriesEndUserExperienceModal {
  onCancel: () => void;
  isIosOrIpadosApp?: boolean;
  name?: string;
  displayName?: string;
  iconUrl?: string;
  source?: string;
  mobileVersion?: string;
}

const CategoriesEndUserExperienceModal = ({
  onCancel,
  isIosOrIpadosApp = false,
  name = "Software name",
  displayName = "Software name",
  iconUrl,
  source,
  mobileVersion,
}: ICategoriesEndUserExperienceModal): JSX.Element => {
  const { config } = useContext(AppContext);
  return (
    <Modal title="End user experience" onExit={onCancel} className={baseClass}>
      <>
        <span>What end users see:</span>

        {isIosOrIpadosApp ? (
          <Card
            borderRadiusSize="medium"
            color="white"
            className={`${baseClass}__preview-card`}
            paddingSize="xlarge"
          >
            <div className={`${baseClass}__preview-img-container--mobile`}>
              <img
                className={`${baseClass}__preview-img--mobile`}
                src={PreviewSelfServiceMobileIcon}
                alt="Preview icon on Fleet Desktop > Self-service"
              />
            </div>
            <div className={`${baseClass}__self-service-preview--mobile`}>
              <SoftwareIcon name={name} source={source} url={iconUrl} />
              <div
                className={`${baseClass}__self-service-preview-name-version--mobile`}
              >
                <div
                  className={`${baseClass}__self-service-preview-name--mobile`}
                >
                  <TooltipTruncatedText value={displayName || name} />
                </div>
                <div
                  className={`${baseClass}__self-service-preview-version--mobile`}
                >
                  {mobileVersion || "Version (unknown)"}
                </div>
              </div>
            </div>
          </Card>
        ) : (
          <Card
            borderRadiusSize="medium"
            color="grey"
            className={`${baseClass}__preview-card`}
            paddingSize="xlarge"
          >
            <div className={`${baseClass}__disabled-overlay`} />
            <Card
              className={`${baseClass}__preview-card__self-service`}
              borderRadiusSize="xxlarge"
            >
              <SelfServiceHeader
                contactUrl={config?.org_info.contact_url || ""}
                variant="preview"
              />
              <SearchField
                placeholder="Search by name"
                onChange={noop}
                disabled
              />
              <div className={`${baseClass}__table`}>
                <CategoriesMenu
                  categories={CATEGORIES_NAV_ITEMS}
                  queryParams={{
                    query: "",
                    order_direction: "asc",
                    order_key: "name",
                    page: 0,
                    per_page: 100,
                  }}
                  readOnly
                />
                <BasicSoftwareTable
                  name={name}
                  displayName={displayName}
                  source={source}
                  iconUrl={iconUrl}
                />
              </div>
            </Card>
          </Card>
        )}
        <div className="modal-cta-wrap">
          <Button onClick={onCancel}>Done</Button>
        </div>
      </>
    </Modal>
  );
};

export default CategoriesEndUserExperienceModal;
