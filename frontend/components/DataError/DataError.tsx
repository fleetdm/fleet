import React from "react";
import classnames from "classnames";

import CustomLink from "components/CustomLink";
import Icon from "components/Icon";
import Graphic from "components/Graphic";
import { Padding } from "styles/var/padding";

const baseClass = "data-error";

interface IDataErrorProps {
  /** the description text displayed under the header */
  description?: string;
  /** Excludes the link that asks user to create an issue. Defaults to `false` */
  excludeIssueLink?: boolean;
  children?: React.ReactNode;
  /**
   * Sets the vertical padding for the component.
   * **Recommended values:**
   * - For card-level components, use "pad-large" `24px`.
   * - For page-level components, use "pad-xxxlarge"`80px`.
   * These values help maintain consistent spacing across the application.
   */
  verticalPaddingSize?: Padding;
  className?: string;
  /** Flag to use the updated DataError design */
  useNew?: boolean;
  /** Overrides something gone wrong line with description text to condense error onto one line */
  singleCustomLine?: boolean;
}

const DEFAULT_DESCRIPTION = "Refresh the page or log in again.";

const DataError = ({
  description = DEFAULT_DESCRIPTION,
  excludeIssueLink = false,
  children,
  verticalPaddingSize,
  className,
  useNew = false,
  singleCustomLine = false,
}: IDataErrorProps): JSX.Element => {
  const classes = classnames(baseClass, className);

  if (singleCustomLine) {
    return (
      <div className={classes}>
        <div
          className={`${baseClass}__inner ${
            verticalPaddingSize &&
            `${baseClass}__vertical-${verticalPaddingSize}`
          }`}
        >
          <div className={`${baseClass}__content`}>
            <span
              className={`${baseClass}__header ${baseClass}__header--single-line`}
            >
              <Icon name="error" />
              {description}
            </span>
          </div>
        </div>
      </div>
    );
  }

  if (useNew) {
    return (
      <div className={classes}>
        <div
          className={`${baseClass}__inner-new ${
            verticalPaddingSize &&
            `${baseClass}__vertical-${verticalPaddingSize}`
          }`}
        >
          <Graphic name="data-error" />
          <div className={`${baseClass}__header`}>
            Something&apos;s gone wrong.
          </div>
          {children || (
            <>
              <div className={`${baseClass}__description`}>
                Refresh to try again.
              </div>
              {!excludeIssueLink && (
                <div className={`${baseClass}__file-issue`}>
                  If this keeps happening please&nbsp;
                  <CustomLink
                    url="https://github.com/fleetdm/fleet/issues/new/choose"
                    text="file an issue"
                    newTab
                  />
                </div>
              )}
            </>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className={classes}>
      <div
        className={`${baseClass}__inner ${
          verticalPaddingSize && `${baseClass}__vertical-${verticalPaddingSize}`
        }`}
      >
        <div className={`${baseClass}__content`}>
          <span className={`${baseClass}__header`}>
            <Icon name="error" />
            Something&apos;s gone wrong.
          </span>

          <>
            {children || (
              <>
                {description && (
                  <span className={`${baseClass}__description`}>
                    {description}
                  </span>
                )}
                {!excludeIssueLink && (
                  <span className={`${baseClass}__file-issue`}>
                    If this keeps happening, please&nbsp;
                    <CustomLink
                      url="https://github.com/fleetdm/fleet/issues/new/choose"
                      text="file an issue"
                      newTab
                    />
                  </span>
                )}
              </>
            )}
          </>
        </div>
      </div>
    </div>
  );
};

export default DataError;
