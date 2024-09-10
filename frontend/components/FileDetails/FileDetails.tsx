import React from "react";

import { ISupportedGraphicNames } from "components/FileUploader/FileUploader";
import Graphic from "components/Graphic";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

interface IFileDetailsProps {
  graphicNames: ISupportedGraphicNames | ISupportedGraphicNames[];
  fileDetails: {
    name: string;
    platform?: string;
  };
  canEdit: boolean;
  onFileSelect: (e: React.ChangeEvent<HTMLInputElement>) => void;
  accept?: string;
}

const baseClass = "file-details";

const FileDetails = ({
  graphicNames,
  fileDetails,
  canEdit,
  onFileSelect,
  accept,
}: IFileDetailsProps) => {
  return (
    <div className={baseClass}>
      <div className={`${baseClass}__info`}>
        <Graphic
          name={
            typeof graphicNames === "string" ? graphicNames : graphicNames[0]
          }
        />
        <div className={`${baseClass}__content`}>
          <div className={`${baseClass}__name`}>{fileDetails.name}</div>
          {fileDetails.platform && (
            <div className={`${baseClass}__platform`}>
              {fileDetails.platform}
            </div>
          )}
        </div>
      </div>
      {canEdit && (
        <div className={`${baseClass}__edit`}>
          <Button className={`${baseClass}__edit-button`} variant="icon">
            <label htmlFor="edit-file">
              <Icon name="pencil" color="ui-fleet-black-75" />
            </label>
          </Button>
          <input
            accept={accept}
            id="edit-file"
            type="file"
            onChange={onFileSelect}
          />
        </div>
      )}
    </div>
  );
};

export default FileDetails;
