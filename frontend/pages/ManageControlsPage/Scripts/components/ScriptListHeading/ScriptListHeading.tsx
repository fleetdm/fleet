import React from "react";
import ReactTooltip from "react-tooltip";

import { COLORS } from "styles/var/colors";

const baseClass = "script-list-heading";

const ScriptListHeading = () => {
  return (
    <div className={baseClass}>
      <div className={`${baseClass}__heading-group`}>
        <span>Script</span>
      </div>
      <div
        className={`${baseClass}__heading-group ${baseClass}__actions-heading`}
      >
        <span>Actions</span>
      </div>

      <ReactTooltip
        type="dark"
        effect="solid"
        id="ran"
        backgroundColor={COLORS["tooltip-bg"]}
      >
        <span className={`${baseClass}__tooltip-text`}>
          Script ran and exited with status code 0.
        </span>
      </ReactTooltip>
      <ReactTooltip
        type="dark"
        effect="solid"
        id="pending"
        backgroundColor={COLORS["tooltip-bg"]}
      >
        <span className={`${baseClass}__tooltip-text`}>
          Script will run when the host comes online.
        </span>
      </ReactTooltip>
      <ReactTooltip
        type="dark"
        effect="solid"
        id="errors"
        backgroundColor={COLORS["tooltip-bg"]}
      >
        <span className={`${baseClass}__tooltip-text`}>
          Script ran and exited with a non-zero status code. Click on a host to
          view error(s).
        </span>
      </ReactTooltip>
    </div>
  );
};

export default ScriptListHeading;
