import classnames from "classnames";
import React from "react";

const baseClass = "end-user-experience-preview";

interface IEndUserExperiencePreviewProps {
  previewImage: string;
  altText?: string;
  children?: React.ReactNode;
  className?: string;
}

const EndUserExperiencePerview = ({
  previewImage,
  altText = "end user experience preview",
  children,
  className,
}: IEndUserExperiencePreviewProps) => {
  const classes = classnames(baseClass, className);

  return (
    <div className={classes}>
      <h3>End user experience</h3>
      <>{children}</>
      <img
        className={`${baseClass}__preview-img`}
        src={previewImage}
        alt={altText}
      />
    </div>
  );
};

export default EndUserExperiencePerview;
