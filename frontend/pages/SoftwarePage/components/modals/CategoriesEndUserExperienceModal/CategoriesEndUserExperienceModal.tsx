/**
 * Previews match preview in Edit Appearance modal for Edit Appearance modal
 *
 * Currently only shown from the edit UI, though wired through the Add UI
 * Users currently can set categories only when editing a current installer
 */

import React, { useContext } from "react";

import { Column } from "react-table";
import { AppContext } from "context/app";
import { IHeaderProps } from "interfaces/datatable_config";

import TableContainer from "components/TableContainer";
import SoftwareNameCell from "components/TableContainer/DataTable/SoftwareNameCell";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";

import SelfServicePreview from "../../cards/SelfServicePreview";
import SoftwareIcon from "../../icons/SoftwareIcon";

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
  iconUrl?: string | null;
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
        <SelfServicePreview
          isIosOrIpadosApp={isIosOrIpadosApp}
          contactUrl={config?.org_info.contact_url || ""}
          name={name}
          displayName={displayName || name}
          versionLabel={mobileVersion || "Version (unknown)"}
          renderIcon={() => (
            <SoftwareIcon
              name={name}
              source={source}
              url={iconUrl ?? undefined}
            />
          )}
          renderTable={() => (
            <BasicSoftwareTable
              name={name}
              displayName={displayName}
              source={source}
              iconUrl={iconUrl}
            />
          )}
        />
        <div className="modal-cta-wrap">
          <Button onClick={onCancel}>Done</Button>
        </div>
      </>
    </Modal>
  );
};

export default CategoriesEndUserExperienceModal;
