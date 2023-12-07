import React from "react";

import { ISoftwareTitleVersion } from "interfaces/software";

import TextCell from "components/TableContainer/DataTable/TextCell";

interface IVersionCellProps {
  versions: ISoftwareTitleVersion[];
}

const VersionCell = ({ versions }: IVersionCellProps) => {
  const text =
    versions.length !== 1 ? `${versions.length} versions` : versions[0].version;
  return <TextCell value={text} greyed={versions.length !== 1} />;
};

export default VersionCell;
