import React from "react";

import { IFileDetails } from "utilities/file/fileUtils";

import Button from "components/buttons/Button";
import { ISupportedGraphicNames } from "components/FileUploader/FileUploader";
import Graphic from "components/Graphic";
import Icon from "components/Icon";

interface IFileDetailsProps {
  graphicNames: ISupportedGraphicNames | ISupportedGraphicNames[];
  fileDetails: IFileDetails;
  canEdit: boolean;
  onFileSelect: (e: React.ChangeEvent<HTMLInputElement>) => void;
  accept?: string;
  progress?: number;
}

const baseClass = "file-details";

const FileDetails = ({
  graphicNames,
  fileDetails,
  canEdit,
  onFileSelect,
  accept,
  progress,
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
      {!progress && canEdit && (
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
      {!!progress && (
        <div className={`${baseClass}__progress-wrapper`}>
          <div className={`${baseClass}__progress-bar`}>
            <div
              className={`${baseClass}__progress-bar--uploaded`}
              style={{
                width: `${progress * 100}%`,
              }}
              title="upload progress bar"
            />
          </div>
          <div className={`${baseClass}__progress-text`}>
            {Math.round(progress * 100)}%
          </div>
        </div>
      )}
    </div>
  );
};

export default FileDetails;
