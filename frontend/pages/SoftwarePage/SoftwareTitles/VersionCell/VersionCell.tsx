import TextCell from "components/TableContainer/DataTable/TextCell";
import React from "react";

import { ISoftwareVersions } from "services/entities/software";

interface IVersionCellProps {
  versions: ISoftwareVersions[];
}

const VersionCell = ({ versions }: IVersionCellProps) => {
  const text =
    versions.length !== 1 ? `${versions.length} versions` : versions[0].version;
  return <TextCell value={text} greyed={versions.length !== 1} />;
};

export default VersionCell;
