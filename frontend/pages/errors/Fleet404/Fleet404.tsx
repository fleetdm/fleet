import React from "react";

import { SUPPORT_LINK } from "utilities/constants";

import Button from "components/buttons/Button";

// @ts-ignore
import illustration from "../../../../assets/images/404.png";

const baseClass = "fleet-404";

const Fleet404 = () => (
  <>
    <img className={`${baseClass}__illustration`} src={illustration} alt="" />
    <div className="error-page__details">
      <h1 className={`${baseClass}__title`}>
        404: We can&apos;t find that page!
      </h1>
      <p className={`${baseClass}__description`}>
        The page you are looking for has either moved, or doesn&apos;t exist.
      </p>
      <Button
        variant="inverse"
        onClick={() =>
          window.open(SUPPORT_LINK, "_blank", "noopener,noreferrer")
        }
      >
        Get help with Fleet
      </Button>
    </div>
  </>
);

export default Fleet404;
