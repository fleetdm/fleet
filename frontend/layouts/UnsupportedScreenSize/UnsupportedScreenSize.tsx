import React from "react";

import smallScreenImage from "../../../assets/images/small-screen-160x80@2x.png";

const baseClass = "unsupported-screen-size";

const UnsupportedScreenSize = () => {
  return (
    <div className={baseClass}>
      <img src={smallScreenImage} alt="Unsupported screen size" />
      <div className={`${baseClass}__text`}>
        <h1>This screen size is not supported yet.</h1>
        <p>Please enlarge your browser or try again on a computer.</p>
      </div>
    </div>
  );
};

export default UnsupportedScreenSize;
