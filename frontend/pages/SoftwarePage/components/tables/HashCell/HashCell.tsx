import React from "react";
import { flatMap } from "lodash";

import Button from "components/buttons/Button";
import CopyButton from "components/buttons/CopyButton";
import TextCell from "components/TableContainer/DataTable/TextCell";
import { IHostSoftware, ISoftwareInstallVersion } from "interfaces/software";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

const baseClass = "hash-cell";

interface IHashCellProps {
  installedVersion: ISoftwareInstallVersion[] | null;
  onClickMultipleHashes: (software: IHostSoftware) => void;
}

const HashCell = ({
  installedVersion,
  onClickMultipleHashes,
}: IHashCellProps) => {
  const renderHash = (hash: string) => {
    return (
      <>
        <span className={`${baseClass}__sha256`}>
          {hash.slice(0, 7)}&hellip;{" "}
        </span>
        <CopyButton copyText={hash} variant="compact" />
      </>
    );
  };

  const allSignatureInformation = flatMap(
    installedVersion,
    (v) => v.signature_information ?? []
  );

  const allHash = flatMap(
    allSignatureInformation,
    (sigInfo) => sigInfo.hash_sha256 ?? []
  );
  const uniqueHash = new Set(allHash);
  const uniqueHashCount = uniqueHash.size;

  // 0 hashes
  if (installedVersion === null || uniqueHashCount === 0) {
    return (
      <TextCell className={baseClass} value={DEFAULT_EMPTY_CELL_VALUE} grey />
    );
  }

  // 1 hash
  if (uniqueHashCount === 1) {
    const [onlyHash] = uniqueHash;
    return <div className={baseClass}>{renderHash(onlyHash)}</div>;
  }

  // 2 or more hashes
  return (
    <Button variant="link" onClick={onClickMultipleHashes}>
      {uniqueHashCount.toString()} hashes
    </Button>
  );
};

export default HashCell;
